package clientgoutils

import (
	"nocalhost/internal/nhctl/fp"
	"testing"
)

func TestAuth(t *testing.T) {

	if err := CheckForResource(
		fp.NewFilePath("/Users/anur/.kube/config").ReadFile(),
		"default", "secrets", "pods", "deployments",
	); err != nil {
		t.Error(err)
	}
}
