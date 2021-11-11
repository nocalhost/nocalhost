package router

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"nocalhost/internal/nocalhost-api/service"
	yaml "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
	"nocalhost/pkg/nocalhost-api/app/api/v1/cluster"
	"regexp"
	"time"
)

var (
	prefix, _ = regexp.Compile("/kubernetes/clusters/\\d+/")
)

func Attach(g *gin.Engine) {
	g.Any("/kubernetes/clusters/:id/api/:version/namespaces/:namespace/*rest", handler)
	g.NoRoute(handler)
}

func handler(c *gin.Context) {
	id := cast.ToUint64(c.Param("id"))
	cm, err := service.Svc.ClusterSvc().GetCache(id)
	if err != nil {
		failure(c, err)
		return
	}

	kc := &cluster.KubeConfig{}
	err = yaml.Unmarshal([]byte(cm.KubeConfig), kc)
	if err != nil {
		failure(c, err)
		return
	}

	var cs *cluster.Cluster = nil
	for _, v := range kc.Clusters {
		if v.Name == cm.Name {
			cs = &v.Cluster
			break
		}
	}

	if cs == nil {
		failure(c, errors.New("cannot find the cluster"))
		return
	}

	// kubectl does not send `Authorization` header over plain HTTP
	// https://github.com/kubernetes/kubectl/issues/744#issuecomment-545757997
	uri, err := url.Parse(cs.Server)
	if err != nil {
		failure(c, err)
		return
	}

	transport, err := newTransport(cs)
	if err != nil {
		failure(c, err)
		return
	}

	ns := c.Param("namespace")
	if len(ns) > 0 {
		// TODO: update the annotations of namespace
	}

	proxy := httputil.NewSingleHostReverseProxy(uri)
	proxy.Transport = transport
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		// strip prefix
		path := prefix.ReplaceAllString(req.URL.Path, "/")
		req.URL.Path = path
		req.URL.RawPath = path
	}

	c.Request.Host = uri.Host
	proxy.ServeHTTP(c.Writer, c.Request)
}

func newTransport(cs *cluster.Cluster) (*http.Transport, error) {
	ca, err := base64.StdEncoding.DecodeString(cs.CertificateAuthorityData)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(ca)
	if !ok {
		return nil, errors.New("failed to parse CA")
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

type Response struct {
	Status		string	`json:"status"`
	Message		string	`json:"message"`
}

func failure(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, Response{
		Message: err.Error(),
		Status:  "Failure",
	})
}
