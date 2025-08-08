package httputil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/afero"
)

func BuildTLSConfig(host, certFile, keyFile string, fs afero.Fs) (*tls.Config, error) {
	_, err := fs.Stat(keyFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("could not stat private key file: %w", err)
		}
		err = generateSelfSignedCert(certFile, keyFile, host, true, fs)
		if err != nil {
			return nil, fmt.Errorf("could not generate self-signed certificate: %w", err)
		}
	}

	certificates, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{certificates},
	}, nil
}

func generateSelfSignedCert(certFile, keyFile, host string, isCA bool, fs afero.Fs) error {
	slog.Info("generating self-signed certificate", "host", host, "cert", certFile, "key", keyFile)
	keyUsage := x509.KeyUsageDigitalSignature

	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour) // 10 years

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Sat-System sp. z o.o."},
			Country:            []string{"PL"},
			Province:           []string{"Mazowieckie"},
			Locality:           []string{"Łomianki"},
			StreetAddress:      []string{"ul. Stanisława Staszica 47"},
			PostalCode:         []string{"05-092"},
			OrganizationalUnit: []string{"HUSAR Display"},
			CommonName:         host,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certOut, err := fs.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", certFile, err)
	}
	defer func() {
		err := certOut.Close()
		if err != nil {
			slog.Error("error closing %s: %w", certFile, err)
		}
	}()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write data to %s: %w", certFile, err)
	}

	keyOut, err := fs.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", keyFile, err)
	}
	defer func() {
		err := keyOut.Close()
		if err != nil {
			slog.Error("error closing %s: %w", keyFile, err)
		}
	}()
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("unable to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write data to %s: %w", keyFile, err)
	}
	return nil
}
