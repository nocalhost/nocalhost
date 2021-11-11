package main

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/conf"
	"nocalhost/pkg/nocalhost-proxy/router"
)

var (
	config = pflag.StringP("config", "c", "", "config file path.")
)

func main() {
	// init config
	if err := conf.Init(*config); err != nil {
		panic(err)
	}
	// init service
	service.Svc = service.New()

	app := gin.Default()
	router.Attach(app)

	log.Printf("Start the kube-apiserver proxy on http address: %s", viper.GetString("proxy.addr"))
	_ = app.Run(viper.GetString("proxy.addr"))
}

