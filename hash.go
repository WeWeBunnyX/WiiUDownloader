package wiiudownloader

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"fmt"
	"os"
	"reflect"
)

var commonKey = []byte{0xD7, 0xB0, 0x04, 0x02, 0x65, 0x9B, 0xA2, 0xAB, 0xD2, 0xCB, 0x0D, 0xB2, 0x7F, 0xA2, 0xB6, 0x56}

func checkContentHashes(path string, encryptedTitleKey []byte, titleID []byte, content contentInfo) error {
	c, err := aes.NewCipher(commonKey)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	decryptedTitleKey := make([]byte, len(encryptedTitleKey))
	cbc := cipher.NewCBCDecrypter(c, append(titleID, make([]byte, 8)...))
	cbc.CryptBlocks(decryptedTitleKey, encryptedTitleKey)

	h3Data, err := os.ReadFile(fmt.Sprintf("%s/%s.h3", path, content.ID))
	if err != nil {
		return fmt.Errorf("failed to read H3 hash tree file: %w", err)
	}
	encryptedFile, err := os.Open(fmt.Sprintf("%s/%s.app", path, content.ID))
	if err != nil {
		return fmt.Errorf("failed to open encrypted file: %w", err)
	}

	h3Hash := sha1.Sum(h3Data)
	if !reflect.DeepEqual(h3Hash[:8], content.Hash[:8]) {
		return fmt.Errorf("h3 Hash mismatch")
	}

	chunkCount := int(content.Size / 0x10000)
	decryptedContent := make([]byte, content.Size)

	h0HashNum := 0
	h1HashNum := 0
	h2HashNum := 0
	h3HashNum := 0

	for chunkNum := 0; chunkNum < chunkCount; chunkNum++ {
		cipherHashTree, err := aes.NewCipher(decryptedTitleKey)
		if err != nil {
			return fmt.Errorf("failed to create AES cipher: %w", err)
		}
		hashTree := cipher.NewCBCDecrypter(cipherHashTree, make([]byte, aes.BlockSize))
		buffer := make([]byte, 0x400)
		encryptedFile.Read(buffer)
		hashTree.CryptBlocks(decryptedContent, buffer)

		h0Hashes := decryptedContent[0:0x140]
		h1Hashes := decryptedContent[0x140:0x280]
		h2Hashes := decryptedContent[0x280:0x3c0]

		h1Hash := h1Hashes[(h1HashNum * 0x14):((h1HashNum + 1) * 0x14)]
		h2Hash := h2Hashes[(h2HashNum * 0x14):((h2HashNum + 1) * 0x14)]
		h3Hash := h3Data[(h3HashNum * 0x14):((h3HashNum + 1) * 0x14)]

		h0HashesHash := sha1.Sum(h0Hashes)
		h1HashesHash := sha1.Sum(h1Hashes)
		h2HashesHash := sha1.Sum(h2Hashes)

		if !reflect.DeepEqual(h0HashesHash[:], h1Hash) {
			return fmt.Errorf("h0 Hashes Hash mismatch")
		}
		if !reflect.DeepEqual(h1HashesHash[:], h2Hash) {
			return fmt.Errorf("h1 Hashes Hash mismatch")
		}
		if !reflect.DeepEqual(h2HashesHash[:], h3Hash) {
			return fmt.Errorf("h2 Hashes Hash mismatch")
		}
		encryptedFile.Seek(0xFC00, 1)
		h0HashNum++
		if h0HashNum >= 16 {
			h0HashNum = 0
			h1HashNum++
		}
		if h1HashNum >= 16 {
			h1HashNum = 0
			h2HashNum++
		}
		if h2HashNum >= 16 {
			h2HashNum = 0
			h3HashNum++
		}
	}
	return nil
}

type contentInfo struct {
	ID   string
	Size int64
	Hash []byte
}
