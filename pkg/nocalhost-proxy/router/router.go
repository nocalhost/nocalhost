package router

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"nocalhost/internal/nocalhost-api/service"
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

	kc, err := clientcmd.Load([]byte(cm.KubeConfig))
	if err != nil {
		failure(c, err)
		return
	}

	cx, ok := kc.Contexts[kc.CurrentContext]
	if !ok {
		failure(c, errors.New("cannot find current context"))
		return
	}

	cs, ok := kc.Clusters[cx.Cluster]
	if !ok {
		failure(c, errors.New("cannot find current cluster"))
		return
	}

	// kubectl does not send `Authorization` header over plain HTTP
	// https://github.com/kubernetes/kubectl/issues/744#issuecomment-545757997
	target, err := url.Parse(cs.Server)
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

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		// strip prefix
		path := prefix.ReplaceAllString(req.URL.Path, "/")
		req.URL.Path = path
		req.URL.RawPath = path
	}

	c.Request.Host = target.Host
	proxy.ServeHTTP(c.Writer, c.Request)
}

func newTransport(cs *clientcmdapi.Cluster) (*http.Transport, error) {
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cs.CertificateAuthorityData)
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
