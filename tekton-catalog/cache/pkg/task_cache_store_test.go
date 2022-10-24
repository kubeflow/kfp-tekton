// Copyright 2022 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg/db"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg/model"
)

func newTestingCacheStore(disabled bool) (*TaskCacheStore, error) {
	t := TaskCacheStore{
		Disabled: disabled,
		// Params: db.ConnectionParams{DbDriver: "mysql", DbName: "testdb",
		// 	DbHost: "127.0.0.1", DbPort: "3306", DbPwd: "", DbUser: "root",
		// 	Timeout: 10 * time.Second,
		// },
		Params: db.ConnectionParams{DbDriver: "sqlite", DbName: ":memory:"},
	}
	err := t.Connect()
	return &t, err
}

func createTaskCache(cacheKey string, cacheOutput string) *model.TaskCache {
	return &model.TaskCache{
		TaskHashKey: cacheKey,
		TaskOutput:  cacheOutput,
	}
}

func TestPut(t *testing.T) {
	taskCacheStore, err := newTestingCacheStore(false)
	if err != nil {
		t.Fatal(err)
	}
	entry := createTaskCache("x", "y")
	err = taskCacheStore.Put(entry)
	if err != nil {
		t.Fatal(err)
	}

}

func TestGet(t *testing.T) {
	taskCacheStore, err := newTestingCacheStore(false)
	if err != nil {
		t.Fatal(err)
	}
	entry := createTaskCache("x", "y")
	err = taskCacheStore.Put(entry)
	if err != nil {
		t.Fatal(err)
	}
	cacheResult, err := taskCacheStore.Get(entry.TaskHashKey)
	if err != nil {
		t.Error(err)
	}
	if cacheResult.TaskHashKey != entry.TaskHashKey {
		t.Errorf("Mismatached key. Expected %s Found: %s", entry.TaskHashKey,
			cacheResult.TaskHashKey)
	}
	if cacheResult.TaskOutput != entry.TaskOutput {
		t.Errorf("Mismatached output. Expected : %s Found: %s",
			entry.TaskOutput,
			cacheResult.TaskOutput)
	}
}

// Get should get the latest entry each time.
func TestGetLatest(t *testing.T) {
	taskCacheStore, err := newTestingCacheStore(false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 10; i++ {
		entry := createTaskCache("x", fmt.Sprintf("y%d", i))
		err := taskCacheStore.Put(entry)
		if err != nil {
			t.Fatal(err)
		}
		cacheResult, err := taskCacheStore.Get(entry.TaskHashKey)
		if err != nil {
			t.Error(err)
		}
		if cacheResult.TaskHashKey != entry.TaskHashKey {
			t.Errorf("Mismatached key. Expected %s Found: %s", entry.TaskHashKey,
				cacheResult.TaskHashKey)
		}
		if cacheResult.TaskOutput != entry.TaskOutput {
			t.Errorf("Mismatached output. Expected : %s Found: %s",
				entry.TaskOutput,
				cacheResult.TaskOutput)
		}
	}
}

func TestDisabledCache(t *testing.T) {
	taskCacheStore, err := newTestingCacheStore(true)
	if err != nil {
		t.Fatal(err)
	}
	taskCache, err := taskCacheStore.Get("random")
	if err != nil {
		t.Errorf("a disabled cache returned non nil error: %s", err)
	}
	if taskCache != nil {
		t.Errorf("a disabled cache should return nil")
	}
}

func TestPruneOlderThan(t *testing.T) {
	taskCacheStore, err := newTestingCacheStore(false)
	if err != nil {
		t.Fatal(err)
	}
	hashKey := "cacheKey"
	for i := 1; i < 10000000; i *= 100 {
		t1 := &model.TaskCache{
			TaskHashKey: hashKey,
			TaskOutput:  "cacheOutput",
			CreatedAt:   time.UnixMicro(int64(i * 100)),
		}
		err = taskCacheStore.Put(t1)
		if err != nil {
			t.Fatal(err)
		}
	}
	taskCache, err := taskCacheStore.Get(hashKey)
	if err != nil {
		t.Error(err)
	}
	if taskCache == nil {
		t.Error("TaskCache should be not nil.")
	}
	err = taskCacheStore.PruneOlderThan(time.UnixMicro(100000000))
	if err != nil {
		t.Fatal(err)
	}
	_, err = taskCacheStore.Get(hashKey)
	if err == nil {
		t.Errorf("Expected error to be not nil")
	}
	if !strings.HasPrefix(err.Error(), "failed to get entry from cache") {
		t.Error("Should fail with entry not found in cache.")
	}
}
