package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2idVersion     = 19
	argon2idMemoryKiB   = 19 * 1024
	argon2idIterations  = 2
	argon2idParallelism = 1
	argon2idSaltLength  = 16
	argon2idKeyLength   = 32
	passwordAlphabet    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

var errInvalidPasswordHash = errors.New("invalid password hash")

type argon2idParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	keyLength   uint32
}

// HashPassword returns a PHC-style Argon2id hash suitable for database storage.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2idSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2idIterations, argon2idMemoryKiB, argon2idParallelism, argon2idKeyLength)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idVersion,
		argon2idMemoryKiB,
		argon2idIterations,
		argon2idParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func IsArgon2idHash(encoded string) bool {
	return strings.HasPrefix(strings.TrimSpace(encoded), "$argon2id$")
}

func VerifyPassword(password string, encoded string) bool {
	params, salt, expected, err := decodeArgon2idHash(encoded)
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, params.keyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func RandomPassword(length int) (string, error) {
	if length <= 0 {
		length = 16
	}
	out := make([]byte, length)
	max := big.NewInt(int64(len(passwordAlphabet)))
	for i := range out {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = passwordAlphabet[index.Int64()]
	}
	return string(out), nil
}

func decodeArgon2idHash(encoded string) (argon2idParams, []byte, []byte, error) {
	parts := strings.Split(strings.TrimSpace(encoded), "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return argon2idParams{}, nil, nil, errInvalidPasswordHash
	}
	versionText, ok := strings.CutPrefix(parts[2], "v=")
	if !ok {
		return argon2idParams{}, nil, nil, errInvalidPasswordHash
	}
	version, err := strconv.Atoi(versionText)
	if err != nil || version != argon2idVersion {
		return argon2idParams{}, nil, nil, errInvalidPasswordHash
	}
	params, err := parseArgon2idParams(parts[3])
	if err != nil {
		return argon2idParams{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2idParams{}, nil, nil, errInvalidPasswordHash
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) == 0 {
		return argon2idParams{}, nil, nil, errInvalidPasswordHash
	}
	params.keyLength = uint32(len(hash))
	return params, salt, hash, nil
}

func parseArgon2idParams(encoded string) (argon2idParams, error) {
	values := map[string]int{}
	for part := range strings.SplitSeq(encoded, ",") {
		key, raw, ok := strings.Cut(part, "=")
		if !ok {
			return argon2idParams{}, errInvalidPasswordHash
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return argon2idParams{}, errInvalidPasswordHash
		}
		values[key] = value
	}
	memory := values["m"]
	iterations := values["t"]
	parallelism := values["p"]
	if memory <= 0 || iterations <= 0 || parallelism <= 0 || parallelism > 255 {
		return argon2idParams{}, errInvalidPasswordHash
	}
	return argon2idParams{
		memory:      uint32(memory),
		iterations:  uint32(iterations),
		parallelism: uint8(parallelism),
	}, nil
}
