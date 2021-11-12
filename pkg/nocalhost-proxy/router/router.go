package router

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"nocalhost/internal/nocalhost-api/service/cluster"
	"nocalhost/internal/nocalhost-proxy/utils"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"regexp"
	"strconv"
	"time"
)

var (
	prefix, _ = regexp.Compile("/kubernetes/clusters/\\d+/")
)

func AttachTo(g *gin.Engine) {
	g.Any("/kubernetes/clusters/:id/api/:version/namespaces/:namespace/*rest", handler)
	g.NoRoute(handler)
}

func handler(c *gin.Context) {
	id := cast.ToUint64(c.Param("id"))
	cm, err := cluster.NewClusterService().GetCache(id)
	if err != nil {
		utils.Failure(c, err)
		return
	}

	kc, err := clientcmd.Load([]byte(cm.KubeConfig))
	if err != nil {
		utils.Failure(c, err)
		return
	}

	cx, ok := kc.Contexts[kc.CurrentContext]
	if !ok {
		utils.Failure(c, errors.New("cannot find current context"))
		return
	}

	cs, ok := kc.Clusters[cx.Cluster]
	if !ok {
		utils.Failure(c, errors.New("cannot find current cluster"))
		return
	}

	// kubectl does not send `Authorization` header over plain HTTP
	// https://github.com/kubernetes/kubectl/issues/744#issuecomment-545757997
	target, err := url.Parse(cs.Server)
	if err != nil {
		utils.Failure(c, err)
		return
	}

	transport, err := utils.NewTransport(cs)
	if err != nil {
		utils.Failure(c, err)
		return
	}

	ns := c.Param("namespace")
	if len(ns) > 0 {
		go lastActivity(cm.KubeConfig, ns)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		// strip the path prefix
		path := prefix.ReplaceAllString(req.URL.Path, "/")
		req.URL.Path = path
		req.URL.RawPath = path
	}

	c.Request.Host = target.Host
	proxy.ServeHTTP(c.Writer, c.Request)
}

func lastActivity(config string, namespace string) {
	// init client-go
	client, err := clientgo.NewAdminGoClient([]byte(config))
	if err != nil {
		log.Printf("Failed to update `nocalhost.dev.sleep/last-activity`: %v\n\n", err)
		return
	}
	// update the annotations of namespace
	patch, _ := json.Marshal(map[string]map[string]map[string]string {
		"metadata": {
			"annotations": {
				"nocalhost.dev.sleep/last-activity": strconv.FormatInt(time.Now().Unix(), 10),
			},
		},
	})
	_, err = client.
		GetClientset().
		CoreV1().
		Namespaces().
		Patch(context.TODO(), namespace, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Printf("Failed to update `nocalhost.dev.sleep/last-activity`: %v\n\n", err)
		return
	}
}
