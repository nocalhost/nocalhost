/*
* webhook admission 处理依赖
* @Author 王炜
 */
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	webhook "nocalhost/internal/nocalhost-dep/webhook"
	"github.com/golang/glog"
)

func main() {
	var parameters webhook.WhSvrParameters

	// get command line parameters
	flag.IntVar(&parameters.Port, "port", 8443, "Webhook server port.")
	flag.StringVar(&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.StringVar(&parameters.SidecarCfgFile, "sidecarCfgFile", "/etc/webhook/config/sidecarconfig.yaml", "File containing the mutation configuration.")
	flag.Parse()

	sidecarConfig, err := webhook.LoadConfig(parameters.SidecarCfgFile)
	if err != nil {
		glog.Errorf("Failed to load configuration: %v", err)
	}

	pair, err := tls.LoadX509KeyPair(parameters.CertFile, parameters.KeyFile)
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
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
	whsvr.Server.Handler = mux

	// start webhook server in new rountine
	go func() {
		if err := whsvr.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	whsvr.Server.Shutdown(context.Background())
}
