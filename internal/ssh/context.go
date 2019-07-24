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

	"github.com/hashicorp/terraform/communicator"
)

const (
	sshContextKey = contextKey("ssh")
)

// UIOutput is the interface that must be implemented to output
// data to the end user.
type UIOutput interface {
	Output(string)
}

type OutputFunc func(s string)

func (f OutputFunc) Output(s string) { f(s) }

///////////////////////////////////////////////////////////////////////////////////////////////

type cache map[string]interface{}

///////////////////////////////////////////////////////////////////////////////////////////////

// sshContext is the "internal" context we pass around
type sshContext struct {
	useSudo    bool
	userOutput UIOutput
	execOutput UIOutput
	comm       communicator.Communicator
	cache      cache
}

// WithValues creates a new "internal" SSH context
func WithValues(ctx context.Context, userOutput UIOutput, execOutput UIOutput, comm communicator.Communicator, useSudo bool) context.Context {
	return context.WithValue(ctx, sshContextKey, sshContext{
		useSudo:    useSudo,
		userOutput: userOutput,
		execOutput: execOutput,
		comm:       comm,
		cache:      cache{},
	})
}

func getSSHContext(ctx context.Context) sshContext {
	sshc, ok := ctx.Value(sshContextKey).(sshContext)
	if !ok {
		panic("could not get SSH context info info from context")
	}
	return sshc
}

// GetUseSudoFromContext gets the "shoudl we use sudo?" value
func GetUseSudoFromContext(ctx context.Context) bool {
	return getSSHContext(ctx).useSudo
}

// GetUserOutputFromContext gets the user output
func GetUserOutputFromContext(ctx context.Context) UIOutput {
	return getSSHContext(ctx).userOutput
}

// GetExecOutputFromContext gets the exec output from the current context
func GetExecOutputFromContext(ctx context.Context) UIOutput {
	return getSSHContext(ctx).execOutput
}

// GetCommFromContext gets the communicator from the current context
func GetCommFromContext(ctx context.Context) communicator.Communicator {
	return getSSHContext(ctx).comm
}

// GetCacheFromContext gets the cache from the current context
func GetCacheFromContext(ctx context.Context) cache {
	return getSSHContext(ctx).cache
}

// GetFromCacheInContext gets a value from the cache
func GetFromCacheInContext(ctx context.Context, key string) (interface{}, bool) {
	c := GetCacheFromContext(ctx)
	value, ok := c[key]
	Debug("[CACHE] getting %q [found:%t] = %v ", key, ok, value)
	return value, ok
}

// SetInCacheInContext sets a value in the cache
func SetInCacheInContext(ctx context.Context, key string, value interface{}) {
	c := GetCacheFromContext(ctx)
	Debug("[CACHE] setting %q = %v", key, value)
	c[key] = value
}

// DelInCacheInContext removes akey in the cache
func DelInCacheInContext(ctx context.Context, key string) {
	c := GetCacheFromContext(ctx)
	Debug("[CACHE] deleting %q", key)
	delete(c, key)
}
