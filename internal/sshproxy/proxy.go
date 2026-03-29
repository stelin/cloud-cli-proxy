package sshproxy

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ContainerResolver maps an SSH login (short_id + entry_password)
// to the target container's SSH address (host:port).
type ContainerResolver interface {
	ResolveContainer(ctx context.Context, shortID, password string) (targetAddr string, err error)
}

type Server struct {
	addr     string
	logger   *slog.Logger
	resolver ContainerResolver
	hostKey  ssh.Signer

	containerUser     string
	containerPassword string
}

func NewServer(addr, containerUser, containerPassword string, resolver ContainerResolver, logger *slog.Logger) (*Server, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate host key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}

	if containerUser == "" {
		containerUser = "workspace"
	}
	if containerPassword == "" {
		containerPassword = "workspace"
		logger.Warn("SSH proxy using default container password — set CONTAINER_SSH_PASSWORD in production")
	}

	return &Server{
		addr:              addr,
		logger:            logger,
		resolver:          resolver,
		hostKey:           signer,
		containerUser:     containerUser,
		containerPassword: containerPassword,
	}, nil
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	config := &ssh.ServerConfig{
		MaxAuthTries:  3,
		ServerVersion: "SSH-2.0-CloudCLIProxy",
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			authCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			targetAddr, err := s.resolver.ResolveContainer(authCtx, conn.User(), string(password))
			if err != nil {
				s.logger.Debug("SSH auth failed", "user", conn.User(), "remote", conn.RemoteAddr(), "reason", err)
				return nil, fmt.Errorf("auth failed")
			}
			return &ssh.Permissions{
				Extensions: map[string]string{"target_addr": targetAddr},
			}, nil
		},
	}
	config.AddHostKey(s.hostKey)

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}
	defer listener.Close()
	s.logger.Info("SSH proxy listening", "addr", s.addr)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			s.logger.Error("accept failed", "error", err)
			continue
		}
		go s.handleConnection(conn, config)
	}
}

func (s *Server) handleConnection(netConn net.Conn, config *ssh.ServerConfig) {
	defer netConn.Close()

	sshConn, chans, globalReqs, err := ssh.NewServerConn(netConn, config)
	if err != nil {
		s.logger.Debug("SSH handshake failed", "remote", netConn.RemoteAddr(), "error", err)
		return
	}
	defer sshConn.Close()

	targetAddr := sshConn.Permissions.Extensions["target_addr"]
	s.logger.Info("SSH proxy session", "user", sshConn.User(), "target", targetAddr, "remote", netConn.RemoteAddr())

	go ssh.DiscardRequests(globalReqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "only session channels supported")
			continue
		}
		go s.handleChannel(newChan, targetAddr)
	}
}

func (s *Server) handleChannel(newChan ssh.NewChannel, targetAddr string) {
	clientChan, clientReqs, err := newChan.Accept()
	if err != nil {
		s.logger.Error("accept channel failed", "error", err)
		return
	}
	defer clientChan.Close()

	targetConfig := &ssh.ClientConfig{
		User:            s.containerUser,
		Auth:            []ssh.AuthMethod{ssh.Password(s.containerPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	targetClient, err := ssh.Dial("tcp", targetAddr, targetConfig)
	if err != nil {
		s.logger.Error("dial container SSH failed", "addr", targetAddr, "error", err)
		fmt.Fprintf(clientChan.Stderr(), "Failed to connect to container: connection refused\r\n")
		return
	}
	defer targetClient.Close()

	targetChan, targetReqs, err := targetClient.OpenChannel("session", nil)
	if err != nil {
		s.logger.Error("open target session failed", "addr", targetAddr, "error", err)
		fmt.Fprintf(clientChan.Stderr(), "Failed to open session on container\r\n")
		return
	}
	defer targetChan.Close()

	// Forward requests from client to target (pty-req, shell, exec, env, window-change).
	go func() {
		for req := range clientReqs {
			ok, err := targetChan.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			if req.WantReply {
				req.Reply(ok, nil)
			}
		}
	}()

	// Forward requests from target to client (exit-status, exit-signal).
	go func() {
		for req := range targetReqs {
			ok, err := clientChan.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			if req.WantReply {
				req.Reply(ok, nil)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(targetChan, clientChan)
		targetChan.CloseWrite()
	}()

	go func() {
		defer wg.Done()
		io.Copy(clientChan, targetChan)
		clientChan.CloseWrite()
	}()

	go io.Copy(clientChan.Stderr(), targetChan.Stderr())

	wg.Wait()
	s.logger.Debug("SSH proxy channel closed", "target", targetAddr)
}
