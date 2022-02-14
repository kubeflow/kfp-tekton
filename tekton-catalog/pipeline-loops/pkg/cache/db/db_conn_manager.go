// Copyright 2020 The Kubeflow Authors
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
	"flag"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"

	"github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/cache/model"
)

const (
	mysqlDBDriverDefault            = "mysql"
	mysqlDBHostDefault              = "mysql.kubeflow.svc.cluster.local"
	mysqlDBPortDefault              = "3306"
	mysqlDBGroupConcatMaxLenDefault = "4194304"
)

type DBConnectionParameters struct {
	DbDriver            string
	DbHost              string
	DbPort              string
	DbName              string
	DbUser              string
	DbPwd               string
	DbGroupConcatMaxLen string
	DbExtraParams       string
}

func (params *DBConnectionParameters) LoadDefaults() {
	flag.StringVar(&params.DbDriver, "db_driver", mysqlDBDriverDefault, "Database driver name, mysql is the default value")
	flag.StringVar(&params.DbHost, "db_host", mysqlDBHostDefault, "Database host name.")
	flag.StringVar(&params.DbPort, "db_port", mysqlDBPortDefault, "Database port number.")
	flag.StringVar(&params.DbName, "db_name", "cachedb", "Database name.")
	flag.StringVar(&params.DbUser, "db_user", "root", "Database user name.")
	flag.StringVar(&params.DbPwd, "db_password", "", "Database password.")
	flag.StringVar(&params.DbGroupConcatMaxLen, "db_group_concat_max_len", mysqlDBGroupConcatMaxLenDefault, "Database group concat max length.")
	flag.Parse()
}

func InitDBClient(params DBConnectionParameters, initConnectionTimeout time.Duration, log *zap.SugaredLogger) (*gorm.DB, error) {
	driverName := params.DbDriver
	var arg string
	var err error

	switch driverName {
	case mysqlDBDriverDefault:
		arg, err = initMysql(params, initConnectionTimeout, log)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("driver %v is not supported", driverName)
	}

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(driverName, arg)
	if err != nil {
		return nil, err
	}
	// Create table
	response := db.AutoMigrate(&model.TaskCache{})
	if response.Error != nil {
		return nil, fmt.Errorf("failed to initialize the databases: Error: %w", response.Error)
	}
	return db, nil
}
