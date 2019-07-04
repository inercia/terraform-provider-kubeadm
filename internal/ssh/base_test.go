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
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
)

type DummyOutput struct{}

func (_ DummyOutput) Output(s string) {
	fmt.Print(s)
}

type DummyCommunicator struct{}

func (_ DummyCommunicator) Connect(terraform.UIOutput) error {
	return nil
}

func (_ DummyCommunicator) Disconnect() error {
	return nil
}

func (_ DummyCommunicator) Timeout() time.Duration {
	return 1 * time.Hour
}

func (_ DummyCommunicator) ScriptPath() string {
	return ""
}

func (_ DummyCommunicator) Start(*remote.Cmd) error {
	return nil
}

func (_ DummyCommunicator) Upload(string, io.Reader) error {
	return nil
}

func (_ DummyCommunicator) UploadScript(string, io.Reader) error {
	return nil
}

func (_ DummyCommunicator) UploadDir(string, string) error {
	return nil
}

func TestApply(t *testing.T) {
	counter := 0
	appliers := []Applyer{
		DoMessage("test"),
		ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
			counter = counter + 1
			return nil
		}),
		ApplyError("some error"),
	}

	o := DummyOutput{}
	comm := DummyCommunicator{}

	err := Apply(appliers, o, comm, false)
	if err == nil {
		t.Fatal("Error: no error detected")
	}
	if counter > 0 {
		t.Fatal("Error: error was raised after some function was run")
	}

}
