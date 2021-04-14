/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"fmt"
	"time"

	"github.com/spf13/viper"

	// MySQL driver.
	"github.com/jinzhu/gorm"
	// GORM MySQL
	_ "github.com/jinzhu/gorm/dialects/mysql"

	"nocalhost/pkg/nocalhost-api/pkg/log"
)

var DB *gorm.DB

// InitDir
func Init() *gorm.DB {
	return openDB(viper.GetString("mysql.username"),
		viper.GetString("mysql.password"),
		viper.GetString("mysql.addr"),
		viper.GetString("mysql.name"))
}

// openDB
func openDB(username, password, addr, name string) *gorm.DB {
	config := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=%t&loc=%s",
		username,
		password,
		addr,
		name,
		true,
		//"Asia/Shanghai"),
		"Local")

	db, err := gorm.Open("mysql", config)
	if err != nil {
		log.Errorf("Database connection failed. Database name: %s, err: %+v", name, err)
		panic(err)
	}

	db.Set("gorm:table_options", "CHARSET=utf8mb4")

	// set for db connection
	db.LogMode(viper.GetBool("mysql.show_log"))
	// To set the maximum number of open connections, replace with 0 to indicate unlimited.
	// Setting the maximum number of connections can avoid too high concurrency leading
	// to too many connection errors when connecting to mysql.
	db.DB().SetMaxOpenConns(viper.GetInt("mysql.max_open_conn"))
	// Used to set the number of idle connections.
	// When the number of idle connections is set,
	// the opened connection can be placed in the pool for the next use.
	db.DB().SetMaxIdleConns(viper.GetInt("mysql.max_idle_conn"))
	db.DB().SetConnMaxLifetime(time.Minute * viper.GetDuration("mysql.conn_max_life_time"))

	DB = db

	return db
}

// GetDB
func GetDB() *gorm.DB {
	return DB
}

// MigrateDB
func MigrateDB() {
	DB.AutoMigrate(
		&ApplicationModel{},
		&ClusterModel{},
		&ClusterUserModel{},
		&PrePullModel{},
		&UserBaseModel{},
		&ApplicationUserModel{},
	)
}
