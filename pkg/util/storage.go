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

package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"code.aliyun.com/openyurt/edge-proxy/pkg/util/storage"
	"k8s.io/klog/v2"
)

const tmpPrefix = "tmp_"

type diskStorage struct {
	baseDir          string
	keyPendingStatus map[string]struct{}
	sync.Mutex
}

// NewDiskStorage creates a storage.Store for caching data into local disk
func NewDiskStorage(dir string) (storage.Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("empty dir")
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	ds := &diskStorage{
		keyPendingStatus: make(map[string]struct{}),
		baseDir:          dir,
	}

	err := ds.Recover("")
	if err != nil {
		klog.Errorf("could not recover local storage, %v, and skip the error", err)
	}
	return ds, nil
}

// Create new a file with key and contents or create dir only
// when contents are empty.
func (ds *diskStorage) Create(key string, contents []byte) error {
	if key == "" {
		return storage.ErrKeyIsEmpty
	}

	if !ds.lockKey(key) {
		return storage.ErrStorageAccessConflict
	}
	defer ds.unLockKey(key)

	// no contents, create key dir only
	if len(contents) == 0 {
		keyPath := filepath.Join(ds.baseDir, key)
		if info, err := os.Stat(keyPath); err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(keyPath, 0755); err == nil {
					return nil
				}
			}
			return err
		} else if info.IsDir() {
			return nil
		} else {
			return storage.ErrKeyHasNoContent
		}
	}

	return ds.create(key, contents)
}

// create will make up a file with key as file path and contents as file contents.
func (ds *diskStorage) create(key string, contents []byte) error {
	if key == "" {
		return storage.ErrKeyIsEmpty
	}

	keyPath := filepath.Join(ds.baseDir, key)
	dir, _ := filepath.Split(keyPath)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// dir for key is already exist
	}

	// open file with synchronous I/O
	f, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_SYNC, 0600)
	if err != nil {
		return err
	}
	n, err := f.Write(contents)
	if err == nil && n < len(contents) {
		err = io.ErrShortWrite
	}

	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

// Get get contents from the file that specified by key
func (ds *diskStorage) Get(key string) ([]byte, error) {
	if key == "" {
		return []byte{}, storage.ErrKeyIsEmpty
	}

	if !ds.lockKey(key) {
		return nil, storage.ErrStorageAccessConflict
	}
	defer ds.unLockKey(key)
	return ds.get(filepath.Join(ds.baseDir, key))
}

// get returns contents from the file of path
func (ds *diskStorage) get(path string) ([]byte, error) {
	if path == "" {
		return []byte{}, storage.ErrKeyIsEmpty
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte{}, storage.ErrStorageNotFound
		}
		return nil, fmt.Errorf("failed to get bytes from %s, %w", path, err)
	} else if info.Mode().IsRegular() {
		b, err := os.ReadFile(path)
		if err != nil {
			return []byte{}, err
		}

		return b, nil
	} else if info.IsDir() {
		return []byte{}, storage.ErrKeyHasNoContent
	}

	return nil, fmt.Errorf("%s is exist, but not recognized, %v", path, info.Mode())
}

// Recover recover storage error
func (ds *diskStorage) Recover(key string) error {
	if !ds.lockKey(key) {
		return nil
	}
	defer ds.unLockKey(key)

	dir := filepath.Join(ds.baseDir, key)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			if isTmpFile(path) {
				tmpKey := strings.TrimPrefix(path, ds.baseDir)
				key := getKey(tmpKey)
				keyPath := filepath.Join(ds.baseDir, key)
				iErr := os.Rename(path, keyPath)
				if iErr != nil {
					klog.V(2).Infof("failed to recover bytes %s, %v", tmpKey, err)
					return nil
				}
				klog.V(2).Infof("bytes %s recovered successfully", key)
			}
		}

		return nil
	})

	return err
}

func (ds *diskStorage) lockKey(key string) bool {
	ds.Lock()
	defer ds.Unlock()
	if _, ok := ds.keyPendingStatus[key]; ok {
		klog.Infof("key(%s) storage is pending, just skip it", key)
		return false
	}

	for pendingKey := range ds.keyPendingStatus {
		if len(key) > len(pendingKey) {
			if strings.Contains(key, fmt.Sprintf("%s/", pendingKey)) {
				klog.Infof("key(%s) storage is pending, skip to store key(%s)", pendingKey, key)
				return false
			}
		} else {
			if strings.Contains(pendingKey, fmt.Sprintf("%s/", key)) {
				klog.Infof("key(%s) storage is pending, skip to store key(%s)", pendingKey, key)
				return false
			}
		}
	}
	ds.keyPendingStatus[key] = struct{}{}
	return true
}

func (ds *diskStorage) unLockKey(key string) {
	ds.Lock()
	defer ds.Unlock()
	delete(ds.keyPendingStatus, key)
}

func isTmpFile(path string) bool {
	_, file := filepath.Split(path)
	if strings.HasPrefix(file, tmpPrefix) {
		return true
	}
	return false
}

func getKey(tmpKey string) string {
	dir, file := filepath.Split(tmpKey)
	return filepath.Join(dir, strings.TrimPrefix(file, tmpPrefix))
}
