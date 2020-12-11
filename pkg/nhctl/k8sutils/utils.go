package k8sutils

import (
	"k8s.io/apimachinery/pkg/util/validation"
)

func ValidateDNS1123Name(name string) bool {
	errs := validation.IsDNS1123Subdomain(name)
	if len(errs) == 0 {
		return true
	} else {
		return false
	}
}
