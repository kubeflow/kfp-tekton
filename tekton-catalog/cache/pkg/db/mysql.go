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
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	mysqlDBDriverDefault            = "mysql"
	mysqlDBHostDefault              = "mysql.kubeflow.svc.cluster.local"
	mysqlDBPortDefault              = "3306"
	mysqlDBGroupConcatMaxLenDefault = "4194304"
	DefaultConnectionTimeout        = time.Minute * 6
)

func setDefault(field *string, defaultVal string) {
	if *field == "" {
		*field = defaultVal
	}
}

func (params *ConnectionParams) LoadMySQLDefaults() {
	setDefault(&params.DbDriver, mysqlDBDriverDefault)
	setDefault(&params.DbHost, mysqlDBHostDefault)
	setDefault(&params.DbPort, mysqlDBPortDefault)
	setDefault(&params.DbName, "cachedb")
	setDefault(&params.DbUser, "root")
	setDefault(&params.DbPwd, "")
	setDefault(&params.DbGroupConcatMaxLen, mysqlDBGroupConcatMaxLenDefault)
	if params.Timeout == 0 {
		params.Timeout = DefaultConnectionTimeout
	}
}

func initMysql(params ConnectionParams) (*gorm.DB, error) {
	var mysqlExtraParams = map[string]string{}
	data := []byte(params.DbExtraParams)
	_ = json.Unmarshal(data, &mysqlExtraParams)
	mysqlConfigDSN := CreateMySQLConfigDSN(
		params.DbUser,
		params.DbPwd,
		params.DbHost,
		params.DbPort,
		params.DbName,
		params.DbGroupConcatMaxLen,
		mysqlExtraParams,
	)
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       mysqlConfigDSN, // data source name, refer https://github.com/go-sql-driver/mysql#dsn-data-source-name
		DefaultStringSize:         256,            // add default size for string fields, by default, will use db type `longtext` for fields without size, not a primary key, no index defined and don't have default values
		DontSupportRenameIndex:    true,           // drop & create index when rename index, rename index not supported before MySQL 5.7, MariaDB
		DontSupportRenameColumn:   true,           // use change when rename column, rename rename not supported before MySQL 8, MariaDB
		SkipInitializeWithVersion: false,          // smart configure based on used version
	}), &gorm.Config{})

	return db, err
}

func CreateMySQLConfigDSN(user, password, mysqlServiceHost, mysqlServicePort, dbName, mysqlGroupConcatMaxLen string,
	mysqlExtraParams map[string]string) string {

	if mysqlGroupConcatMaxLen == "" {
		mysqlGroupConcatMaxLen = "4194304"
	}
	params := map[string]string{
		"parseTime":            "True",
		"loc":                  "Local",
		"group_concat_max_len": mysqlGroupConcatMaxLen,
	}

	for k, v := range mysqlExtraParams {
		params[k] = v
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4", user, password, mysqlServiceHost, mysqlServicePort, dbName)

	for k, v := range params {
		dsn = fmt.Sprintf("%s&%s=%s", dsn, k, v)
	}
	return dsn
}
