package common

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// GetParentPath extracts the parent path from a path
// Example: /protocols/simplebuzz -> /protocols
func GetParentPath(path string) string {
	arr := strings.Split(path, "/")
	if len(arr) < 3 {
		return ""
	}
	return strings.Join(arr[0:len(arr)-1], "/")
}

// ValidateOperation validates if the operation type is valid
func ValidateOperation(operation string) bool {
	validOps := map[string]bool{
		"create": true,
		"modify": true,
		"revoke": true,
	}
	return validOps[strings.ToLower(operation)]
}

// NormalizeContentType normalizes the content-type
func NormalizeContentType(contentType string) string {
	if contentType == "" {
		return "application/json"
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

// NormalizePath normalizes a path
func NormalizePath(path string) string {
	return strings.ToLower(strings.TrimSpace(path))
}

// CalculateMetaId calculates MetaID (sha256(address))
func CalculateMetaId(address string) string {
	if address == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(address))
	return hex.EncodeToString(hash[:])
}
