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
	"time"

	"github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg/db"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg/model"
	"gorm.io/gorm"
)

type TaskCacheStore struct {
	db       *gorm.DB
	Disabled bool
	Params   db.ConnectionParams
}

func (t *TaskCacheStore) Connect() error {
	if t.db != nil || t.Disabled {
		return nil
	}
	var err error
	t.db, err = db.InitDBClient(t.Params, t.Params.Timeout)
	return err
}

func (t *TaskCacheStore) Get(taskHashKey string) (*model.TaskCache, error) {
	if t.Disabled || t.db == nil {
		return nil, nil
	}
	entry := &model.TaskCache{}
	d := t.db.Model(&model.TaskCache{}).Where("TaskHashKey = ?", taskHashKey).
		Order("CreatedAt DESC").First(entry)
	if d.Error != nil {
		return nil, fmt.Errorf("failed to get entry from cache: %q. Error: %v", taskHashKey, d.Error)
	}
	return entry, nil
}

func (t *TaskCacheStore) Put(entry *model.TaskCache) error {
	if t.Disabled || t.db == nil {
		return nil
	}
	d := t.db.Create(entry)
	if d.Error != nil {
		return fmt.Errorf("failed to create a new cache entry, %#v, Error: %v", entry, t.db.Error)
	}
	return nil
}

func (t *TaskCacheStore) Delete(id string) error {
	if t.Disabled || t.db == nil {
		return nil
	}
	d := t.db.Delete(&model.TaskCache{}, "ID = ?", id)
	if d.Error != nil {
		return d.Error
	}
	return nil
}

func (t *TaskCacheStore) PruneOlderThan(timestamp time.Time) error {
	if t.Disabled || t.db == nil {
		return nil
	}
	d := t.db.Delete(&model.TaskCache{}, "CreatedAt <= ?", timestamp)
	if d.Error != nil {
		return d.Error
	}
	return nil
}
