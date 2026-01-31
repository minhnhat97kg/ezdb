package db

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHConfig holds SSH connection details
type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	KeyPath  string
	UseAgent bool
}

// SSHTunnel represents an active SSH connection that can dial
type SSHTunnel struct {
	client *ssh.Client
}

// NewSSHTunnel establishes an SSH connection
func NewSSHTunnel(config *SSHConfig) (*SSHTunnel, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("SSH host is required")
	}

	authMethods := []ssh.AuthMethod{}

	// 1. Private Key File (Prioritize explicit key)
	if config.KeyPath != "" {
		keyPath := config.KeyPath
		if len(keyPath) > 1 && keyPath[:2] == "~/" {
			home, err := os.UserHomeDir()
			if err == nil {
				keyPath = filepath.Join(home, keyPath[2:])
			}
		}

		key, err := os.ReadFile(keyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				// Try with passphrase if password is provided
				if config.Password != "" {
					signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(config.Password))
				}
			}

			if err == nil {
				log.Printf("SSH: Successfully loaded private key. Type: %s", signer.PublicKey().Type())
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			} else {
				log.Printf("SSH: Failed to create signer from key: %v", err)
			}
		} else {
			log.Printf("SSH: Failed to read private key file %s (expanded from %s): %v", keyPath, config.KeyPath, err)
		}
	} else {
		log.Printf("SSH: No private key path provided")
	}

	// 2. SSH Agent
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			agentClient := agent.NewClient(conn)
			authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
			log.Printf("SSH: Added Agent auth method")
		} else {
			log.Printf("SSH: Failed to dial SSH_AUTH_SOCK: %v", err)
		}
	} else {
		log.Printf("SSH: SSH_AUTH_SOCK not set")
	}

	// 3. Password
	if config.Password != "" {
		log.Printf("SSH: Adding password authentication")
		authMethods = append(authMethods, ssh.Password(config.Password))

		// 4. Keyboard Interactive (sometimes required instead of Password)
		log.Printf("SSH: Adding keyboard-interactive authentication")
		authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = config.Password
			}
			return answers, nil
		}))
	} else {
		log.Printf("SSH: No password provided")
	}

	log.Printf("SSH: Total auth methods configured: %d (Agent: %v, Key: %v)", len(authMethods), os.Getenv("SSH_AUTH_SOCK") != "", config.KeyPath != "")

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no valid SSH authentication methods found")
	}

	cliConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For simplicity, in real app should verify
		Timeout:         0,
		// Explicitly enable common/legacy algorithms to ensure compatibility
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoRSASHA512,
			ssh.KeyAlgoRSASHA256,
			ssh.KeyAlgoRSA,
			ssh.KeyAlgoDSA,
			ssh.KeyAlgoECDSA256,
			ssh.KeyAlgoECDSA384,
			ssh.KeyAlgoECDSA521,
		},
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("SSH: Dialing %s with user %s", address, config.User)
	client, err := ssh.Dial("tcp", address, cliConfig)
	if err != nil {
		log.Printf("SSH: Dial failed: %v", err)
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}
	log.Printf("SSH: Connected successfully")

	return &SSHTunnel{client: client}, nil
}

// Dial connects to a remote address through the tunnel
func (t *SSHTunnel) Dial(network, addr string) (net.Conn, error) {
	return t.client.Dial(network, addr)
}

// DialContext connects to a remote address through the tunnel with context support
func (t *SSHTunnel) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		conn, err := t.client.Dial(network, addr)
		ch <- result{conn, err}
	}()

	select {
	case <-ctx.Done():
		// If the context is cancelled, we return an error.
		// Note: The goroutine above might still be dialing and will leak if it succeeds,
		// but since we return from the driver's Connect, the app can recover.
		return nil, ctx.Err()
	case res := <-ch:
		return res.conn, res.err
	}
}

// Close closes the SSH connection
func (t *SSHTunnel) Close() error {
	return t.client.Close()
}
