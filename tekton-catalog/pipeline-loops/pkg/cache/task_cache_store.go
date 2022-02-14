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
	"github.com/jinzhu/gorm"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/cache/db"
	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/cache/model"
	"go.uber.org/zap"
	"time"
)

const (
	DefaultConnectionTimeout = time.Minute * 6
)

type TaskCacheStore struct {
	db      *gorm.DB
	Enabled bool
}

func (t *TaskCacheStore) Connect(params db.DBConnectionParameters, logger *zap.SugaredLogger) error {
	if t.db != nil {
		return nil
	}
	var err error
	t.db, err = db.InitDBClient(params, DefaultConnectionTimeout, logger)
	if err == nil {
		t.Enabled = true
	}
	return err
}

func (t *TaskCacheStore) Get(taskHashKey string) (*model.TaskCache, error) {
	if !t.Enabled {
		return nil, nil
	}
	entry := &model.TaskCache{}
	d := t.db.Table("task_caches").Where("TaskHashKey = ?", taskHashKey).
		Order("CreatedAt DESC").First(entry)
	if d.Error != nil {
		return nil, fmt.Errorf("failed to get entry from cache: %q. Error: %v", taskHashKey, d.Error)
	}
	return entry, nil
}

func (t *TaskCacheStore) Put(entry *model.TaskCache) (*model.TaskCache, error) {
	if !t.Enabled {
		return nil, nil
	}
	ok := t.db.NewRecord(entry)
	if !ok {
		return nil, fmt.Errorf("failed to create a new cache entry, %#v, Error: %v", entry, t.db.Error)
	}
	rowInsert := &model.TaskCache{}
	d := t.db.Create(entry).Scan(rowInsert)
	if d.Error != nil {
		return nil, d.Error
	}
	return rowInsert, nil
}

func (t *TaskCacheStore) Delete(id string) error {
	if !t.Enabled {
		return nil
	}
	d := t.db.Delete(&model.TaskCache{}, "ID = ?", id)
	if d.Error != nil {
		return d.Error
	}
	return nil
}

func (t *TaskCacheStore) PruneOlderThan(timestamp time.Time) error {
	if !t.Enabled {
		return nil
	}
	d := t.db.Delete(&model.TaskCache{}, "CreatedAt <= ?", timestamp)
	if d.Error != nil {
		return d.Error
	}
	return nil
}
