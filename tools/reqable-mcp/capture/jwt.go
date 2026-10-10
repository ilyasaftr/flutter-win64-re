package capture

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func decodeSegment(s string) ([]byte, error) {
	// JWT uses unpadded base64url; restore padding before decoding.
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64 segment: %w", err)
	}
	return raw, nil
}

// DecodeJWTUnverified decodes a JWT header and payload without verifying
// the signature.
func DecodeJWTUnverified(token string) (map[string]any, map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, nil, fmt.Errorf("malformed JWT: want header.payload[.signature]")
	}
	hraw, err := decodeSegment(parts[0])
	if err != nil {
		return nil, nil, fmt.Errorf("JWT header: %w", err)
	}
	praw, err := decodeSegment(parts[1])
	if err != nil {
		return nil, nil, fmt.Errorf("JWT payload: %w", err)
	}
	var header, payload map[string]any
	if err := json.Unmarshal(hraw, &header); err != nil {
		return nil, nil, fmt.Errorf("JWT header JSON: %w", err)
	}
	if err := json.Unmarshal(praw, &payload); err != nil {
		return nil, nil, fmt.Errorf("JWT payload JSON: %w", err)
	}
	return header, payload, nil
}
