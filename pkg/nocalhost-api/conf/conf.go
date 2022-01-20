/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package conf

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"nocalhost/pkg/nocalhost-api/pkg/log"
)

var (
	Conf *Config
)

// InitDir init config
func Init(confPath string) error {
	err := initConfig(confPath)
	if err != nil {
		return err
	}
	return nil
}

// initConfig init config from conf file
func initConfig(confPath string) error {
	if confPath != "" {
		viper.SetConfigFile(confPath)
	} else {
		viper.AddConfigPath("conf")
		viper.SetConfigName("config.local")
	}
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("napp")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil { // viper解析配置文件
		return errors.WithStack(err)
	}

	// parse to config struct
	err := viper.Unmarshal(&Conf)
	if err != nil {
		return err
	}

	watchConfig()

	return nil
}

// Monitor configuration file changes and hot load programs
func watchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infof("Config file changed: %s", e.Name)
	})
}

// Config global config
// include common and biz config
type Config struct {
	// common
	App   AppConfig
	Log   LogConfig
	MySQL MySQLConfig
	Redis RedisConfig
	Cache CacheConfig

	// here can add biz conf
}

// AppConfig
type AppConfig struct {
	Name      string
	RunMode   string
	Addr      string
	Url       string
	JwtSecret string
}

// LogConfig
type LogConfig struct {
	Writers          string
	LoggerLevel      string
	LoggerFile       string
	LoggerWarnFile   string
	LoggerErrorFile  string
	LogFormatText    bool
	LogRollingPolicy string
	LogRotateDate    int
	LogRotateSize    int
	LogBackupCount   int
}

// MySQLConfig
type MySQLConfig struct {
	Name            string
	Addr            string
	UserName        string
	Password        string
	ShowLog         bool
	MaxIdleConn     int
	MaxOpenConn     int
	ConnMaxLifeTime int
}

// RedisConfig
type RedisConfig struct {
	Addr         string
	Password     string
	Db           int
	DialTimeout  int
	ReadTimeout  int
	WriteTimeout int
	PoolSize     int
}

// CacheConfig
type CacheConfig struct {
	Driver string
	Prefix string
}

// init coloredoutput
func InitLog() {
	config := log.Config{
		Writers:          viper.GetString("log.writers"),
		LoggerLevel:      viper.GetString("log.logger_level"),
		LoggerFile:       viper.GetString("log.logger_file"),
		LoggerWarnFile:   viper.GetString("log.logger_warn_file"),
		LoggerErrorFile:  viper.GetString("log.logger_error_file"),
		LogFormatText:    viper.GetBool("log.log_format_text"),
		LogRollingPolicy: viper.GetString("log.log_rolling_policy"),
		LogRotateDate:    viper.GetInt("log.log_rotate_date"),
		LogRotateSize:    viper.GetInt("log.log_rotate_size"),
		LogBackupCount:   viper.GetInt("log.log_backup_count"),
	}
	err := log.NewLogger(&config, log.InstanceZapLogger)
	if err != nil {
		fmt.Printf("InitWithConfig err: %v", err)
	}
}
