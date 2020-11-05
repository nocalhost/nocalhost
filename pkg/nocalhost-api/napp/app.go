/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package napp

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/conf"

	"github.com/gin-gonic/gin"
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
}

// New create a app
func New(cfg *conf.Config) *Application {
	app := new(Application)

	// init db
	app.DB = model.Init()

	// init redis
	// app.RedisClient = redis2.Init()

	// init router
	app.Router = gin.Default()

	// init log
	conf.InitLog()

	if viper.GetString("app.run_mode") == ModeDebug {
		app.Debug = true
	}

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
