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
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/cenkalti/backoff"
	"go.uber.org/zap"
	"time"

	"github.com/go-sql-driver/mysql"
)

func initMysql(params DBConnectionParameters, initConnectionTimeout time.Duration, log *zap.SugaredLogger) (string, error) {

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
		log.Infof("Backing database for cache is created")
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

func CreateMySQLConfig(user, password string, mysqlServiceHost string,
	mysqlServicePort string, dbName string, mysqlGroupConcatMaxLen string, mysqlExtraParams map[string]string) *mysql.Config {

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
