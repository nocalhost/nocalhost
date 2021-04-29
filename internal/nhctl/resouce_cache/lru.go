package resouce_cache

import (
	"math"
	"sync"
)

func NewLRU(capacity int32, evict ...func(i interface{})) *lru {
	return &lru{
		capacity: capacity,
		evict:    evict,
		cache:    make(map[string]*cacheItem),
	}
}

type lru struct {
	capacity int32
	cache    map[string]*cacheItem
	evict    []func(i interface{})
	lock     sync.RWMutex
}

func (l *lru) Add(key string, i interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	item := l.cache[key]
	if item == nil {
		if l.size() >= l.capacity {
			l.remove()
		}
		l.cache[key] = &cacheItem{value: i, times: 1}
	} else {
		item.times++
		item.value = &i
	}
}

func (l *lru) size() int32 {
	return int32(len(l.cache))
}

func (l *lru) Get(key string) (interface{}, bool) {
	l.lock.RLocker().Lock()
	defer l.lock.RLocker().Unlock()
	d, exist := l.cache[key]
	if exist && d != nil {
		d.times++
		return d.value, exist
	} else {
		return nil, false
	}
}

func (l *lru) Delete(key string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	delete(l.cache, key)
}

func (l *lru) remove() {
	var key string
	var times = int32(math.MaxInt32)
	for k, item := range l.cache {
		if item.times < times {
			times = item.times
			key = k
		}
	}
	v := l.cache[key]
	delete(l.cache, key)
	for _, f := range l.evict {
		f(v.value)
	}
}

type cacheItem struct {
	key   string
	value interface{}
	times int32
}
