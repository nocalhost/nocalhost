/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package daemon_handler

type resourceInfo struct {
	string string
	Health HealthEnum
}

func (r resourceInfo) status() string {
	return r.Health.String()
}

type Set map[string]*resourceInfo

func NewSet(items ...*resourceInfo) Set {
	ss := Set{}
	ss.Insert(items...)
	return ss
}

func NewSetByKeys(items ...string) Set {
	ss := Set{}
	ss.InsertByKeys(items...)
	return ss
}

func (s Set) Insert(items ...*resourceInfo) Set {
	for _, item := range items {
		s[item.string] = item
	}
	return s
}

func (s Set) InsertByKeys(items ...string) Set {
	for _, item := range items {
		s[item] = &resourceInfo{
			string: item,
			Health: UnHealthy,
		}
	}
	return s
}

func (s Set) Delete(items ...*resourceInfo) Set {
	for _, item := range items {
		delete(s, item.string)
	}
	return s
}

func (s Set) DeleteByKeys(items ...string) Set {
	for _, item := range items {
		delete(s, item)
	}
	return s
}

func (s Set) Has(item *resourceInfo) bool {
	_, contained := s[item.string]
	return contained
}

func (s Set) HasKey(item string) bool {
	_, contained := s[item]
	return contained
}

func (s Set) Get(item string) *resourceInfo {
	if k, contained := s[item]; contained {
		return k
	}
	return &resourceInfo{
		string: item,
		Health: Unknown,
	}
}

func (s Set) List() []*resourceInfo {
	res := make([]*resourceInfo, 0, len(s))
	for _, v := range s {
		res = append(res, v)
	}
	return res
}

func (s Set) KeySet() []string {
	res := make([]string, 0, len(s))
	for k := range s {
		res = append(res, k)
	}
	return res
}

func (s Set) Len() int {
	return len(s)
}

func (s Set) ForEach(f func(k string, v *resourceInfo)) {
	for k, v := range s {
		f(k, v)
	}
}
