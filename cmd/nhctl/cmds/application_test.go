package cmds

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"nocalhost/pkg/nhctl/utils"
	"testing"
)

func TestApplication_StopAllPortForward(t *testing.T) {
	application, err := NewApplication("eeee")
	if err != nil {
		printlnErr("fail to create application", err)
		return
	}

	err = application.StopAllPortForward()
	if err != nil {
		printlnErr("fail to stop port-forward", err)
	}
}

func TestForTest(t *testing.T) {
	map2 := v1.ConfigMap{}
	map2.Name = "nocalhost-depends-do-not-overwrite-"
	//map2.Data
}

func TestGetNameSpaces(t *testing.T) {
	fbytes, err := ioutil.ReadFile("/Users/xinxinhuang/.kube/macbook-xinxinhuang-config")
	utils.Mush(err)
	NewNonInteractiveDeferredLoadingClientConfig(
		&ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterUrl}}).ClientConfig()

	err = yaml.Unmarshal(fbytes, configMap)
	utils.Mush(err)
	fmt.Printf("%v\n", configMap.Data)
}
