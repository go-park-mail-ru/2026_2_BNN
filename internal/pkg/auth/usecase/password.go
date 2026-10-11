package usecase

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memory      uint32 = 19 * 1024
	iterations  uint32 = 2
	parallelism uint8  = 1
	keyLength   uint32 = 32
	saltLength         = 16
)

func hashPassword(password string) ([]byte, error) {
	salt := make([]byte, saltLength)

	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	saltBase64 := base64.RawStdEncoding.EncodeToString(salt)
	hashBase64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		saltBase64,
		hashBase64,
	)

	return []byte(encoded), nil
}

func verifyPassword(password string, encodedHash []byte) bool {
	parts := strings.Split(string(encodedHash), "$")

	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}

	var (
		hashMemory      uint32
		hashIterations  uint32
		hashParallelism uint8
	)

	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &hashMemory, &hashIterations, &hashParallelism)
	if err != nil || hashIterations < 1 || hashParallelism < 1 {
		return false
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(expectedHash) == 0 {
		return false
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		hashIterations,
		hashMemory,
		hashParallelism,
		uint32(len(expectedHash)),
	)

	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1
}
