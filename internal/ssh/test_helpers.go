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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
)

type DummyOutput struct{}

func (_ DummyOutput) Output(s string) {
	fmt.Print(s)
}

type DummyCommunicator struct {
}

func (_ DummyCommunicator) Connect(terraform.UIOutput) error {
	Debug("DummyCommunicator: Connect()")
	return nil
}

func (_ DummyCommunicator) Disconnect() error {
	Debug("DummyCommunicator: Disconnect()")
	return nil
}

func (_ DummyCommunicator) Timeout() time.Duration {
	Debug("DummyCommunicator: Timeout()")
	return 1 * time.Hour
}

func (_ DummyCommunicator) ScriptPath() string {
	Debug("DummyCommunicator: ScriptPath()")
	return ""
}

func (dc DummyCommunicator) Start(cmd *remote.Cmd) error {
	Debug("DummyCommunicator: Start(%s)", cmd.Command)
	return nil
}

func (_ DummyCommunicator) Upload(string, io.Reader) error {
	Debug("DummyCommunicator: Upload()")
	return nil
}

func (_ DummyCommunicator) UploadScript(string, io.Reader) error {
	Debug("DummyCommunicator: UploadScript()")
	return nil
}

func (_ DummyCommunicator) UploadDir(string, string) error {
	Debug("DummyCommunicator: UploadDir()")
	return nil
}

////////////////////////////////////////////////////////////////////////////

func NewTestingContextWithCommunicator(comm communicator.Communicator) context.Context {
	ctx := context.Background()
	out := DummyOutput{}
	return WithValues(ctx, out, out, comm, false)
}

func NewTestingContext() context.Context {
	return NewTestingContextWithCommunicator(DummyCommunicator{})
}

type dummyCommunicatorWithResponses struct {
	DummyCommunicator

	responses []string
	counter   *int
	uploads   map[string]int
}

func (dc dummyCommunicatorWithResponses) Start(cmd *remote.Cmd) error {
	cmd.Init()
	if (*dc.counter) >= len(dc.responses) {
		cmd.Stdout.Write([]byte(""))
	} else {
		cmd.Stdout.Write([]byte(dc.responses[*dc.counter]))
	}
	cmd.SetExitStatus(0, nil)
	*dc.counter += 1
	return nil
}

func (dc dummyCommunicatorWithResponses) Upload(dst string, r io.Reader) error {
	all, _ := ioutil.ReadAll(r)
	dc.uploads[dst] = len(all)
	return nil
}

func (dc dummyCommunicatorWithResponses) UploadScript(dst string, r io.Reader) error {
	all, _ := ioutil.ReadAll(r)
	dc.uploads[dst] = len(all)
	return nil
}

func (_ dummyCommunicatorWithResponses) UploadDir(string, string) error {
	return nil
}

func NewTestingContextWithResponses(responses []string) context.Context {
	// we must keep the "counter" out of the communicator object as this
	// object is inmutable... :-/
	counter := 0
	comm := dummyCommunicatorWithResponses{responses: responses, counter: &counter}
	return NewTestingContextWithCommunicator(comm)
}

func NewTestingContextForUploads(responses []string) (context.Context, *map[string]int) {
	// we must keep the "counter" out of the communicator object as this
	// object is inmutable... :-/
	uploads := map[string]int{}
	counter := 0
	comm := dummyCommunicatorWithResponses{responses: responses, counter: &counter, uploads: uploads}
	return NewTestingContextWithCommunicator(comm), &uploads
}
