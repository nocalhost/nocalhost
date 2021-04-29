package resouce_cache

import (
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
	head     *cacheItem
	tail     *cacheItem
}

func (l *lru) Add(key string, i interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	item := l.cache[key]
	if item == nil {
		if l.size() >= l.capacity {
			l.remove()
		}
		d := &cacheItem{key: key, value: i, before: l.tail, after: nil}
		l.cache[key] = d
		l.afterAddNode(d)
	} else {
		item.value = &i
		l.afterAccessNode(item)
	}
}

func (l *lru) size() int32 {
	return int32(len(l.cache))
}

func (l *lru) Get(key string) (interface{}, bool) {
	l.lock.RLocker().Lock()
	defer l.lock.RLocker().Unlock()
	item, exist := l.cache[key]
	if exist && item != nil {
		l.afterAccessNode(item)
		return item.value, exist
	} else {
		return nil, false
	}
}

// todo have bug
func (l *lru) afterAccessNode(b *cacheItem) {
	a := b.before
	c := b.after
	if a == nil && c == nil { // means it's empty
		//l.tail = b
		//l.head = b
	} else if l.head == b && l.tail == b {

	} else if a == nil { // means b is head
		c.before = nil
		l.head = c

		l.tail.after = b
		b.before = l.tail
		b.after = nil
		l.tail = b
	} else if c == nil { // means b is tail
		//l.tail = b
	} else { // means it's in middle
		a.after = c
		a.before = nil
		c.before = a

		l.tail.after = b
		b.before = l.tail
		b.after = nil
		l.tail = b
	}
}

func (l *lru) afterDeleteNode(b *cacheItem) {
	a := b.before
	c := b.after
	if a == nil && c == nil { // means it's empty
		l.tail = nil
		l.head = nil
	} else if a == nil { // means b is head
		c.before = nil
		l.head = c
	} else if c == nil { // means b is tail
		a.after = nil
		l.tail = a
	} else { // means it's in middle
		a.after = c
		c.before = a
	}
}

func (l *lru) afterAddNode(b *cacheItem) {
	if l.head == nil && l.tail == nil {
		l.head = b
		l.tail = b
	} else {
		b.before = l.tail
		b.after = nil
		l.tail.after = b
		l.tail = b
	}
}

func (l *lru) Delete(key string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	item, ok := l.cache[key]
	if ok && item != nil {
		l.afterDeleteNode(item)
	}
	delete(l.cache, key)
}

func (l *lru) remove() {
	if l.head != nil {
		l.afterDeleteNode(l.head)
		key := l.head.key
		v := l.cache[key]
		delete(l.cache, key)
		for _, f := range l.evict {
			f(v.value)
		}
	}
}

type cacheItem struct {
	before *cacheItem
	after  *cacheItem
	key    string
	value  interface{}
}
