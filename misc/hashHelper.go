package misc

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
)

const PAYLOADHASH_SHA512 string = "sha512"

func GetPayloadHash(payload []byte, algo string) (string, error) {
	switch algo {
	case PAYLOADHASH_SHA512:
		sum := sha512.Sum512(payload)
		return hex.EncodeToString(sum[:]), nil
	default:
		return "not implemented", errors.New("Not implemented. Use sha512")
	}
}
