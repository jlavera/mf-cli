package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AppSupportDir is the base directory for mf's shared state. It lives outside
// any user home directory because the DNS and proxy daemons run as root (via
// LaunchDaemons) while `mf up`/`down` run as the logged-in user — both need to
// agree on a single path regardless of $HOME or sudo. Exposed as a variable so
// tests can point it at a temp directory.
var AppSupportDir = "/Library/Application Support/mf"

// CADir returns the directory where CA files are stored.
func CADir() string {
	return filepath.Join(AppSupportDir, "ca")
}

// CertCacheDir returns the directory for cached per-hostname certs.
func CertCacheDir() string {
	return filepath.Join(AppSupportDir, "certs")
}

// RootCertPath returns the path to the root CA certificate file.
func RootCertPath() string {
	return filepath.Join(CADir(), "mf-rootCA.pem")
}

// rootKeyPath returns the path to the root CA private key file.
func rootKeyPath() string {
	return filepath.Join(CADir(), "mf-rootCA-key.pem")
}

// CAExists reports whether a root CA has already been generated.
func CAExists() bool {
	_, err := os.Stat(RootCertPath())
	return err == nil
}

// CA manages a local root certificate authority for issuing TLS certs.
type CA struct {
	RootCert *x509.Certificate
	RootKey  *ecdsa.PrivateKey
	CertDir  string

	mu    sync.Mutex
	cache map[string]*tls.Certificate
}

// GenerateCA creates a new root CA cert+key and writes them to disk.
func GenerateCA() (*CA, error) {
	// Create the shared base dir world-traversable (0755) so non-root users can
	// reach sibling dirs like routes/. The CA subdir below stays 0700 to keep
	// the private key root-only.
	if err := os.MkdirAll(AppSupportDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create %s: %w", AppSupportDir, err)
	}
	os.Chmod(AppSupportDir, 0755)

	caDir := CADir()
	if err := os.MkdirAll(caDir, 0700); err != nil {
		return nil, fmt.Errorf("could not create CA directory: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not generate CA key: %w", err)
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"mf-cli local CA"},
			CommonName:   "mf-cli Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("could not create CA certificate: %w", err)
	}

	certFile, err := os.Create(RootCertPath())
	if err != nil {
		return nil, err
	}
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyFile, err := os.OpenFile(rootKeyPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	keyFile.Close()

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	certDir := CertCacheDir()
	os.MkdirAll(certDir, 0700)

	return &CA{
		RootCert: cert,
		RootKey:  key,
		CertDir:  certDir,
		cache:    make(map[string]*tls.Certificate),
	}, nil
}

// LoadCA reads an existing root CA from disk.
func LoadCA() (*CA, error) {
	certPEM, err := os.ReadFile(RootCertPath())
	if err != nil {
		return nil, fmt.Errorf("CA not found — run 'mf proxy install' first: %w", err)
	}
	keyPEM, err := os.ReadFile(rootKeyPath())
	if err != nil {
		return nil, fmt.Errorf("CA key not found: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("could not decode CA certificate")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("could not decode CA key")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return &CA{
		RootCert: cert,
		RootKey:  key,
		CertDir:  CertCacheDir(),
		cache:    make(map[string]*tls.Certificate),
	}, nil
}

// GetCertificate returns a TLS certificate for the given hostname, generating
// and caching it on first request. Suitable for use as tls.Config.GetCertificate.
func (ca *CA) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	hostname := hello.ServerName

	ca.mu.Lock()
	if cert, ok := ca.cache[hostname]; ok {
		ca.mu.Unlock()
		return cert, nil
	}
	ca.mu.Unlock()

	cert, err := ca.issueCert(hostname)
	if err != nil {
		return nil, err
	}

	ca.mu.Lock()
	ca.cache[hostname] = cert
	ca.mu.Unlock()

	return cert, nil
}

func (ca *CA) issueCert(hostname string) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: hostname,
		},
		DNSNames:  []string{hostname},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.RootCert, &key.PublicKey, ca.RootKey)
	if err != nil {
		return nil, fmt.Errorf("could not issue certificate for %s: %w", hostname, err)
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	return tlsCert, nil
}

// RemoveCA deletes all CA and cached cert files.
func RemoveCA() error {
	os.RemoveAll(CertCacheDir())
	return os.RemoveAll(CADir())
}
