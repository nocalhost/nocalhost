/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

// nocalhost-dep Based on webhook-admission
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"net/http"
	"nocalhost/internal/nocalhost-dep/controllers"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
	"nocalhost/internal/nocalhost-dep/webhook"
)

var (
	GIT_COMMIT_SHA string
	scheme         = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(helmv1alpha1.AddToScheme(scheme))
}

func main() {
	var parameters webhook.WhSvrParameters
	var enableLeaderElection bool

	// get command line parameters
	flag.IntVar(&parameters.Port, "port", 8443, "Webhook server port.")
	flag.StringVar(
		&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem",
		"File containing the x509 Certificate for HTTPS.",
	)
	flag.StringVar(
		&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem",
		"File containing the x509 private key to --tlsCertFile.",
	)
	flag.StringVar(
		&parameters.SidecarCfgFile, "sidecarCfgFile", "/etc/webhook/config/sidecarconfig.yaml",
		"File containing the mutation configuration.",
	)
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	glog.Infof("Current Version :[%s]", GIT_COMMIT_SHA)

	sidecarConfig, err := webhook.LoadConfig(parameters.SidecarCfgFile)
	if err != nil {
		glog.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	pair, err := tls.LoadX509KeyPair(parameters.CertFile, parameters.KeyFile)
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
		os.Exit(1)
	}

	whsvr := &webhook.WebhookServer{
		SidecarConfig: sidecarConfig,
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.Serve)
	whsvr.Server.Handler = timer(mux)

	// create manager
	glog.Info("creating manager")
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "controller.nocalhost.dev",
	})
	if err != nil {
		glog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// create controller
	glog.Info("creating controller")
	if err := controllers.Setup(mgr); err != nil {
		glog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	// start webhook server in new rountine
	go func() {
		glog.Info("starting webhook")
		if err := whsvr.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
			//os.Exit(1)
		}
	}()

	glog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		glog.Error(err, "problem running manager")
		os.Exit(1)
	}

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	whsvr.Server.Shutdown(context.Background())
}

func timer(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			//startTime := time.Now()
			h.ServeHTTP(w, r)
			//duration := time.Now(/**/).Sub(startTime)
			//glog.Infof("total cost time %d", duration.Milliseconds())

		},
	)
}
