package tls

import (
	"crypto/aes"
	"errors"
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

func (c *TLSConnection) HandshakeClientController() error {

	// Send ClientHello
	if err := c.sendClientHello(); err != nil {
		return err
	}

	// Receive Server Info
	if err := c.receiveServerMessages(); err != nil {
		return err
	}

	// Send Client Certificate
	if err := c.sendClientCertificate(); err != nil {
		return err
	}

	// Send ClientKeyExchange
	preMasterSecret, err := c.sendClientKeyExchange()
	if err != nil {
		return err
	}

	// Send CertificateVerify
	if err := c.sendCertificateVerify(); err != nil {
		return err
	}

	// Generate Master Secret and Keys
	c.secParam.MasterSecret = generateMasterSecret(c.secParam.ClientRandom, c.secParam.ServerRandom, preMasterSecret)
	c.ClientGenerateKeys()

	// Send ChangeCipherSpec
	if err := c.sendChangeCipherSpec(); err != nil {
		return err
	}

	// Send Finished
	if err := c.sendFinished(); err != nil {
		return err
	}

	// Receive Server's ChangeCipherSpec and Finished
	if err := c.receiveServerFinished(); err != nil {
		return err
	}

	log.Println("Client: Handshake completed successfully")
	return nil
}

func (c *TLSConnection) receiveServerMessages() error {
	log.Println("Client: Waiting to receive server messages")
	for {
		ct, err := c.readTLSContent()
		if err != nil {
			return err
		}

		if ct.tlsType != Handshake {
			return errors.New("unexpected content type")
		}

		c.msgHistory = append(c.msgHistory, ct.data...)
		handshakeType := HandshakeType(ct.data[0])

		switch handshakeType {
		case ServerHello:
			log.Println("Client: Received ServerHello")
			c.secParam.onServerHello(ct.data)
		case Certificate:
			log.Println("Client: Received Server Certificate")
			if err := c.secParam.onServerCertificate(ct.data); err != nil {
				return err
			}
		case CertificateRequest:
			log.Println("Client: Received CertificateRequest")
			// NOOP for now
		case ServerHelloDone:
			log.Println("Client: Received ServerHelloDone")
		default:
			return errors.New("unexpected handshake message type")
		}

		if handshakeType == ServerHelloDone {
			break
		}
	}
	return nil
}

func (c *TLSConnection) sendClientHello() error {
	log.Println("Client: Preparing Client Hello")
	clientHelloPayload := c.secParam.ClientHelloPayload()
	if err := c.writeTLSContent(byte(Handshake), clientHelloPayload); err != nil {
		return err
	}
	log.Println("Client: Sent ClientHello")
	c.msgHistory = append(c.msgHistory, clientHelloPayload...)
	return nil
}

func (c *TLSConnection) sendClientCertificate() error {
	// Check if server requested a certificate
	certificatePayload, err := c.secParam.ClientHelloCertificatePayload()
	if err != nil {
		return err
	}

	if err := c.writeTLSContent(byte(Handshake), certificatePayload); err != nil {
		return err
	}
	log.Println("Client: Sent Client Certificate")
	c.msgHistory = append(c.msgHistory, certificatePayload...)
	return nil
}

func (c *TLSConnection) sendClientKeyExchange() ([]byte, error) {
	log.Println("Client: Computing preMasterSecrets")
	keyExchangePayload, preMasterSecret, err := c.secParam.ClientHelloKeyExchangePayload()
	if err != nil {
		log.Println("Client: Error computing preMasterSecret")
		return nil, err
	}

	if err := c.writeTLSContent(byte(Handshake), keyExchangePayload); err != nil {
		log.Println("Client: Error writing ClientKeyExchange")
		return nil, err
	}

	log.Println("Client: Sent ClientKeyExchange")
	c.msgHistory = append(c.msgHistory, keyExchangePayload...)
	return preMasterSecret, nil
}

func (c *TLSConnection) sendCertificateVerify() error {
	// Check if a certificate was sent and needs to be verified
	certificateVerifyPayload, err := c.secParam.ClientHelloCertificateVerifyPayload(c.msgHistory)
	if err != nil {
		return err
	}

	if err := c.writeTLSContent(byte(Handshake), certificateVerifyPayload); err != nil {
		return err
	}

	log.Println("Client: Sent CertificateVerifyPayload")
	c.msgHistory = append(c.msgHistory, certificateVerifyPayload...)
	return nil
}

func (c *TLSConnection) sendChangeCipherSpec() error {
	changeCipherSpecPayload, err := c.secParam.ClientChangeCipherSpecPayload()
	if err != nil {
		return err
	}

	if err := c.writeTLSContent(byte(ChangeCipherSpec), changeCipherSpecPayload); err != nil {
		return err
	}

	log.Println("Client: Sent ChangeCipherSpec")
	return nil
}

func (c *TLSConnection) sendFinished() error {
	finishedPayload, err := c.secParam.ClientFinishedPayload(c.msgHistory)
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

	log.Println("Client: Sent Finished")
	return nil
}

func (c *TLSConnection) receiveServerFinished() error {
	log.Println("Client: Waiting to receive Server's ChangeCipherSpec")
	ct, err := c.readTLSContent()
	if err != nil {
		return err
	}

	if ct.tlsType != ChangeCipherSpec {
		return errors.New("expected ChangeCipherSpec from server")
	}
	log.Println("Client: Received ChangeCipherSpec from server")

	log.Println("Client: Waiting to receive Server's Finished")
	ct, err = c.readTLSContent()
	if err != nil {
		return err
	}

	decrFinishedPayload, err := c.decryptContent(ct.data)
	if err != nil {
		return err
	}

	if ct.tlsType != Handshake || decrFinishedPayload[0] != byte(Finished) {
		log.Println("Client: Invalid Finished payload from server")
		return errors.New("invalid Finished message from server")
	}

	log.Println("Client: Received and verified Server's Finished")
	return nil
}

func (c *SecurityParameters) onServerHello(data []byte) {
	data = data[4:]
	c.ServerRandom = data[2:34]
}

func (c *SecurityParameters) onServerCertificate(data []byte) error {
	cert, err := readCertificateFromBuffer(data)
	if err != nil {
		return err
	}
	println("Certificate has been loaded from remote.")
	c.RemoteCertificate = cert
	return nil
}

func (c *TLSConnection) ClientGenerateKeys() {
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

	keyMaterial.EncryptCipher, _ = aes.NewCipher(keyMaterial.ClientWriteKey)
	keyMaterial.DecryptCipher, _ = aes.NewCipher(keyMaterial.ServerWriteKey)
	c.matBlock = keyMaterial
}
