package config

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type RoutingAPIConfig struct {
	URI          string `yaml:"uri"`
	Port         uint16 `yaml:"port"`
	AuthDisabled bool   `yaml:"auth_disabled"`

	ClientCertificatePath string `yaml:"client_cert_path"`
	ClientPrivateKeyPath  string `yaml:"client_private_key_path"`
	CACertificatePath     string `yaml:"ca_cert_path"`
}

type OAuthConfig struct {
	TokenEndpoint     string `yaml:"token_endpoint"`
	Port              uint16 `yaml:"port"`
	SkipSSLValidation bool   `yaml:"skip_ssl_validation"`
	ClientName        string `yaml:"client_name"`
	ClientSecret      string `yaml:"client_secret"`
	CACerts           string `yaml:"ca_certs"`
}

type FrontendTLSConfig struct {
	Enabled bool `yaml:"enabled"`
	// https://www.haproxy.com/documentation/haproxy-configuration-manual/latest/#5.1-crt
	// https://www.haproxy.com/documentation/haproxy-configuration-manual/latest/#3.12-load
	CertificateDir string `yaml:"cert_path"`
}

type BackendTLSConfig struct {
	Enabled              bool   `yaml:"enabled"`
	CACertificatePath    string `yaml:"ca_cert_path"`
	ClientCertAndKeyPath string `yaml:"client_cert_and_key_path"`
}

type FrontendTLSJob struct {
	Name       string `yaml:"name"`
	CertChain  string `yaml:"cert_chain"`
	PrivateKey string `yaml:"private_key"`
}

type Config struct {
	OAuth                        OAuthConfig         `yaml:"oauth"`
	RoutingAPI                   RoutingAPIConfig    `yaml:"routing_api"`
	HaProxyPidFile               string              `yaml:"haproxy_pid_file"`
	IsolationSegments            []string            `yaml:"isolation_segments"`
	ReservedSystemComponentPorts []uint16            `yaml:"reserved_system_component_ports"`
	DrainWaitDuration            time.Duration       `yaml:"drain_wait"`
	BackendTLS                   BackendTLSConfig    `yaml:"backend_tls"`
	FrontendTLS                  []FrontendTLSConfig `yaml:"frontend_tls_pem"`
	FrontendTLSJob               []FrontendTLSJob    `yaml:"frontend_tls"`
}

const DrainWaitDefault = 20 * time.Second

func New(path string) (*Config, error) {
	c := &Config{}
	err := c.initConfigFromFile(path)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) FrontendTLSJobBasePath() string {
	if bp := os.Getenv("FRONTEND_TLS_BASE_PATH"); bp != "" {
		return bp
	}
	return "/var/vcap/jobs/tcp_router/config/certs/tcp-router/frontend"
}

func resolveGroupID(primary, fallback string) (int, error) {
	group, err := user.LookupGroup(primary)
	if err != nil {
		group, err = user.LookupGroup(fallback)
		if err != nil {
			return -1, err
		}
	}

	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		return -1, err
	}

	return gid, nil
}

func (c *Config) initConfigFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(b, &c); err != nil {
		return err
	}

	if c.HaProxyPidFile == "" {
		return errors.New("haproxy_pid_file is required")
	}

	if c.DrainWaitDuration < 0 {
		c.DrainWaitDuration = DrainWaitDefault
	}

	owner, err := user.Lookup("root")
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(owner.Uid)
	if err != nil {
		return err
	}

	gid, err := resolveGroupID("vcap", "root")
	if err != nil {
		return err
	}

	if len(c.FrontendTLSJob) > 0 {
		var outputs []FrontendTLSConfig
		basePath := c.FrontendTLSJobBasePath()
		for i, cert := range c.FrontendTLSJob {

			name := strings.TrimSpace(cert.Name)
			certChain := strings.TrimSpace(cert.CertChain)
			privateKey := strings.TrimSpace(cert.PrivateKey)

			if name == "" || certChain == "" || privateKey == "" {
				return fmt.Errorf("frontend_tls[%d] must include name, cert_chain, and private_key", i)
			}

			block, _ := pem.Decode([]byte(certChain))
			if block == nil {
				return errors.New("failed to parse PEM block")
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return err
			}

			hasSAN := certHasSAN(cert)
			if !hasSAN {
				return fmt.Errorf("frontend_tls[%d].cert_chain must include a subjectAltName extension", i)
			}

			dirPath := filepath.Join(basePath, name)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				if !(isReadOnlyFS(err) || isPermissionDenied(err)) {
					return err
				}
			}

			certFilePath := filepath.Join(dirPath, fmt.Sprintf("%s.pem", name))
			keyFilePath := filepath.Join(dirPath, fmt.Sprintf("%s.pem.key", name))

			if err := writeFile(certFilePath, []byte(certChain), 0640, uid, gid); err != nil {
				return err
			}

			if err := writeFile(keyFilePath, []byte(privateKey), 0600, uid, gid); err != nil {
				return err
			}

			outputs = append(outputs, FrontendTLSConfig{
				Enabled:        true,
				CertificateDir: dirPath,
			})
		}

		c.FrontendTLS = outputs

	}

	if c.BackendTLS.Enabled {
		if c.BackendTLS.CACertificatePath != "" {
			pemData, err := os.ReadFile(c.BackendTLS.CACertificatePath)
			if err != nil {
				return err
			}

			pemData = []byte(strings.TrimSpace(string(pemData)))
			if len(pemData) > 0 {
				var block *pem.Block
				block, _ = pem.Decode(pemData)
				if block == nil {
					return fmt.Errorf("Invalid PEM block found in file %q", c.BackendTLS.CACertificatePath)
				}
				if len(block.Headers) != 0 {
					return fmt.Errorf("Unexpected headers in PEM block in file %q: %v", c.BackendTLS.CACertificatePath, block.Headers)
				}
				if block.Type != "CERTIFICATE" {
					return fmt.Errorf("Unexpected PEM block type %q in file %q (wanted CERTIFICATE)", block.Type, c.BackendTLS.CACertificatePath)
				}
				_, err = x509.ParseCertificate(block.Bytes)
				if err != nil {
					return fmt.Errorf("failed to parse certificate in %q: %s", c.BackendTLS.CACertificatePath, err)
				}
			}
		} else {
			return fmt.Errorf("Backend TLS was enabled but no CA certificates were specified")
		}

		if c.BackendTLS.ClientCertAndKeyPath != "" {
			pemData, err := os.ReadFile(c.BackendTLS.ClientCertAndKeyPath)
			if err != nil {
				return err
			}

			pemData = []byte(strings.TrimSpace(string(pemData)))
			var certBlock *pem.Block
			certBlock, pemData = pem.Decode(pemData)
			if certBlock == nil {
				return fmt.Errorf("Invalid PEM CERTIFICATE found in file %q", c.BackendTLS.ClientCertAndKeyPath)
			}
			certPEM := bytes.NewBuffer([]byte{})
			err = pem.Encode(certPEM, certBlock)
			if err != nil {
				return fmt.Errorf("Could not encode cert as PEM data: %s", err)
			}

			pemData = []byte(strings.TrimSpace(string(pemData)))
			var keyBlock *pem.Block
			keyBlock, pemData = pem.Decode(pemData)
			if keyBlock == nil {
				return fmt.Errorf("Invalid PEM PRIVATE KEY found in file %q", c.BackendTLS.ClientCertAndKeyPath)
			}
			keyPEM := bytes.NewBuffer([]byte{})
			err = pem.Encode(keyPEM, keyBlock)
			if err != nil {
				return fmt.Errorf("Could not encode key as PEM data: %s", err)
			}

			if len(pemData) > 0 {
				return fmt.Errorf("Unexpected data at the end of %s", c.BackendTLS.ClientCertAndKeyPath)
			}

			_, err = tls.X509KeyPair(certPEM.Bytes(), keyPEM.Bytes())
			if err != nil {
				return fmt.Errorf("Unable to validate backend TLS client cert + key in file %q: %s", c.BackendTLS.ClientCertAndKeyPath, err)
			}
		}
	} else {
		c.BackendTLS = BackendTLSConfig{}
	}

	return nil
}

func certHasSAN(cert *x509.Certificate) bool {
	hasSANExtension := false
	for _, ext := range cert.Extensions {
		if ext.Id.String() == "2.5.29.17" {
			hasSANExtension = true
			break
		}
	}

	hasDNSEntries := len(cert.DNSNames) > 0

	return hasSANExtension || hasDNSEntries
}

// writeFile writes data to the given path with the specified file mode and ownership.
//
// this function is accessed from:
//  1. prestart errand which has the necessary privs and is responsible for creating the files
//  2. tcp_router_ctl which doesn't have the necessary privs but also invokes this function
//     and so can be safely skipped
func writeFile(path string, data []byte, mode os.FileMode, uid, gid int) error {
	if err := os.WriteFile(path, data, 0600); err != nil {
		if isReadOnlyFS(err) || isPermissionDenied(err) {
			return nil
		}
		return err
	}

	if err := os.Chown(path, uid, gid); err != nil {
		if isReadOnlyFS(err) || isPermissionDenied(err) {
			return nil
		}
		return err
	}

	if err := os.Chmod(path, mode); err != nil {
		if isReadOnlyFS(err) || isPermissionDenied(err) {
			return nil
		}
		return err
	}

	return nil
}

func isReadOnlyFS(err error) bool {
	return errors.Is(err, syscall.EROFS)
}

func isPermissionDenied(err error) bool {
	return errors.Is(err, syscall.EPERM) ||
		errors.Is(err, syscall.EACCES) ||
		errors.Is(err, os.ErrPermission)
}
