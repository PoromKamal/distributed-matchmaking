package tls

import (
	"errors"
	"net"
)

type TLSConnectionConfig struct {
	Conn     net.Conn
	IsServer bool
	CertPath string
	KeyPath  string
}

/*
May need to add a overlay for server to keep track of connections.
*/

func NewTLSConn(config *TLSConnectionConfig) (*TLSConnection, error) {
	endType := ClientCE
	if config.IsServer {
		endType = ServerCE
	}

	publicCert, certErr := readCertificateFromFile(config.CertPath)
	privateKey, pkErr := readPrivateKeyFromFile(config.KeyPath)

	if certErr != nil || pkErr != nil {
		return nil, errors.New("Failed to read Certificate / Private key")
	}

	return &TLSConnection{
		conn: config.Conn,
		secParam: &SecurityParameters{
			ConnectionEnd:         endType,
			Certificate:           publicCert,
			CertificatePrivateKey: privateKey,
		},
		matBlock: &KeyMaterialBlock{},
		state:    0,
	}, nil
}

func (c *TLSConnection) Read(data []byte) (int, error) {
	ct, err := c.readTLSContent()
	if err != nil {
		return 0, err
	}
	decrData, err := c.decryptContent(ct.data)
	if err != nil {
		return 0, err
	}
	copy(data, decrData)
	return len(data), nil
}

func (c *TLSConnection) Write(data []byte) (int, error) {
	encrData, err := c.encryptContent(data)
	if err != nil {
		return 0, errors.New("failed to encrypt Finished payload")
	}

	if err := c.writeTLSContent(byte(ApplicationData), encrData); err != nil {
		return 0, err
	}
	return len(data), nil
}
