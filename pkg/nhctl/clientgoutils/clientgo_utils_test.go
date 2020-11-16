package clientgoutils

import (
	"fmt"
	"testing"
	"time"
)

func TestNewClientGoUtils(t *testing.T) {
	//client, err := NewClientGoUtils("", time.Minute)
	client, err := NewClientGoUtils("/Users/xinxinhuang/.kube/macbook-xinxinhuang-config", time.Minute)
	if err != nil {
		panic(err)
	}
	n, b, err := client.ClientConfig.Namespace()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v %v \n", n, b)
}

func TestClientGoUtils_Delete(t *testing.T) {
	client, err := NewClientGoUtils("", time.Minute)
	Must(err)
	client.Delete("/Users/xinxinhuang/.nhctl/application/gggg/manifest/templates/ratings.yaml", "demo30")
}
