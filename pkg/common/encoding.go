// Copyright © 2019 Alvaro Saurin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"encoding/base64"
)

// ToTerraformSafeString converts some (possibly binary) data to a
// string that can be stored in the Terraform state
func ToTerraformSafeString(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

// FromTerraformSafeString converts some Terraform state data to
// (possibly binary) data.
func FromTerraformSafeString(data string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(data)
}
