package k8sutils

import (
	"fmt"
	"testing"
)

func TestValidate(t *testing.T) {
	if ValidateDNS1123Name("-111-11") {
		fmt.Println("valid")
	} else {
		fmt.Println("invalid")
	}
}
