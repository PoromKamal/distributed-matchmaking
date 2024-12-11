/*

	Ref: https://datatracker.ietf.org/doc/html/rfc5246
	Client                                            Server

	ClientHello                  -------->
													ServerHello
													Certificate*
											ServerKeyExchange*
											CertificateRequest*

								<--------      ServerHelloDone
	Certificate*
	ClientKeyExchange
	CertificateVerify*
	[ChangeCipherSpec]
	Finished                     -------->
											[ChangeCipherSpec]
								<--------             Finished
	Application Data             <------->     Application Data

*/

package tls

import (
	"crypto/cipher"
	"crypto/rsa"
	"crypto/x509"
	"net"
)

/* Handshake types for FSM*/
type ProtocolVersion uint16
type HandshakeType uint16
type ContentType uint8
type CipherSuite [2]byte
type ConnectionEnd int

const (
	ClientCE ConnectionEnd = 0
	ServerCE ConnectionEnd = 1
)

const (
	ClientHello        HandshakeType = 0x01
	ServerHello        HandshakeType = 0x02
	Certificate        HandshakeType = 0x0b
	ServerHelloDone    HandshakeType = 0x0e
	ClientKeyExchange  HandshakeType = 0x10
	Finished           HandshakeType = 0x14
	CertificateRequest HandshakeType = 0x0d
	CertificateVerify  HandshakeType = 0x0f
)

const (
	ChangeCipherSpec ContentType = 0x14
	Alert            ContentType = 0x15
	Handshake        ContentType = 0x16
	ApplicationData  ContentType = 0x17
)

const (
	major ProtocolVersion = 0x03
	minor ProtocolVersion = 0x03
)

/*
Ref: https://datatracker.ietf.org/doc/html/rfc5246#appendix-A.5
For simplicty some fields may not be used in this implementation.
*/
var (
	TLS_RSA_WITH_AES_128_CBC_SHA CipherSuite = CipherSuite{0x00, 0x2F}
)

/*
Ref: https://datatracker.ietf.org/doc/html/rfc5246#appendix-A.6
For simplicty some fields may not be used in this implementation.

> Non standard fields/modifcation:
  - Certificate: Holds the raw public certificate of the ConnectionEnd Type
  - RemoteCertificate: Holds the raw public certificate of the remote connector.
  - MasterSecret: Encoded as rsa.PrivateKey type rather than []byte.
  - CertificatePrivateKey *rsa.PrivateKey holds the private key of the Certificate.
*/
type SecurityParameters struct {
	ConnectionEnd         ConnectionEnd
	BulkCipherAlgorithm   string
	CipherType            uint16
	EncKeyLength          uint8
	BlockLength           uint8
	FixedIVLength         uint8
	RecordIVLength        uint8
	MACAlgorithm          string
	MACLength             uint8
	MACKeyLength          uint8
	Certificate           *x509.Certificate
	CertificatePrivateKey *rsa.PrivateKey
	RemoteCertificate     *x509.Certificate
	MasterSecret          []byte
	ClientRandom          []byte
	ServerRandom          []byte
}

type KeyMaterialBlock struct {
	ClientWriteMACKey []byte
	ServerWriteMACKey []byte
	ClientWriteKey    []byte
	ServerWriteKey    []byte
	ClientWriteIV     []byte
	ServerWriteIV     []byte

	EncryptCipher cipher.Block
	DecryptCipher cipher.Block
}

/*
https://datatracker.ietf.org/doc/html/rfc5246#section-6.2.3
TLSCipherText Format
*/
type TLSCiphertext struct {
	tlsType ContentType
	version []byte
	length  []byte
	data    []byte
}

/*
TLSConnection / connection.
*/
type TLSConnection struct {
	conn       net.Conn
	secParam   *SecurityParameters
	matBlock   *KeyMaterialBlock
	msgHistory []byte
	state      int
}
