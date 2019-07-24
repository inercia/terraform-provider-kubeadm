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

import "context"

const (
	// CacheRemoteFileExistsPrefix is the prefix for file checks
	CacheRemoteFileExistsPrefix = "remote-file-exists"

	// CacheRemoteDirExistsPrefix is the prefix for dir checks
	CacheRemoteDirExistsPrefix = "remote-dir-exists"
)

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
				SetInCacheInContext(ctx, key, true)
			}
			return res
		}))
}

// DoRemoveFromCache removes some key from the cache
func DoRemoveFromCache(key string) Action {
	return ActionFunc(func(ctx context.Context) Action {
		DelInCacheInContext(ctx, key)
		return nil
	})
}

// CheckInCache returns true if the key is in the cache
func CheckInCache(key string) CheckerFunc {
	return CheckerFunc(func(ctx context.Context) (bool, error) {
		_, ok := GetFromCacheInContext(ctx, key)
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
		value, ok := GetFromCacheInContext(ctx, key)
		if ok {
			return value.(bool), nil
		}

		res, err := check.Check(ctx)
		if err != nil {
			return false, err
		}

		SetInCacheInContext(ctx, key, res)
		return res, nil
	})
}
