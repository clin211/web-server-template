package options

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTLSConfigDisabled(t *testing.T) {
	cfg, err := NewTLSOptions().TLSConfig()
	if err != nil || cfg != nil {
		t.Fatalf("disabled TLSConfig = %v, %v, want nil, nil", cfg, err)
	}
}

func TestTLSConfigCertificateBundle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	var bundle []byte
	for i, name := range []string{"first root", "second root"} {
		pub, private, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		cert := &x509.Certificate{SerialNumber: big.NewInt(int64(i + 1)), Subject: pkix.Name{CommonName: name}, IsCA: true, BasicConstraintsValid: true, NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
		der, err := x509.CreateCertificate(rand.Reader, cert, cert, pub, private)
		if err != nil {
			t.Fatal(err)
		}
		bundle = append(bundle, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	if err := os.WriteFile(path, bundle, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := (&TLSOptions{UseTLS: true, CaCert: path}).TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RootCAs == nil {
		t.Fatal("TLS root pool is nil")
	}
	expected := x509.NewCertPool()
	if !expected.AppendCertsFromPEM(bundle) {
		t.Fatal("cannot parse test roots")
	}
	if !cfg.RootCAs.Equal(expected) {
		t.Fatal("TLS config did not load both root certificates")
	}
}
