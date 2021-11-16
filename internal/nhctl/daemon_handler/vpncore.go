package daemon_handler

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"sync"
)

// VPNCore for reverse some resource, connect disconnect reconnect
type VPNCore struct {
	// key value
	sets.String
	lock *sync.RWMutex
}

type key string

func toKey(resourceName, resourceType, app, nid, ns string) key {
	return key(fmt.Sprintf("%s-%s-%s-%s-%s", ns, nid, app, resourceType, resourceName))
}

// VPNStatus for get vpn status, sync with secret
type VPNStatus struct {
}
