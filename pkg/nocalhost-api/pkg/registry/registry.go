/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package registry

import (
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func New(registryURL, user, password string) (*registry.Registry, error) {
	if strings.Contains(registryURL, "https://") {
		return NewSecure(registryURL, user, password)
	}

	return NewInsecure(registryURL, user, password)
}

func NewSecure(registryURL, username, password string) (*registry.Registry, error) {
	transport := http.DefaultTransport
	return newFromTransport(registryURL, username, password, transport, registryLog)
}

func NewInsecure(registryURL, username, password string) (*registry.Registry, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	return newFromTransport(registryURL, username, password, transport, registryLog)
}

func registryLog(format string, args ...interface{}) {
	log.Debug(format, args)
}

func newFromTransport(registryURL, username, password string, transport http.RoundTripper, logf registry.LogfCallback) (*registry.Registry, error) {
	url := strings.TrimSuffix(registryURL, "/")
	transport = registry.WrapTransport(transport, url, username, password)
	r := &registry.Registry{
		URL: url,
		Client: &http.Client{
			Transport: transport,
		},
		Logf: logf,
	}

	//if err := r.Ping(); err != nil {
	//	return nil, err
	//}
	return r, nil
}
