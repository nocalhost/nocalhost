/**
 _   _                 _ _               _
| \ | |               | | |             | |
|  \| | ___   ___ __ _| | |__   ___  ___| |_
| . ` |/ _ \ / __/ _` | | '_ \ / _ \/ __| __|
| |\  | (_) | (_| (_| | | | | | (_) \__ \ |_
|_| \_|\___/ \___\__,_|_|_| |_|\___/|___/\__|

*
* @Author 王炜
*/
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	routers "nocalhost/pkg/nocalhost-api/app/router"
	"nocalhost/pkg/nocalhost-api/conf"
	"nocalhost/pkg/nocalhost-api/napp"
	v "nocalhost/pkg/nocalhost-api/pkg/version"
)

var (
	cfg     = pflag.StringP("config", "c", "", "config file path.")
	version = pflag.BoolP("version", "v", false, "show version info.")

	Svc *service.Service
)


// @title Nocalhost docs api
// @version 1.0
// @description Nocalhost demo

// @contact.name wangwei
// @contact.url codingcorp.coding.net
// @contact.email

// @host localhost:8080
// @BasePath /v1
func main() {
	pflag.Parse()
	if *version {
		ver := v.Get()
		marshaled, err := json.MarshalIndent(&ver, "", "  ")
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(marshaled))
		return
	}

	// init config
	if err := conf.Init(*cfg); err != nil {
		panic(err)
	}

	// init app
	napp.App = napp.New(conf.Conf)

	// Set gin mode.
	gin.SetMode(napp.ModeRelease)
	if viper.GetString("app.run_mode") == napp.ModeDebug {
		gin.SetMode(napp.ModeDebug)
		napp.App.DB.Debug()
	}

	// Create the Gin engine.
	router := napp.App.Router

	// Health 健康检查
	router.GET("/health", api.HealthCheck)
	// metrics prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API Routes.
	routers.Load(router)
	// 加载 router
	//routers.LoadWebRouter(router)

	// init service
	svc := service.New()

	// set global service
	service.Svc = svc

	// start grpc server 预留
	//go server.New(svc)

	// start server
	napp.App.Run()
}
