package clientgoutils

import (
	"fmt"
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
	"testing"
	"time"
)

func TestNewClientGoUtils(t *testing.T) {

}

func TestClientGoUtils_Delete(t *testing.T) {
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube/admin-config"), time.Minute)
	Must(err)
	err = client.Delete("/tmp/pre-install-cm.yaml", "unit-test")
	if err != nil {
		panic(err)
	}
}

func TestClientGoUtils_Create(t *testing.T) {
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube/admin-config"), time.Minute)
	if err != nil {
		panic(err)
	}
	err = client.Create("/tmp/pre-install-cm.yaml", "unit-test", false, true)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	//s := runtime.Scheme{}
	//s.New()
}
