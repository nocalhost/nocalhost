/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package napp

import (
	"context"
	"github.com/go-playground/validator/v10"
	"log"
	"net/http"
	"nocalhost/internal/nocalhost-api/validation"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/conf"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

const (
	// ModeDebug debug mode
	ModeDebug string = "debug"
	// ModeRelease release mode
	ModeRelease string = "release"
	// ModeTest test mode
	ModeTest string = "test"
)

// App is singleton
var App *Application

// Application a container for your application.
type Application struct {
	Conf        *conf.Config
	DB          *gorm.DB
	RedisClient *redis.Client
	Router      *gin.Engine
	Debug       bool
	Validate    *validator.Validate
}

// New create a app
func New(cfg *conf.Config) *Application {
	app := new(Application)

	// init db
	app.DB = model.Init()

	// migrate db
	model.MigrateDB()

	// init redis
	// app.RedisClient = redis2.InitDir()

	// init router
	// Set gin mode.
	gin.SetMode(gin.ReleaseMode)
	if viper.GetString("app.run_mode") == ModeDebug {
		gin.SetMode(ModeDebug)
		app.DB.Debug()
	}
	app.Router = gin.Default()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("timing", validation.Timing)
	}

	// init coloredoutput
	conf.InitLog()

	if viper.GetString("app.run_mode") == ModeDebug {
		app.Debug = true
	}

	//init validate
	app.Validate = validator.New()

	return app
}

// Run start a app
func (a *Application) Run() {
	log.Printf("Start to listening the incoming requests on http address: %s", viper.GetString("app.addr"))
	srv := &http.Server{
		Addr:    viper.GetString("app.addr"),
		Handler: a.Router,
	}
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err.Error())
		}
	}()

	gracefulStop(srv)
}

// gracefulStop 优雅退出
// 等待中断信号以超时 5 秒正常关闭服务器
// 官方说明：https://github.com/gin-gonic/gin#graceful-shutdown-or-restart
func gracefulStop(srv *http.Server) {
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}
