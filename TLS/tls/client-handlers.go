package tls

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

/*
On building the ClientPayload:
Ref: https://datatracker.ietf.org/doc/html/rfc5246#section-7.4.1.2 - Page[40]

	>Payload must generate random at the time of connection. (Store security params)
	>For now, let every connection start new, so we set sessionId
	>Client advertises only 1 supported cipher: TLS_RSA_WITH_AES_128_CBC_SHA
	>No compression for this implementation so always advertise NULL OP for Compression
*/

func (c *SecurityParameters) ClientHelloPayload() []byte {
	clientVersion := []byte{byte(major), byte(minor)}
	random := make([]byte, 32)
	rand.Read(random)
	c.ClientRandom = random
	sessionId := []byte{0x00}
	cipherSuites := append(
		[]byte{0x00, 0x02},                 //Size
		TLS_RSA_WITH_AES_128_CBC_SHA[:]..., //Type
	)
	compressionMethod := []byte{0x01, 0x00}

	body := bytes.Join([][]byte{
		clientVersion,
		random,
		sessionId,
		cipherSuites,
		compressionMethod,
	}, nil)

	header := []byte{byte(ClientHello)}
	length := make([]byte, 3)
	binary.BigEndian.PutUint16(length[1:], uint16(len(body)))
	header = append(header, length...)
	return append(header, body...)
}

/******************************************************************/
/*Stage 2 Code Region::  Client Response to Hello*/
/******************************************************************/

/* After Server Hello End, the client server shall send the following information:
1. Certificate
2. ClientKeyExchange [Required]

Ref: https://datatracker.ietf.org/doc/html/rfc5246
*/

/*
Same implementation as the ServerHelloCertificatePayload.
For debugging we will keep this seperate for now.
*/
func (c *SecurityParameters) ClientHelloCertificatePayload() ([]byte, error) { //Done
	cert := c.Certificate.Raw
	certLen := len(cert)

	// Buffer for the certificate length. Followed this is the data.
	certPayloadLen := make([]byte, 3)
	certPayloadLen[0] = 0
	binary.BigEndian.PutUint16(certPayloadLen[1:], uint16(certLen))

	// Buffer for the certificate chain length payload. (include length metadata)
	certChainPayloadLen := make([]byte, 3)
	certChainPayloadLen[0] = 0
	binary.BigEndian.PutUint16(certChainPayloadLen[1:], uint16(certLen)+3)

	body := bytes.Join([][]byte{
		certChainPayloadLen,
		certPayloadLen,
		cert,
	}, nil)

	header := []byte{byte(Certificate)}
	length := make([]byte, 3)
	binary.BigEndian.PutUint16(length[1:], uint16(len(body)))
	header = append(header, length...)
	return append(header, body...), nil
}

/*
This function returns the payload to send, the premaster secret for storage, as well as the error if any.
*/
func (c *SecurityParameters) ClientHelloKeyExchangePayload() ([]byte, []byte, error) { // Done
	preMasterSecret := make([]byte, 48)
	rand.Read(preMasterSecret[2:])
	clientVersion := []byte{byte(major), byte(minor)}
	copy(preMasterSecret, clientVersion)
	peersPublicKey := c.RemoteCertificate.PublicKey.(*rsa.PublicKey)
	encrPreMasterSecret, err := rsa.EncryptPKCS1v15(rand.Reader, peersPublicKey, preMasterSecret)
	if err != nil {
		fmt.Println("Error encryting data..")
		return nil, nil, err
	}
	preMasterSecretlength := make([]byte, 2)
	binary.BigEndian.PutUint16(preMasterSecretlength, uint16(len(encrPreMasterSecret)))
	body := append(preMasterSecretlength, encrPreMasterSecret...)
	header := []byte{byte(ClientKeyExchange)}
	headerLength := make([]byte, 3)
	binary.BigEndian.PutUint16(headerLength[1:], uint16(len(body)))
	header = append(header, headerLength...)

	return append(header, body...), preMasterSecret, nil
}

func (c *SecurityParameters) ClientHelloCertificateVerifyPayload(messages []byte) ([]byte, error) { // Done
	fmt.Println("Client message history:")
	signatureHashAlgorithm := crypto.SHA256
	hashed := sha256.New()
	hashed.Write(messages)
	handshakeMessagesHash := hashed.Sum(nil)

	signature, err := rsa.SignPKCS1v15(
		rand.Reader,
		c.CertificatePrivateKey,
		signatureHashAlgorithm,
		handshakeMessagesHash,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %v", err)
	}

	signatureLength := make([]byte, 2)
	binary.BigEndian.PutUint16(signatureLength, uint16(len(signature)))
	certificateVerifyBody := append(signatureLength, signature...)

	handshakeHeader := []byte{byte(CertificateVerify)}
	handshakeMessageLength := make([]byte, 3)
	binary.BigEndian.PutUint16(handshakeMessageLength[1:], uint16(len(certificateVerifyBody)))
	handshakeHeader = append(handshakeHeader, handshakeMessageLength...)

	return append(handshakeHeader, certificateVerifyBody...), nil
}

func (c *SecurityParameters) ClientChangeCipherSpecPayload() ([]byte, error) {
	return []byte{0x01}, nil
}

func (c *SecurityParameters) ClientFinishedPayload(messages []byte) ([]byte, error) {
	hash := sha256.New()
	hash.Write(c.MasterSecret)
	hash.Write(messages)
	finishedHash := hash.Sum(nil)

	verifyData := finishedHash[:12]
	handshakeHeader := []byte{byte(Finished)}
	lengthHandshake := make([]byte, 3)
	binary.BigEndian.PutUint16(lengthHandshake[1:], uint16(len(verifyData)))
	handshakeHeader = append(handshakeHeader, lengthHandshake...)
	return append(handshakeHeader, verifyData...), nil
}
