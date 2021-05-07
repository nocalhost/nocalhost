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

// nocalhost-api
package main

import (
	"encoding/json"
	"fmt"
	"nocalhost/internal/nocalhost-api/global"
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
// @contact.url codingcorp.coding.net
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
	svc := service.New()

	// set global service
	service.Svc = svc

	fmt.Printf("current run version %s, tag %s, branch %s \n", global.CommitId, global.Version, global.Branch)

	// start grpc server reserved
	//go server.New(controller)

	// start server
	napp.App.Run()
}
