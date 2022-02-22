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
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/go-sql-driver/mysql"
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

func initMysql(params ConnectionParams, initConnectionTimeout time.Duration) (string, error) {
	var mysqlExtraParams = map[string]string{}
	data := []byte(params.DbExtraParams)
	_ = json.Unmarshal(data, &mysqlExtraParams)
	mysqlConfig := CreateMySQLConfig(
		params.DbUser,
		params.DbPwd,
		params.DbHost,
		params.DbPort,
		"",
		params.DbGroupConcatMaxLen,
		mysqlExtraParams,
	)

	var db *sql.DB
	var err error
	var operation = func() error {
		db, err = sql.Open(params.DbDriver, mysqlConfig.FormatDSN())
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)
	if err != nil {
		return "", err
	}
	defer db.Close()

	// Create database if not exist
	dbName := params.DbName
	operation = func() error {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
		if err != nil {
			return err
		}
		return nil
	}
	b = backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	operation = func() error {
		_, err = db.Exec(fmt.Sprintf("USE %s", dbName))
		if err != nil {
			return err
		}
		return nil
	}
	b = backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	mysqlConfig.DBName = dbName
	// Config reference: https://github.com/go-sql-driver/mysql#clientfoundrows
	mysqlConfig.ClientFoundRows = true
	return mysqlConfig.FormatDSN(), nil
}

func CreateMySQLConfig(user, password, mysqlServiceHost, mysqlServicePort, dbName, mysqlGroupConcatMaxLen string,
	mysqlExtraParams map[string]string) *mysql.Config {

	params := map[string]string{
		"charset":              "utf8",
		"parseTime":            "True",
		"loc":                  "Local",
		"group_concat_max_len": mysqlGroupConcatMaxLen,
	}

	for k, v := range mysqlExtraParams {
		params[k] = v
	}

	return &mysql.Config{
		User:                 user,
		Passwd:               password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", mysqlServiceHost, mysqlServicePort),
		Params:               params,
		DBName:               dbName,
		AllowNativePasswords: true,
	}
}
