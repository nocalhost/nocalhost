package utils

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/gin-gonic/gin"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"net"
	"net/http"
	"time"
)

type Response struct {
	Status		string	`json:"status"`
	Message		string	`json:"message"`
}

func Failure(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, Response{
		Message: err.Error(),
		Status:  "Failure",
	})
}

func NewTransport(cs *clientcmdapi.Cluster) (*http.Transport, error) {
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cs.CertificateAuthorityData)
	if !ok {
		return nil, errors.New("failed to parse the certificate")
	}

	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs: pool,
		},
	}, nil
}
