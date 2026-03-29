//go:build linux

package network

import (
	"fmt"
	"runtime"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/vishvananda/netns"
)

const ipprotoTCP = 6

// ApplyFirewallRules installs nftables rules inside the container network
// namespace that enforce a default-deny policy. Only the WireGuard interface,
// loopback, and limited management veth traffic are permitted.
func ApplyFirewallRules(containerNS netns.NsHandle, wgIfIndex, loIfIndex, mgmtIfIndex int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	conn, err := nftables.New(nftables.WithNetNSFd(int(containerNS)))
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("open nftables conn: %v", err),
		}
	}

	applyIPv4Rules(conn, wgIfIndex, loIfIndex, mgmtIfIndex)
	applyIPv6Rules(conn, loIfIndex)

	if err := conn.Flush(); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("apply firewall rules: %v", err),
		}
	}

	return nil
}

func applyIPv4Rules(conn *nftables.Conn, wgIfIndex, loIfIndex, mgmtIfIndex int) {
	policyDrop := nftables.ChainPolicyDrop

	table := conn.AddTable(&nftables.Table{
		Family: nftables.TableFamilyIPv4,
		Name:   "filter",
	})

	outputChain := conn.AddChain(&nftables.Chain{
		Name:     "output",
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityFilter,
		Policy:   &policyDrop,
	})

	addOifAcceptRule(conn, table, outputChain, loIfIndex)
	addOifAcceptRule(conn, table, outputChain, wgIfIndex)
	addOifCtEstablishedRule(conn, table, outputChain, mgmtIfIndex)

	inputChain := conn.AddChain(&nftables.Chain{
		Name:     "input",
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
		Policy:   &policyDrop,
	})

	addIifAcceptRule(conn, table, inputChain, loIfIndex)
	addIifAcceptRule(conn, table, inputChain, wgIfIndex)
	addIifTCPDportAcceptRule(conn, table, inputChain, mgmtIfIndex, 22)
	addIifCtEstablishedRule(conn, table, inputChain, mgmtIfIndex)
}

func applyIPv6Rules(conn *nftables.Conn, loIfIndex int) {
	policyDrop := nftables.ChainPolicyDrop

	table6 := conn.AddTable(&nftables.Table{
		Family: nftables.TableFamilyIPv6,
		Name:   "filter6",
	})

	output6 := conn.AddChain(&nftables.Chain{
		Name:     "output6",
		Table:    table6,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityFilter,
		Policy:   &policyDrop,
	})

	addOifAcceptRule(conn, table6, output6, loIfIndex)
}

func addOifAcceptRule(conn *nftables.Conn, table *nftables.Table, chain *nftables.Chain, ifIndex int) {
	conn.AddRule(&nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyOIF, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.NativeEndian.PutUint32(uint32(ifIndex)),
			},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
	})
}

func addIifAcceptRule(conn *nftables.Conn, table *nftables.Table, chain *nftables.Chain, ifIndex int) {
	conn.AddRule(&nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyIIF, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.NativeEndian.PutUint32(uint32(ifIndex)),
			},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
	})
}

func addOifCtEstablishedRule(conn *nftables.Conn, table *nftables.Table, chain *nftables.Chain, ifIndex int) {
	conn.AddRule(&nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyOIF, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.NativeEndian.PutUint32(uint32(ifIndex)),
			},
			&expr.Ct{Register: 1, SourceRegister: false, Key: expr.CtKeySTATE},
			&expr.Bitwise{
				SourceRegister: 1,
				DestRegister:   1,
				Len:            4,
				Mask:           binaryutil.NativeEndian.PutUint32(expr.CtStateBitESTABLISHED | expr.CtStateBitRELATED),
				Xor:            binaryutil.NativeEndian.PutUint32(0),
			},
			&expr.Cmp{
				Op:       expr.CmpOpNeq,
				Register: 1,
				Data:     []byte{0, 0, 0, 0},
			},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
	})
}

func addIifCtEstablishedRule(conn *nftables.Conn, table *nftables.Table, chain *nftables.Chain, ifIndex int) {
	conn.AddRule(&nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyIIF, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.NativeEndian.PutUint32(uint32(ifIndex)),
			},
			&expr.Ct{Register: 1, SourceRegister: false, Key: expr.CtKeySTATE},
			&expr.Bitwise{
				SourceRegister: 1,
				DestRegister:   1,
				Len:            4,
				Mask:           binaryutil.NativeEndian.PutUint32(expr.CtStateBitESTABLISHED | expr.CtStateBitRELATED),
				Xor:            binaryutil.NativeEndian.PutUint32(0),
			},
			&expr.Cmp{
				Op:       expr.CmpOpNeq,
				Register: 1,
				Data:     []byte{0, 0, 0, 0},
			},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
	})
}

func addIifTCPDportAcceptRule(conn *nftables.Conn, table *nftables.Table, chain *nftables.Chain, ifIndex int, port uint16) {
	conn.AddRule(&nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyIIF, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.NativeEndian.PutUint32(uint32(ifIndex)),
			},
			&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{ipprotoTCP},
			},
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     binaryutil.BigEndian.PutUint16(port),
			},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
	})
}
