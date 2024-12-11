/*
	Proxy functions for regular socket operations (read/write).
	Internal operations for sending tls data over the wire.
*/

package tls

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
)

func (c *TLSConnection) writeTLSContent(contentType byte, data []byte) error {

	clientVersion := []byte{byte(major), byte(minor)}
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(data)))
	payload := bytes.Join([][]byte{
		[]byte{contentType},
		clientVersion,
		length,
		data,
	}, nil)
	_, err := c.conn.Write(payload)
	return err
}

func (c *TLSConnection) readTLSContent() (*TLSCiphertext, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return nil, err
	}

	var ct TLSCiphertext
	ct.tlsType = ContentType(header[0])

	ct.version = make([]byte, 2)
	ct.length = make([]byte, 2)
	copy(ct.version[:], header[1:3])
	copy(ct.length[:], header[3:5])

	payloadLength := binary.BigEndian.Uint16(ct.length[:])
	ct.data = make([]byte, payloadLength)
	if _, err := io.ReadFull(c.conn, ct.data); err != nil {
		return nil, err
	}

	return &ct, nil
}

/*
Encrypt content and return
*/
func (c *TLSConnection) encryptContent(data []byte) ([]byte, error) {
	blockSize := c.matBlock.EncryptCipher.BlockSize()
	padding := blockSize - len(data)%blockSize
	plaintext := append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)

	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	ciphertext := make([]byte, blockSize+len(plaintext))
	copy(ciphertext, iv)
	mode := cipher.NewCBCEncrypter(c.matBlock.EncryptCipher, iv)
	mode.CryptBlocks(ciphertext[blockSize:], plaintext)

	return ciphertext, nil
}

func (c *TLSConnection) decryptContent(data []byte) ([]byte, error) {
	blockSize := c.matBlock.DecryptCipher.BlockSize()
	println(blockSize)
	println(len(data))

	if len(data) < blockSize {
		return nil, errors.New("bad length")
	}
	iv, ciphertext := data[:blockSize], data[blockSize:]
	if len(ciphertext)%blockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}
	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(c.matBlock.DecryptCipher, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	paddingLen := int(plaintext[len(plaintext)-1])
	if paddingLen == 0 || paddingLen > blockSize {
		return nil, errors.New("invalid padding")
	}
	return plaintext[:len(plaintext)-paddingLen], nil
}
