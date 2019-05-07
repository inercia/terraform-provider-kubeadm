package common

import (
	"encoding/base64"
)

func ToTerraformSafeString(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

func FromTerraformSafeString(data string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(data)
}
