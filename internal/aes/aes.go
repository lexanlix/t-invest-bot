package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/pkg/errors"
)

func Encrypt(key []byte, data []byte) ([]byte, error) {
	// Создаем новый блок AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.WithMessage(err, "new aes cipher")
	}

	// Дополняем данные до блока
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.WithMessage(err, "new gcm")
	}

	// Генерируем случайный nonce
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, errors.WithMessage(err, "read nonce")
	}

	// Шифруем данные
	encrypted := gcm.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

func Decrypt(key []byte, data []byte) ([]byte, error) {
	// Создаем новый блок AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.WithMessage(err, "new aes cipher")
	}

	// Проверяем размер nonce
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.WithMessage(err, "new gcm")
	}

	// Извлекаем nonce из данных
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Расшифровываем данные
	decrypted, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "gcm open")
	}

	return decrypted, nil
}
