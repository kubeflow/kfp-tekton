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
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/cache/model"
	_ "github.com/mattn/go-sqlite3"
)

func NewTestingDb() (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("Could not create the GORM database: %v", err)
	}
	// Create tables
	db.AutoMigrate(&model.TaskCache{})

	return db, nil
}

func createTaskCache(cacheKey string, cacheOutput string) *model.TaskCache {
	return &model.TaskCache{
		TaskHashKey: cacheKey,
		TaskOutput:  cacheOutput,
	}
}

func TestPut(t *testing.T) {
	db, err := NewTestingDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	taskCacheStore := TaskCacheStore{db: db, Enabled: true}
	entry := createTaskCache("x", "y")
	taskCache, err := taskCacheStore.Put(entry)
	if err != nil {
		t.Fatal(err)
	}
	if taskCache.TaskHashKey != entry.TaskHashKey {
		t.Errorf("Mismatached key. Expected %s Found: %s", entry.TaskHashKey,
			taskCache.TaskHashKey)
	}
	if taskCache.TaskOutput != entry.TaskOutput {
		t.Errorf("Mismatached output. Expected : %s Found: %s",
			entry.TaskOutput,
			taskCache.TaskOutput)
	}
}

func TestGet(t *testing.T) {
	db, err := NewTestingDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	taskCacheStore := TaskCacheStore{db: db, Enabled: true}
	entry := createTaskCache("x", "y")
	taskCache, err := taskCacheStore.Put(entry)
	if err != nil {
		t.Fatal(err)
	}
	cacheResult, err := taskCacheStore.Get(taskCache.TaskHashKey)
	if err != nil {
		t.Fatal(err)
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
	db, err := NewTestingDb()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	taskCacheStore := TaskCacheStore{db: db, Enabled: true}
	for i := 1; i < 10; i++ {
		entry := createTaskCache("x", fmt.Sprintf("y%d", i))
		taskCache, err := taskCacheStore.Put(entry)
		if err != nil {
			t.Fatal(err)
		}
		cacheResult, err := taskCacheStore.Get(taskCache.TaskHashKey)
		if err != nil {
			t.Fatal(err)
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
