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
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

type ConfigSaveParams struct {
	Application string
	Kubeconfig  string
	Name        string
	Namespace   string
	Type        string
	Config      string
}

type ConfigSaveResp struct {
	Success bool
	Message string
}

func startHttpServer() {
	log.Info("Starting http server")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Nocalhost http-server is working"))
	})

	http.HandleFunc("/config-save", handlingConfigSave)

	err := http.ListenAndServe(":30125", nil)
	if err != nil {
		log.ErrorE(err, "Http Server occur errors")
	}
}

func handlingConfigSave(w http.ResponseWriter, r *http.Request) {
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
	csp.Type = r.PostForm[key][0]

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

	err = controller.UpdateSvcConfig(csp.Namespace, csp.Application, csp.Kubeconfig, svcConfig)
	if err != nil {
		fail(w, err.Error())
		return
	}
	success(w, "Service config has been update")
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
	writeResp(w, 200, c)
}

func writeResp(w http.ResponseWriter, statusCode int, c *ConfigSaveResp) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(c)
}
