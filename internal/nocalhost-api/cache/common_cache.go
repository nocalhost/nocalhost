package cache

import (
	"github.com/muesli/cache2go"
)

const (
	CLUSTER      CacheModule = "CLUSTER"
	USER         CacheModule = "USER"
	CLUSTER_USER CacheModule = "CLUSTER_USER"
)

type CacheModule string

func Module(cacheModule CacheModule) *cache2go.CacheTable {
	return cache2go.Cache(string(cacheModule))
}
