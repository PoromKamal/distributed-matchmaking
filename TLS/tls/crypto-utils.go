/*
Rolling your own crypto.
A set of cryptography utils for tls.
*/

package tls

import (
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// Returns a certificate
func readCertificateFromFile(filePath string) (*x509.Certificate, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse client certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func readPrivateKeyFromFile(filePath string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse client key PEM")
	}
	keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	key, ok := keyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("client key is not an RSA private key")
	}
	return key, nil
}

/*
Extracts the certificate from ConnectionEndHelloCertificatePayload.
We know that there is exactly 1 certificate.
The payload is expected to have the following format:

Header: 4bytes
Record Data: 4 bytes
certChainPayloadLen, 3 bytes
certPayloadLen, 3 bytes
cert, K bytes
*/
func readCertificateFromBuffer(data []byte) (*x509.Certificate, error) {
	data = data[4:]
	if len(data) < 6 {
		return nil, fmt.Errorf("data too short to contain certificate information")
	}
	certLen := int(data[3])<<16 | int(data[4])<<8 | int(data[5])
	certData := data[6 : 6+certLen]
	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return cert, nil
}

/*
Based on the TLS 1.2 PRF implementation guide.
P_hash(secret, seed) = HMAC_hash(secret, A(1) + seed) +

	HMAC_hash(secret, A(2) + seed) +
	HMAC_hash(secret, A(3) + seed) + ...
	where + indicates concatenation.
	A() is defined as:
	   A(0) = seed
	   A(i) = HMAC_hash(secret, A(i-1))

https://datatracker.ietf.org/doc/html/rfc5246#section-5
*/
func prf(secret, seed []byte, outputLength int) []byte {
	output := make([]byte, 0, outputLength)
	A := seed // A(0)
	for len(output) < outputLength {
		//A(i) = HMAC_hash(secret, A(i-1))
		hmacA := hmac.New(sha256.New, secret)
		hmacA.Write(A)
		A = hmacA.Sum(nil)

		//HMAC_hash(secret, A(i) + seed)
		hmacOutput := hmac.New(sha256.New, secret)
		hmacOutput.Write(A)
		hmacOutput.Write(seed)
		output = append(output, hmacOutput.Sum(nil)...)
	}

	// Truncate the output to the desired length.
	return output[:outputLength]
}

/*
Generate the MasterKey from
https://datatracker.ietf.org/doc/html/rfc5246#section-8.1
master_secret = PRF(pre_master_secret, "master secret",

	ClientHello.random + ServerHello.random)
	[0..47];
*/
func generateMasterSecret(clientRandom []byte, serverRandom []byte, preMasterSecret []byte) []byte {
	seed := append([]byte("master secret"), clientRandom...)
	seed = append(seed, serverRandom...)
	masterSecret := prf(preMasterSecret, seed, 48)
	return masterSecret
}
