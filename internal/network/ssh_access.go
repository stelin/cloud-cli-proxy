package network

import (
	"encoding/binary"
	"fmt"
	"net"
)

// SSHAccessInfo holds the derived SSH connection parameters
// for reaching a container through its management veth interface.
type SSHAccessInfo struct {
	Host string
	Port int
}

// DeriveManagementSSHAccess computes the container-side management IP
// and SSH port from a hostID, using the same /30 subnet derivation
// as InjectManagementVeth. This avoids duplicating the address
// calculation in downstream consumers.
func DeriveManagementSSHAccess(hostID string) SSHAccessInfo {
	idx := mgmtSubnetIndexFromID(hostID)
	block := idx / 128
	offset := (idx % 128) * 2

	containerAddr := fmt.Sprintf("10.99.%d.%d/30", block+1, offset+2)
	ip, _, _ := net.ParseCIDR(containerAddr)

	return SSHAccessInfo{
		Host: ip.String(),
		Port: 22,
	}
}

// mgmtSubnetIndexFromID derives a unique /30 subnet index from hostID.
// Mirrors the algorithm in namespace.go (mgmtSubnetIndex) so the two
// remain consistent without requiring the linux-only build tag.
func mgmtSubnetIndexFromID(hostID string) uint16 {
	b := []byte(hostID)
	if len(b) < 4 {
		padded := make([]byte, 4)
		copy(padded, b)
		b = padded
	}
	return binary.BigEndian.Uint16(b[:2]) ^ binary.BigEndian.Uint16(b[2:4])%16382
}
