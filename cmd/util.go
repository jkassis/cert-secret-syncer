package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// splitPEMCertificates takes a byte array containing one or more PEM-encoded certificates
// and returns a slice of byte slices, each representing an individual certificate.
func splitPEMCertificates(pemBytes []byte) ([][]byte, error) {
	var certificates [][]byte
	var block *pem.Block

	for {
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, errors.New("failed to decode PEM block containing a certificate")
		}

		_, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %v", err)
		}

		// Re-encode the single certificate into PEM format and add it to the slice
		certPEM := pem.EncodeToMemory(block)
		certificates = append(certificates, certPEM)
	}

	if len(certificates) == 0 {
		return nil, errors.New("no certificates found")
	}

	return certificates, nil
}

func appendByteSlices(byteSlices [][]byte) []byte {
	var result []byte
	for _, b := range byteSlices {
		result = append(result, b...)
	}
	return result
}
