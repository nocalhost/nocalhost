package clientgoutils

import (
	"fmt"
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
	"testing"
	"time"
)

func TestClientGoUtils_CreateResource(t *testing.T) {
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube/admin-config"), time.Minute)
	Must(err)
	err = client.ApplyForCreate([]string{"/tmp/pre-install-cm.yaml"}, "demo10", true)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
}

func TestClientGoUtils_Exec(t *testing.T) {
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube/admin-config"), time.Minute)
	Must(err)
	err = client.Exec("demo10", "details-555cc5597f-gx5px", "", []string{"ls"})
	Must(err)
}
