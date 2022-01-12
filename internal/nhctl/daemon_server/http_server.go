/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"net/http"
	_ "net/http/pprof"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/config_validate"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"runtime/debug"
	"strconv"
	"strings"
)

type ConfigSaveParams struct {
	Application string
	Kubeconfig  string
	Name        string // svcName
	Namespace   string
	Type        string // svcType
	Config      string // config content
}

type ConfigSaveResp struct {
	Success bool
	Message string
}

func startHttpServer() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Http Server occurs panic: %s", string(debug.Stack()))
		}
	}()
	log.Info("Starting http server")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Nocalhost http-server is working"))
	})

	http.HandleFunc("/config-save", handlingConfigSave)
	http.HandleFunc("/config-get", handlingConfigGet)

	err := http.ListenAndServe("127.0.0.1:"+strconv.Itoa(daemon_common.DaemonHttpPort), nil)
	if err != nil {
		log.ErrorE(err, "Http Server occur errors")
	}
}

func crossOriginFilter(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Max-Age", "300")
}

func handlingConfigSave(w http.ResponseWriter, r *http.Request) {
	crossOriginFilter(w)

	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}

	csp := &ConfigSaveParams{}
	err := r.ParseForm()
	if err != nil {
		fail(w, err.Error())
		return
	}

	if len(r.PostForm) == 0 {
		fail(w, "Post form can not be nil")
		return
	}

	key := "application"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Application = r.PostForm[key][0]

	key = "name"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Name = r.PostForm[key][0]

	key = "namespace"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Namespace = r.PostForm[key][0]

	key = "type"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Type = strings.ToLower(r.PostForm[key][0])

	key = "kubeconfig"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Kubeconfig = r.PostForm[key][0]

	key = "config"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Config = r.PostForm[key][0]

	bys, err := base64.StdEncoding.DecodeString(csp.Config)
	if err != nil {
		fail(w, err.Error())
		return
	}

	svcConfig := &profile.ServiceConfigV2{}
	err = yaml.Unmarshal(bys, svcConfig)
	if err != nil {
		fail(w, err.Error())
		return
	}
	svcConfig.Name = csp.Name
	svcConfig.Type = csp.Type

	client, err := clientgoutils.NewClientGoUtils(csp.Kubeconfig, csp.Namespace)
	if err != nil {
		fail(w, err.Error())
		return
	}
	// todo: by hxx
	devAction, _ := nocalhost.GetDevModeActionBySvcType(base.SvcType(csp.Type))
	containers, err := controller.GetOriginalContainers(client, base.SvcType(csp.Type), csp.Name, devAction.PodTemplatePath)
	if err != nil {
		fail(w, err.Error())
		return
	}

	config_validate.PrepareForConfigurationValidate(client, containers)
	if err := config_validate.Validate(svcConfig); err != nil {
		fail(w, err.Error())
		return
	}

	ot := svcConfig.Type
	svcConfig.Type = strings.ToLower(svcConfig.Type)
	if !nocalhost.CheckIfResourceTypeIsSupported(base.SvcType(svcConfig.Type)) {
		fail(w, fmt.Sprintf("Service Type %s is unsupported", ot))
		return
	}

	err = controller.UpdateSvcConfig(csp.Namespace, csp.Application, csp.Kubeconfig, svcConfig)
	if err != nil {
		fail(w, err.Error())
		return
	}
	success(w, "Service config has been update")
}

func handlingConfigGet(w http.ResponseWriter, r *http.Request) {
	crossOriginFilter(w)

	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}

	csp := &ConfigSaveParams{}
	err := r.ParseForm()
	if err != nil {
		fail(w, err.Error())
		return
	}

	if len(r.PostForm) == 0 {
		fail(w, "Post form can not be nil")
		return
	}

	key := "application"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Application = r.PostForm[key][0]

	key = "name"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Name = r.PostForm[key][0]

	key = "namespace"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Namespace = r.PostForm[key][0]

	key = "type"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Type = strings.ToLower(r.PostForm[key][0])

	key = "kubeconfig"
	if len(r.PostForm[key]) == 0 {
		fail(w, fmt.Sprintf("%s can not be nil", key))
		return
	}
	csp.Kubeconfig = r.PostForm[key][0]

	nhApp, err := app.NewApplication(csp.Application, csp.Namespace, csp.Kubeconfig, true)
	if err != nil {
		fail(w, err.Error())
		return
	}
	nhSvc, err := nhApp.Controller(csp.Name, base.SvcType(csp.Type))
	if err != nil {
		fail(w, err.Error())
		return
	}

	_ = nhSvc.LoadConfigFromHub()
	// need to load latest config
	_ = nhApp.ReloadSvcCfg(csp.Name, base.SvcType(csp.Type), false, true)
	nhSvc.ReloadConfig()

	c := nhSvc.Config()

	if len(c.ContainerConfigs) == 0 {
		if cs, err := nhSvc.GetOriginalContainers(); err != nil {
			log.LogE(err)
		} else {
			c.ContainerConfigs = make([]*profile.ContainerConfig, 0)
			for _, container := range cs {
				c.ContainerConfigs = append(c.ContainerConfigs, &profile.ContainerConfig{
					Name: container.Name,
				})
			}
		}
	}

	writeJsonResp(w, 200, c)
}

func success(w http.ResponseWriter, mes string) {
	c := &ConfigSaveResp{
		Success: true,
		Message: mes,
	}
	writeResp(w, 200, c)
}

func fail(w http.ResponseWriter, mes string) {
	c := &ConfigSaveResp{
		Success: false,
		Message: mes,
	}
	writeResp(w, 201, c)
}

func writeResp(w http.ResponseWriter, statusCode int, c *ConfigSaveResp) {
	//w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(c)
}

func writeJsonResp(w http.ResponseWriter, statusCode int, i interface{}) {
	//w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(i)
}
