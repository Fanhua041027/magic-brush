package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

var legacyAESKey = func() []byte {
	h := sha256.Sum256([]byte("JTwZwHaMq3g55PfVgxkah1Edst59LoyF"))
	return h[:]
}()

var currentCipherHeader = []byte("MB2\x00")

func encrypt(plaintext []byte) ([]byte, error) {
	key, err := currentEncryptionKey()
	if err != nil {
		return nil, err
	}
	sealed, err := sealWithKey(key, plaintext)
	if err != nil {
		return nil, err
	}
	return append(append([]byte{}, currentCipherHeader...), sealed...), nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	key := legacyAESKey
	if len(ciphertext) >= len(currentCipherHeader) && string(ciphertext[:len(currentCipherHeader)]) == string(currentCipherHeader) {
		var err error
		key, err = currentEncryptionKey()
		if err != nil {
			return nil, err
		}
		ciphertext = ciphertext[len(currentCipherHeader):]
	}
	return openWithKey(key, ciphertext)
}

func sealWithKey(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func openWithKey(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("密文过短")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
