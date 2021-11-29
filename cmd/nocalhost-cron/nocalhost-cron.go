package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net/http"
	"nocalhost/internal/nocalhost-api/model"

	"nocalhost/cmd/nocalhost-cron/jobs"
	"nocalhost/pkg/nocalhost-api/conf"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-cron/tools"
)

var (
	config = pflag.StringP("config", "c", "", "config file path.")
)

func main() {
	pflag.Parse()
	// init config
	if err := conf.Init(*config); err != nil {
		panic(err)
	}
	// init logger
	conf.InitLog()
	// init database
	model.Init()
	// migrate database
	model.MigrateDB()
	// init cron
	c := cron.New()
	// init jobs
	_, err := c.AddFunc(jobs.Sleep.Spec, func() {
		go jobs.Sleep.Task()
	})
	if err != nil {
		log.Errorf("Failed to start nocalhost-cron, err: %v", err)
		return
	}
	c.Start()
	log.Info("nocalhost-cron was started successfully.")

	go health()

	g := tools.Graceful{}
	g.Wait()
}

func health() {
	gin.SetMode(gin.ReleaseMode)
	app := gin.Default()
	app.GET("/health", func (c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"message": "nocalhost-cron is healthy",
		})
	})

	err := app.Run(viper.GetString("cron.addr"))
	if err != nil {
		log.Errorf("Failed to listen `/health`, err: %v", err)
		panic(fmt.Sprintf("Failed to listen `/health`, err: %v", err))
	}
}
