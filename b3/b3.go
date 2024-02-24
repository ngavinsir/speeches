package b3

import (
	"encoding/hex"
	"fmt"
	"io"

	"lukechampine.com/blake3"
)

func Blake3HashFromFile(f io.Reader) (string, error) {
	h := blake3.New(32, nil)
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("calculating blake3 hash from file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
