package converter

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"software.sslmate.com/src/go-pkcs12"
)

const (
	// AnnotationPFXConvert is the annotation key to trigger PFX to PEM conversion
	AnnotationPFXConvert = "pfx-tls.kubernetes.io/convert"
	// AnnotationPFXPassword is the annotation key for PFX password
	AnnotationPFXPassword = "pfx-tls.kubernetes.io/password"
	// AnnotationPFXPasswordSecretName is the annotation key for secret containing PFX password
	AnnotationPFXPasswordSecretName = "pfx-tls.kubernetes.io/password-secret-name"
	// AnnotationPFXPasswordSecretKey is the annotation key for the key in the secret containing PFX password
	AnnotationPFXPasswordSecretKey = "pfx-tls.kubernetes.io/password-secret-key"
	// AnnotationPFXDataKey is the annotation key specifying which key in the secret contains PFX data
	AnnotationPFXDataKey = "pfx-tls.kubernetes.io/pfx-key"
	// AnnotationConverted is the annotation key to mark a secret as already converted
	AnnotationConverted = "pfx-tls.kubernetes.io/converted"
)

// PFXConverter handles conversion of PFX certificates to PEM format
type PFXConverter struct{}

// NewPFXConverter creates a new PFX converter
func NewPFXConverter() *PFXConverter {
	return &PFXConverter{}
}

// ConvertPFXToPEM converts PFX certificate data to PEM format
// Returns the certificate, private key, and CA certificates in PEM format
func (c *PFXConverter) ConvertPFXToPEM(pfxData []byte, password string) (certPEM, keyPEM, caPEM []byte, err error) {
	// Decode PFX
	privateKey, certificate, caCerts, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode PFX: %w", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate.Raw,
	})

	// Encode private key to PEM
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})

	// Encode CA certificates to PEM
	if len(caCerts) > 0 {
		var caBundle []byte
		for _, caCert := range caCerts {
			caBlock := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: caCert.Raw,
			})
			caBundle = append(caBundle, caBlock...)
		}
		caPEM = caBundle
	}

	return certPEM, keyPEM, caPEM, nil
}
