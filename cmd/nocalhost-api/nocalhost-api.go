/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

// nocalhost-api
package main

import (
	"encoding/json"
	"fmt"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nocalhost-api/app/api/v1/cluster"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	routers "nocalhost/pkg/nocalhost-api/app/router"
	"nocalhost/pkg/nocalhost-api/conf"
	"nocalhost/pkg/nocalhost-api/napp"
	v "nocalhost/pkg/nocalhost-api/pkg/version"
)

var GIT_COMMIT_SHA string

var (
	cfg     = pflag.StringP("config", "c", "", "config file path.")
	version = pflag.BoolP("version", "v", false, "show version info.")

	Svc *service.Service
)

// @title Nocalhost docs api
// @version 1.0
// @description Nocalhost server api

// @contact.name wangwei
// @contact.url nocalhost.coding.net
// @contact.email

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

	// Initial the Gin engine.
	router := napp.App.Router

	// Health Check
	router.GET("/health", api.HealthCheck)
	// metrics prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API Routes.
	routers.Load(router)
	// Load router
	//routers.LoadWebRouter(router)

	// init service
	service.Init()

	cluster.Init()

	service.StartJob()
	fmt.Printf("current run version %s, tag %s, branch %s \n", global.CommitId, global.Version, global.Branch)

	// start grpc server reserved
	//go server.New(controller)

	// start server
	napp.App.Run()
}
