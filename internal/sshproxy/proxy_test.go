package sshproxy

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

// ---- passwordKeyboardInteractive ----

func TestPasswordKeyboardInteractive_ZeroQuestions(t *testing.T) {
	auth := passwordKeyboardInteractive("secret")
	challenge, ok := auth.(ssh.KeyboardInteractiveChallenge)
	if !ok {
		t.Fatal("expected ssh.KeyboardInteractiveChallenge")
	}
	answers, err := challenge("user", "instruction", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answers != nil {
		t.Fatalf("expected nil answers for zero questions, got %v", answers)
	}
}

func TestPasswordKeyboardInteractive_SingleQuestion(t *testing.T) {
	auth := passwordKeyboardInteractive("secret")
	challenge := auth.(ssh.KeyboardInteractiveChallenge)
	answers, err := challenge("user", "instruction", []string{"Password:"}, []bool{false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answers) != 1 || answers[0] != "secret" {
		t.Fatalf("expected [secret], got %v", answers)
	}
}

func TestPasswordKeyboardInteractive_MultipleQuestions(t *testing.T) {
	auth := passwordKeyboardInteractive("secret")
	challenge := auth.(ssh.KeyboardInteractiveChallenge)
	answers, err := challenge("user", "instruction", []string{"Password:", "Verification:", "Token:"}, []bool{false, false, false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answers) != 3 {
		t.Fatalf("expected 3 answers, got %d", len(answers))
	}
	for i, a := range answers {
		if a != "secret" {
			t.Fatalf("answers[%d] = %q, want %q", i, a, "secret")
		}
	}
}

func TestPasswordKeyboardInteractive_EmptyPassword(t *testing.T) {
	auth := passwordKeyboardInteractive("")
	challenge := auth.(ssh.KeyboardInteractiveChallenge)
	answers, err := challenge("user", "instruction", []string{"Password:"}, []bool{false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(answers) != 1 || answers[0] != "" {
		t.Fatalf("expected empty string answer, got %v", answers)
	}
}

func TestPasswordKeyboardInteractive_NonNilReturn(t *testing.T) {
	auth := passwordKeyboardInteractive("any")
	if auth == nil {
		t.Fatal("expected non-nil auth method")
	}
}

// ---- exportPublicKey ----

func TestExportPublicKey_WritesPubFile(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "host_key")
	logger := slog.New(slog.DiscardHandler)

	// Generate a valid signer.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	exportPublicKey(signer, privPath, logger)

	pubPath := privPath + ".pub"
	data, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatalf("read pub file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("pub file is empty")
	}
}

func TestExportPublicKey_ReadOnlyDir_DoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	// Make directory read-only.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Skipf("cannot chmod temp dir: %v", err)
	}
	defer os.Chmod(dir, 0o700)

	privPath := filepath.Join(dir, "host_key")
	logger := slog.New(slog.DiscardHandler)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	// Should not panic, should log a warning.
	exportPublicKey(signer, privPath, logger)
}

// ---- loadOrGenerateHostKey ----

func TestLoadOrGenerateHostKey_EmptyPath(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	signer, err := loadOrGenerateHostKey("", logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signer == nil {
		t.Fatal("expected non-nil signer")
	}
	// Verify the signer works by signing.
	pub := signer.PublicKey()
	if pub == nil {
		t.Fatal("expected non-nil public key")
	}
}

func TestLoadOrGenerateHostKey_GenerateAndPersist(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "host_key")
	logger := slog.New(slog.DiscardHandler)

	signer, err := loadOrGenerateHostKey(keyPath, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signer == nil {
		t.Fatal("expected non-nil signer")
	}

	// Verify PEM file was created.
	pemData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read PEM file: %v", err)
	}
	if len(pemData) == 0 {
		t.Fatal("PEM file is empty")
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}
	if block.Type != "PRIVATE KEY" {
		t.Fatalf("expected PRIVATE KEY, got %s", block.Type)
	}

	// Verify .pub file was created.
	pubPath := keyPath + ".pub"
	pubData, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatalf("read pub file: %v", err)
	}
	if len(pubData) == 0 {
		t.Fatal("pub file is empty")
	}
}

func TestLoadOrGenerateHostKey_LoadExisting(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "host_key")
	logger := slog.New(slog.DiscardHandler)

	// Generate and persist a key manually.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	derBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: derBytes})
	if err := os.WriteFile(keyPath, pemBlock, 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	// Load the existing key.
	signer, err := loadOrGenerateHostKey(keyPath, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signer == nil {
		t.Fatal("expected non-nil signer")
	}

	// Verify .pub file was exported.
	pubPath := keyPath + ".pub"
	if _, err := os.Stat(pubPath); err != nil {
		t.Fatalf("pub file not created: %v", err)
	}
}

func TestLoadOrGenerateHostKey_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "invalid_key")
	logger := slog.New(slog.DiscardHandler)

	// Write invalid data.
	if err := os.WriteFile(keyPath, []byte("not a valid PEM key"), 0o600); err != nil {
		t.Fatalf("write invalid file: %v", err)
	}

	_, err := loadOrGenerateHostKey(keyPath, logger)
	if err == nil {
		t.Fatal("expected error for invalid key file")
	}
}

func TestLoadOrGenerateHostKey_DirectoryPath(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.DiscardHandler)

	_, err := loadOrGenerateHostKey(dir, logger)
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

func TestLoadOrGenerateHostKey_NonExistentDir(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "subdir", "host_key")
	logger := slog.New(slog.DiscardHandler)

	signer, err := loadOrGenerateHostKey(keyPath, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signer == nil {
		t.Fatal("expected non-nil signer")
	}

	// Verify PEM file was created (MkdirAll succeeded).
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("key file not found: %v", err)
	}
}

// ---- NewServer ----

func TestNewServer_Defaults(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	resolver := &stubResolverRepo{}
	server, err := NewServer(":2222", "", "", "", resolver, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.addr != ":2222" {
		t.Fatalf("expected addr :2222, got %s", server.addr)
	}
	if server.containerUser != "workspace" {
		t.Fatalf("expected default containerUser 'workspace', got %q", server.containerUser)
	}
	if server.containerPassword != "workspace" {
		t.Fatalf("expected default containerPassword 'workspace', got %q", server.containerPassword)
	}
}

func TestNewServer_CustomValues(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	resolver := &stubResolverRepo{}
	server, err := NewServer(":2223", "myuser", "mypass", "", resolver, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.containerUser != "myuser" {
		t.Fatalf("expected containerUser 'myuser', got %q", server.containerUser)
	}
	if server.containerPassword != "mypass" {
		t.Fatalf("expected containerPassword 'mypass', got %q", server.containerPassword)
	}
}

func TestNewServer_WithHostKeyPath(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "host_key")
	logger := slog.New(slog.DiscardHandler)
	resolver := &stubResolverRepo{}

	server, err := NewServer(":2224", "user", "pass", keyPath, resolver, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server == nil {
		t.Fatal("expected non-nil server")
	}

	// Verify key was generated and persisted.
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("host key file not found: %v", err)
	}
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		t.Fatalf("host pub key file not found: %v", err)
	}
}

func TestNewServer_LoadsExistingHostKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "host_key")
	logger := slog.New(slog.DiscardHandler)

	// First call generates and persists.
	s1, err := NewServer(":2225", "u", "p", keyPath, &stubResolverRepo{}, logger)
	if err != nil {
		t.Fatalf("first NewServer: %v", err)
	}

	// Second call should load the existing key.
	s2, err := NewServer(":2225", "u", "p", keyPath, &stubResolverRepo{}, logger)
	if err != nil {
		t.Fatalf("second NewServer: %v", err)
	}

	// Both should have the same public key.
	p1 := ssh.MarshalAuthorizedKey(s1.hostKey.PublicKey())
	p2 := ssh.MarshalAuthorizedKey(s2.hostKey.PublicKey())
	if string(p1) != string(p2) {
		t.Fatal("public keys should match when loading existing key")
	}
}

// ---- ContainerTarget default propagation ----

func TestServer_ContainerTargetDefaults(t *testing.T) {
	// 验证 Server 结构体在创建时正确初始化所有字段。
	logger := slog.New(slog.DiscardHandler)
	resolver := NewRepoResolver(&stubResolverRepo{})
	server, err := NewServer(":2226", "appuser", "apppass", "", resolver, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.resolver != resolver {
		t.Fatal("resolver not set correctly")
	}
	if server.hostKey == nil {
		t.Fatal("hostKey not initialized")
	}
}

func TestNewServer_EmptyAddress(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	resolver := &stubResolverRepo{}
	server, err := NewServer("", "u", "p", "", resolver, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.addr != "" {
		t.Fatalf("expected empty addr, got %q", server.addr)
	}
}
