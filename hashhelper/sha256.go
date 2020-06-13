package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func Sha256SumHex(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
