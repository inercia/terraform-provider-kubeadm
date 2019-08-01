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

package ssh

import (
	"bytes"
	"text/template"
)

// ReplaceInTemplate performs replacements in an input text
func ReplaceInTemplate(text string, replacements map[string]interface{}) (string, error) {
	tmpl, err := template.New("template").Parse(text)
	if err != nil {
		return "", err
	}

	b := bytes.Buffer{}
	if err := tmpl.Execute(&b, replacements); err != nil {
		return "", err
	}
	return b.String(), nil
}
