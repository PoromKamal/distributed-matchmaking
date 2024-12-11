package tls

import (
	"crypto"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
)

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

// HandshakeServerController orchestrates the server-side TLS handshake
func (c *TLSConnection) HandshakeServerController() error {
	log.Println("Server: Starting Handshake")

	// Receive ClientHello
	if err := c.receiveClientHello(); err != nil {
		return err
	}

	// Send Server Messages
	if err := c.sendServerMessages(); err != nil {
		return err
	}

	// Receive Client Certificate
	if err := c.receiveClientCertificate(); err != nil {
		return err
	}

	// Receive ClientKeyExchange
	preMasterSecret, err := c.receiveClientKeyExchange()
	if err != nil {
		return err
	}

	// Receive CertificateVerify
	if err := c.receiveCertificateVerify(preMasterSecret); err != nil {
		return err
	}

	// Receive ChangeCipherSpec and Finished from Client
	if err := c.receiveClientChangeCipherSpecAndFinished(); err != nil {
		return err
	}

	// Send ChangeCipherSpec and Finished to Client
	if err := c.sendChangeCipherSpecAndFinished(); err != nil {
		return err
	}

	log.Println("Server: Handshake completed successfully")
	return nil
}

// receiveClientHello handles receiving and processing ClientHello
func (c *TLSConnection) receiveClientHello() error {
	log.Println("Server: Waiting to receive ClientHello")
	ct, err := c.readTLSContent()
	if err != nil {
		return err
	}

	if ct.tlsType != Handshake {
		return errors.New("expected Handshake content type for ClientHello")
	}

	handshakeType := HandshakeType(ct.data[0])
	if handshakeType != ClientHello {
		return errors.New("expected ClientHello handshake type")
	}

	c.msgHistory = append(c.msgHistory, ct.data...)
	log.Println("Server: Received ClientHello")
	c.secParam.onClientHello(ct.data)
	return nil
}

// sendServerMessages handles sending ServerHello, Certificate, CertificateRequest, and ServerHelloDone
func (c *TLSConnection) sendServerMessages() error {
	// Send ServerHello
	if err := c.sendServerHello(); err != nil {
		return err
	}

	// Send Server Certificate
	if err := c.sendServerCertificate(); err != nil {
		return err
	}

	// Send CertificateRequest
	if err := c.sendCertificateRequest(); err != nil {
		return err
	}

	// Send ServerHelloDone
	if err := c.sendServerHelloDone(); err != nil {
		return err
	}

	return nil
}

func (c *TLSConnection) sendServerHello() error {
	log.Println("Server: Preparing ServerHello")
	serverHelloPayload := c.secParam.ServerHelloPayload()
	if err := c.writeTLSContent(byte(Handshake), serverHelloPayload); err != nil {
		return err
	}
	log.Println("Server: Sent ServerHello")
	c.msgHistory = append(c.msgHistory, serverHelloPayload...)
	return nil
}

func (c *TLSConnection) sendServerCertificate() error {
	log.Println("Server: Preparing Server Certificate")
	certificatePayload, err := c.secParam.ServerHelloCertificatePayload()
	if err != nil {
		return err
	}

	if err := c.writeTLSContent(byte(Handshake), certificatePayload); err != nil {
		return err
	}
	log.Println("Server: Sent Server Certificate")
	c.msgHistory = append(c.msgHistory, certificatePayload...)
	return nil
}

func (c *TLSConnection) sendCertificateRequest() error {
	log.Println("Server: Preparing CertificateRequest")
	certificateRequestPayload, err := c.secParam.ServerHelloCertificateRequestPayload()
	if err != nil {
		return err
	}

	if err := c.writeTLSContent(byte(Handshake), certificateRequestPayload); err != nil {
		return err
	}
	log.Println("Server: Sent CertificateRequest")
	c.msgHistory = append(c.msgHistory, certificateRequestPayload...)
	return nil
}

func (c *TLSConnection) sendServerHelloDone() error {
	log.Println("Server: Preparing ServerHelloDone")
	serverHelloDonePayload := c.secParam.ServerHelloDonePayload()
	if err := c.writeTLSContent(byte(Handshake), serverHelloDonePayload); err != nil {
		return err
	}
	log.Println("Server: Sent ServerHelloDone")
	c.msgHistory = append(c.msgHistory, serverHelloDonePayload...)
	return nil
}

// receiveClientCertificate handles receiving and processing Client Certificate
func (c *TLSConnection) receiveClientCertificate() error {
	log.Println("Server: Waiting to receive Client Certificate")
	ct, err := c.readTLSContent()
	if err != nil {
		return err
	}

	if ct.tlsType != Handshake {
		return errors.New("expected Handshake content type for Client Certificate")
	}

	handshakeType := HandshakeType(ct.data[0])
	if handshakeType != Certificate {
		return errors.New("expected Certificate handshake type")
	}

	c.msgHistory = append(c.msgHistory, ct.data...)
	log.Println("Server: Received Client Certificate")
	if err := c.secParam.onClientCertificate(ct.data); err != nil {
		return err
	}

	return nil
}

// receiveClientKeyExchange handles receiving and processing ClientKeyExchange
func (c *TLSConnection) receiveClientKeyExchange() ([]byte, error) {
	log.Println("Server: Waiting to receive ClientKeyExchange")
	ct, err := c.readTLSContent()
	if err != nil {
		return nil, err
	}

	if ct.tlsType != Handshake {
		return nil, errors.New("expected Handshake content type for ClientKeyExchange")
	}

	handshakeType := HandshakeType(ct.data[0])
	if handshakeType != ClientKeyExchange {
		return nil, errors.New("expected ClientKeyExchange handshake type")
	}

	c.msgHistory = append(c.msgHistory, ct.data...)
	log.Println("Server: Received ClientKeyExchange")

	preMasterSecret, err := c.secParam.onClientKeyExchange(ct.data)
	if err != nil {
		return nil, err
	}

	return preMasterSecret, nil
}

// receiveCertificateVerify handles receiving and verifying CertificateVerify
func (c *TLSConnection) receiveCertificateVerify(preMasterSecret []byte) error {

	log.Println("Server: Waiting to receive CertificateVerify")
	ct, err := c.readTLSContent()
	if err != nil {
		return err
	}

	if ct.tlsType != Handshake {
		return errors.New("expected Handshake content type for CertificateVerify")
	}

	handshakeType := HandshakeType(ct.data[0])
	if handshakeType != CertificateVerify {
		return errors.New("expected CertificateVerify handshake type")
	}

	log.Println("Server: Received CertificateVerify")
	if err := c.onCertificateVerify(ct.data); err != nil {
		return err
	}
	c.msgHistory = append(c.msgHistory, ct.data...)

	// Generate Master Secret and Keys
	c.secParam.MasterSecret = generateMasterSecret(c.secParam.ClientRandom, c.secParam.ServerRandom, preMasterSecret)
	c.ServerGenerateKeys()

	return nil
}

// receiveClientChangeCipherSpecAndFinished handles receiving ChangeCipherSpec and Finished messages from client
func (c *TLSConnection) receiveClientChangeCipherSpecAndFinished() error {
	// Receive ChangeCipherSpec
	log.Println("Server: Waiting to receive ChangeCipherSpec from client")
	ct, err := c.readTLSContent()
	if err != nil {
		return err
	}

	if ct.tlsType != ChangeCipherSpec {
		return errors.New("expected ChangeCipherSpec from client")
	}
	log.Println("Server: Received ChangeCipherSpec from client")

	// Receive Finished
	log.Println("Server: Waiting to receive Finished from client")
	ct, err = c.readTLSContent()
	if err != nil {
		return err
	}

	decryptedFinished, err := c.decryptContent(ct.data)
	if err != nil {
		return err
	}

	if ct.tlsType != Handshake || decryptedFinished[0] != byte(Finished) {
		log.Println("Server: Invalid Finished payload from client")
		return errors.New("invalid Finished message from client")
	}

	log.Println("Server: Received and verified Finished from client")
	return nil
}

// sendChangeCipherSpecAndFinished handles sending ChangeCipherSpec and Finished messages to client
func (c *TLSConnection) sendChangeCipherSpecAndFinished() error {
	secParm := c.secParam

	// Send ChangeCipherSpec
	log.Println("Server: Sending ChangeCipherSpec to client")
	changeCipherSpecPayload, err := secParm.ServerChangeCipherSpecPayload()
	if err != nil {
		return err
	}

	if err := c.writeTLSContent(byte(ChangeCipherSpec), changeCipherSpecPayload); err != nil {
		return err
	}
	log.Println("Server: Sent ChangeCipherSpec")

	// Send Finished
	log.Println("Server: Preparing Finished payload")
	finishedPayload, err := secParm.ServerFinishedPayload(c.msgHistory)
	if err != nil {
		return err
	}

	encrFinishedPayload, err := c.encryptContent(finishedPayload)
	if err != nil {
		return errors.New("failed to encrypt Finished payload")
	}

	if err := c.writeTLSContent(byte(Handshake), encrFinishedPayload); err != nil {
		return err
	}
	log.Println("Server: Sent Finished")
	return nil
}

// onCertificateVerify verifies the CertificateVerify message
func (c *TLSConnection) onCertificateVerify(data []byte) error {
	data = data[4:]
	if len(data) < 2 {
		return errors.New("invalid CertificateVerify payload length")
	}
	length := int(binary.BigEndian.Uint16(data[:2]))
	if len(data) < 2+length {
		return errors.New("invalid CertificateVerify payload length")
	}
	sig := data[2 : 2+length]

	hashFunc := crypto.SHA256
	hash := sha256.New()
	hash.Write(c.msgHistory)
	msgHistoryHash := hash.Sum(nil)

	secParam := c.secParam

	publicKey, ok := secParam.RemoteCertificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("invalid public key type in certificate")
	}

	err := rsa.VerifyPKCS1v15(publicKey, hashFunc, msgHistoryHash, sig)
	if err != nil {
		return fmt.Errorf("certificate verify failed: %v", err)
	}

	log.Println("Server: CertificateVerify successfully verified")
	return nil
}

// onClientKeyExchange processes the ClientKeyExchange message and returns the preMasterSecret
func (c *SecurityParameters) onClientKeyExchange(data []byte) ([]byte, error) {
	data = data[4:]
	if len(data) < 2 {
		return nil, errors.New("invalid ClientKeyExchange payload length")
	}
	encrPreMasterSecret := data[2:]
	preMasterSecret, err := rsa.DecryptPKCS1v15(rand.Reader, c.CertificatePrivateKey, encrPreMasterSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt preMasterSecret: %v", err)
	}
	return preMasterSecret, nil
}

// onClientHello processes the ClientHello message
func (c *SecurityParameters) onClientHello(data []byte) {
	data = data[4:]
	c.ClientRandom = data[2:34]
}

// onClientCertificate processes the Client Certificate message
func (c *SecurityParameters) onClientCertificate(data []byte) error {
	cert, err := readCertificateFromBuffer(data)
	if err != nil {
		return err
	}
	log.Println("Server: Certificate has been loaded from client.")
	c.RemoteCertificate = cert
	return nil
}

// ServerGenerateKeys generates keys for the server side
func (c *TLSConnection) ServerGenerateKeys() {
	secParm := c.secParam

	seed := []byte("key expansion")
	seed = append(seed, secParm.ServerRandom...)
	seed = append(seed, secParm.ClientRandom...)

	// Generate the key block using the PRF with the master secret and seed
	keyBlock := prf(secParm.MasterSecret, seed, 128)

	keyMaterial := &KeyMaterialBlock{
		ClientWriteMACKey: keyBlock[0:20],
		ServerWriteMACKey: keyBlock[20:40],
		ClientWriteKey:    keyBlock[40:56],
		ServerWriteKey:    keyBlock[56:72],
		ClientWriteIV:     keyBlock[72:88],
		ServerWriteIV:     keyBlock[88:104],
	}

	var err error
	keyMaterial.EncryptCipher, err = aes.NewCipher(keyMaterial.ServerWriteKey)
	if err != nil {
		log.Fatalf("Failed to create EncryptCipher: %v", err)
	}

	keyMaterial.DecryptCipher, err = aes.NewCipher(keyMaterial.ClientWriteKey)
	if err != nil {
		log.Fatalf("Failed to create DecryptCipher: %v", err)
	}

	c.matBlock = keyMaterial
}
