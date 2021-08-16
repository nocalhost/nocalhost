/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import "sync"

type Set struct {
	inner sync.Map
}

func NewSet(keys ...string) *Set {
	set := &Set{
		sync.Map{},
	}

	for _, key := range keys {
		set.Put(key)
	}
	return set
}

func (s *Set) ToArray() []string {
	result := make([]string, 0)

	s.inner.Range(
		func(key, value interface{}) bool {
			result = append(result, key.(string))
			return true
		},
	)

	return result
}

func (s *Set) Put(key string) {
	s.inner.Store(key, "")
}

func (s *Set) Exist(key string) bool {
	_, ok := s.inner.Load(key)
	return ok
}
