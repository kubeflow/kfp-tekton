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

package db

import (
	"fmt"
	"time"

	"github.com/kubeflow/kfp-tekton/tekton-catalog/cache/pkg/model"
	"gorm.io/gorm"
)

type ConnectionParams struct {
	DbDriver            string
	DbHost              string
	DbPort              string
	DbName              string
	DbUser              string
	DbPwd               string
	DbGroupConcatMaxLen string
	DbExtraParams       string
	Timeout             time.Duration
}

func InitDBClient(params ConnectionParams, initConnectionTimeout time.Duration) (*gorm.DB, error) {
	driverName := params.DbDriver
	var db *gorm.DB
	var err error

	switch driverName {
	case "mysql":
		db, err = initMysql(params)
	case "sqlite":
		db, err = initSqlite(params.DbName)
	default:
		return nil, fmt.Errorf("driver %s is not supported", driverName)
	}

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	if err != nil {
		return nil, err
	}
	// Create table
	response := db.AutoMigrate(&model.TaskCache{})
	if response != nil {
		return nil, fmt.Errorf("failed to initialize the databases: Error: %v", response)
	}
	return db, nil
}
