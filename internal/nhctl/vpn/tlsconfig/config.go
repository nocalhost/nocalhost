/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package tlsconfig

import (
	"crypto/tls"
	"embed"
	log "github.com/sirupsen/logrus"
)

//go:embed server.crt
var crt embed.FS

//go:embed server.key
var key embed.FS

var Server *tls.Config
var Client *tls.Config

func init() {
	crtBytes, _ := crt.ReadFile("server.crt")
	keyBytes, _ := key.ReadFile("server.key")
	pair, err := tls.X509KeyPair(crtBytes, keyBytes)
	if err != nil {
		log.Fatal(err)
	}
	Server = &tls.Config{
		Certificates: []tls.Certificate{pair},
	}

	Client = &tls.Config{
		Certificates:       []tls.Certificate{pair},
		InsecureSkipVerify: true,
	}
}
