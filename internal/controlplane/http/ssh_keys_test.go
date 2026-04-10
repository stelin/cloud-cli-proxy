package http

import (
	"strings"
	"testing"
)

func TestGenerateEd25519KeyPairProducesOpenSSHPrivateKey(t *testing.T) {
	publicKey, privateKey, err := generateEd25519KeyPair("test@example")
	if err != nil {
		t.Fatalf("generateEd25519KeyPair() error = %v", err)
	}

	if !strings.Contains(privateKey, "BEGIN OPENSSH PRIVATE KEY") {
		t.Fatalf("private key is not in OpenSSH format:\n%s", privateKey)
	}

	if err := validateSSHKeyPair(publicKey, privateKey); err != nil {
		t.Fatalf("validateSSHKeyPair() error = %v", err)
	}
}

func TestValidateSSHKeyPairRejectsMismatch(t *testing.T) {
	publicKey, _, err := generateEd25519KeyPair("pub-only")
	if err != nil {
		t.Fatalf("generate first ed25519 key pair: %v", err)
	}

	_, otherPrivateKey, err := generateEd25519KeyPair("priv-only")
	if err != nil {
		t.Fatalf("generate second ed25519 key pair: %v", err)
	}

	if err := validateSSHKeyPair(publicKey, otherPrivateKey); err == nil {
		t.Fatal("validateSSHKeyPair() expected mismatch error, got nil")
	}
}
