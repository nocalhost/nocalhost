package cron

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"nocalhost/internal/nocalhost-api/sleep"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func Start() {
	go func() {
		// init cron
		c := cron.New()
		// init jobs
		_, err := c.AddFunc(sleep.Job.Spec(), func() {
			go sleep.Job.Exec()
		})
		if err != nil {
			log.Errorf("Failed to start cron, err: %v", err)
			panic(fmt.Sprintf("Failed to start cron, err: %v", err))
		}
		c.Start()
		log.Info("Cron was started successfully.")

		g := Graceful{}
		g.Wait("Shutting cron...")
	}()
}
