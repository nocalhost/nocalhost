/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package utils

import (
	"math/rand"
	"reflect"
	"time"
)

// StringSliceReflectEqual
func StringSliceReflectEqual(a, b []string) bool {
	return reflect.DeepEqual(a, b)
}

// StringSliceEqual
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	b = b[:len(a)]
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

// SliceShuffle shuffle a slice
func SliceShuffle(slice []interface{}) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}

// Uint64SliceReverse
func Uint64SliceReverse(a []uint64) []uint64 {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

// StringSliceContains
func StringSliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// IsInSlice
func IsInSlice(value interface{}, sli interface{}) bool {
	switch reflect.TypeOf(sli).Kind() {
	case reflect.Slice, reflect.Array:
		s := reflect.ValueOf(sli)
		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(value, s.Index(i).Interface()) {
				return true
			}
		}
	}
	return false
}

// Uint64ShuffleSlice
func Uint64ShuffleSlice(a []uint64) []uint64 {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	return a
}

// see: https://yourbasic.org/golang/

func Uint64DeleteElemInSlice(i int, s []uint64) []uint64 {
	if i < 0 || i > len(s)-1 {
		return s
	}
	// Remove the element at index i from s.
	s[i] = s[len(s)-1] // Copy last element to index i.
	s[len(s)-1] = 0    // Erase last element (write zero value).
	s = s[:len(s)-1]   // Truncate slice.

	return s
}

// Uint64DeleteElemInSliceWithOrder

func Uint64DeleteElemInSliceWithOrder(i int, s []uint64) []uint64 {
	if i < 0 || i > len(s)-1 {
		return s
	}
	// Remove the element at index i from s.
	copy(s[i:], s[i+1:]) // Shift s[i+1:] left one index.
	s[len(s)-1] = 0      // Erase last element (write zero value).
	s = s[:len(s)-1]     // Truncate slice.

	return s
}
