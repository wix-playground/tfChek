package misc

import (
	"crypto/sha512"
	"encoding/hex"
)

func GetPayloadHash(payload []byte) string {
	sum := sha512.Sum512(payload)
	return hex.EncodeToString(sum[:])
}
