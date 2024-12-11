package tls

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
)

/*
On building the ServerHelloPayload:
Ref: https://datatracker.ietf.org/doc/html/rfc5246#section-7.4.1.2 - Page[40]
	>Payload must generate random at the time of connection. (Store security params)
	>For now, let every connection start new, so we set sessionId
	>Server advertises only 1 supported cipher: TLS_RSA_WITH_AES_128_CBC_SHA
	>No compression for this implementation so always advertise NULL OP for Compression
*/

func (c *SecurityParameters) ServerHelloPayload() []byte {
	serverVersion := []byte{byte(major), byte(minor)}
	random := make([]byte, 32)
	rand.Read(random)
	c.ServerRandom = random
	sessionId := []byte{0x00}
	cipherSuites := append(
		[]byte{0x00, 0x02},                 //Size
		TLS_RSA_WITH_AES_128_CBC_SHA[:]..., //Type
	)
	compressionMethod := []byte{0x01, 0x00}

	body := bytes.Join([][]byte{
		serverVersion,
		random,
		sessionId,
		cipherSuites,
		compressionMethod,
	}, nil)

	header := []byte{byte(ServerHello)}
	length := make([]byte, 3)
	binary.BigEndian.PutUint16(length[1:], uint16(len(body)))
	header = append(header, length...)
	return append(header, body...)
}

/******************************************************************/
/*Stage 2 Code Region::  Server Hello*/
/******************************************************************/

/* After Server Hello, the server shall send the following information:
1. Certificate
2. ServerKeyExchange  [Not Required]
3. CertificateRequest
4. ServerHelloDone

On: Certificate:
	For this implementation we only require sending the server certificate.
	We do not send any certificate chain of trust.
	The client will have a preloaded trusted certificate list.

	Outline:
		1. 3 bytes for certificate chain length, 0XX, XX = length of chain.
		2. 3 bytes for cert length followed by cert data.
		3. repeat 2 until certdata = length of chain.

On ServerKeyExchange:
	The ServerKeyExchange is not required for this implementation.
	We are to use DH_RSA consequently means ServerKeyExchange is forbidden.
	Ref: https://datatracker.ietf.org/doc/html/rfc5246#section-7.4.3

On CertificateRequest:
	We always request client certificate.
*/

func (c *SecurityParameters) ServerHelloCertificatePayload() ([]byte, error) { //Done
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

func (c *SecurityParameters) ServerHelloCertificateRequestPayload() ([]byte, error) {
	header := []byte{byte(CertificateRequest)}
	body := []byte{0x00, 0x00, 0x00}
	return append(header, body...), nil
}

func (c *SecurityParameters) ServerHelloDonePayload() []byte {
	header := []byte{byte(ServerHelloDone)}
	body := []byte{0x00, 0x00, 0x00}
	return append(header, body...)
}

func (c *SecurityParameters) ServerFinishedPayload(messages []byte) ([]byte, error) {
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

func (c *SecurityParameters) ServerChangeCipherSpecPayload() ([]byte, error) {
	return []byte{0x01}, nil
}
