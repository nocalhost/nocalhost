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
	err = client.CreateResource([]string{"/tmp/pre-install-cm.yaml"}, "demo10")
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
}
