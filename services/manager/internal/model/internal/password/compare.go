package password

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ComparePasswordAndHash validates that the provided password and the hash match
func ComparePasswordAndHash(password, encodedPassword string) (bool, error) {
	parts := strings.Split(encodedPassword, "$")

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}

	if argon2.Version != version {
		return false, errors.New("incompatible version of argon2")
	}

	var (
		memory      uint32
		iterations  uint32
		parallelism uint8
	)

	if _, err := fmt.Sscanf(
		parts[3], "m=%d,t=%d,p=%d",
		&memory, &iterations, &parallelism,
	); err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	computed := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(hash)),
	)

	if n := subtle.ConstantTimeCompare(hash, computed); n != 1 {
		return false, nil
	}

	return true, nil
}
