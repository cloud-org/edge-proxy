/*
Copyright 2020 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// modify by ashing

package storage

import "errors"

// ErrStorageAccessConflict is an error for accessing key conflict
var ErrStorageAccessConflict = errors.New("specified key is under accessing")

// ErrStorageNotFound is an error for not found accessing key
var ErrStorageNotFound = errors.New("specified key is not found")

// ErrKeyHasNoContent is an error for file key that has no contents
var ErrKeyHasNoContent = errors.New("specified key has no contents")

// ErrKeyIsEmpty is an error for key is empty
var ErrKeyIsEmpty = errors.New("specified key is empty")

// Store is an interface for caching data into backend storage
type Store interface {
	Create(key string, contents []byte) error
	Get(key string) ([]byte, error)
}
