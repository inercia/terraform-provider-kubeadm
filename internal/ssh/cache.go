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
	"os"
	"strconv"
)

const (
	// environmental variable that can be used for disabling the cache
	cacheEnvVar = "TF_CACHE"
)

const (
	// CacheRemoteFileExistsPrefix is the prefix for file checks
	CacheRemoteFileExistsPrefix = "remote-file-exists"

	// CacheRemoteDirExistsPrefix is the prefix for dir checks
	CacheRemoteDirExistsPrefix = "remote-dir-exists"
)

func isCacheDisabled() bool {
	enabledStr := os.Getenv(cacheEnvVar)
	if len(enabledStr) > 0 {
		enabled, _ := strconv.ParseBool(enabledStr)
		return !enabled
	}
	return false
}

// getFromCacheInContext gets a value from the cache
func getFromCacheInContext(ctx context.Context, key string) (interface{}, bool) {
	if isCacheDisabled() {
		return nil, false
	}
	c := getCacheFromContext(ctx)
	value, ok := c[key]
	Debug("[CACHE] getting %q [found:%t] = %v ", key, ok, value)
	return value, ok
}

// setInCacheInContext sets a value in the cache
func setInCacheInContext(ctx context.Context, key string, value interface{}) {
	if isCacheDisabled() {
		return
	}
	c := getCacheFromContext(ctx)
	Debug("[CACHE] setting %q = %v", key, value)
	c[key] = value
}

// delInCacheInContext removes akey in the cache
func delInCacheInContext(ctx context.Context, key string) {
	if isCacheDisabled() {
		return
	}
	c := getCacheFromContext(ctx)
	Debug("[CACHE] deleting %q", key)
	delete(c, key)
}

// DoOnce runs an action if it is not been saved in the cache
// If the action does not produce any error, it is saved in the cache under the `key`.
// If the action produces an error, the error is returned and nothing is saved in the cache,
// so any subsequent DoOnce for the same key will be executed
func DoOnce(key string, action Action) Action {
	return DoIf(
		CheckNot(CheckInCache(key)),
		ActionFunc(func(ctx context.Context) Action {
			res := ActionList{action}.Apply(ctx)
			if !IsError(res) {
				// save in the cache only when no errors happen
				setInCacheInContext(ctx, key, true)
			}
			return res
		}))
}

// DoRemoveFromCache removes some key from the cache
func DoRemoveFromCache(key string) Action {
	return ActionFunc(func(ctx context.Context) Action {
		delInCacheInContext(ctx, key)
		return nil
	})
}

// DoSetInCache sets some key in the cache
func DoSetInCache(key string, value interface{}) Action {
	return ActionFunc(func(ctx context.Context) Action {
		setInCacheInContext(ctx, key, value)
		return nil
	})
}

// CheckInCache returns true if the key is in the cache
func CheckInCache(key string) CheckerFunc {
	return CheckerFunc(func(ctx context.Context) (bool, error) {
		_, ok := getFromCacheInContext(ctx, key)
		if ok {
			return true, nil
		}
		return false, nil
	})
}

// CheckOnce checks if there is a cached result for the `key`. If not,
// runs the check, storing the result in the cache
func CheckOnce(key string, check Checker) CheckerFunc {
	return CheckerFunc(func(ctx context.Context) (bool, error) {
		value, ok := getFromCacheInContext(ctx, key)
		if ok {
			return value.(bool), nil
		}

		res, err := check.Check(ctx)
		if err != nil {
			return false, err
		}

		setInCacheInContext(ctx, key, res)
		return res, nil
	})
}
