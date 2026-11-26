package api

import (
	"crypto/sha256"
	"encoding/hex"
)

func garminCacheKey(apiKey, resource string) string {
	hash := sha256.Sum256(
		[]byte(apiKey + "\x00" + resource),
	)

	return "garmin:" + hex.EncodeToString(hash[:])
}
