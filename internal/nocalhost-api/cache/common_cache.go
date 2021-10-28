package cache

import (
	"github.com/muesli/cache2go"
	"time"
)

const (
	CLUSTER      CacheModule = "CLUSTER"
	USER         CacheModule = "USER"
	CLUSTER_USER CacheModule = "CLUSTER_USER"

	OUT_OF_DATE = time.Minute * 5
)

type CacheModule string

func Module(cacheModule CacheModule) *cache2go.CacheTable {
	return cache2go.Cache(string(cacheModule))
}
