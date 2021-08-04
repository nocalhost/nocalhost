/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/resource"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
)

// ResourceList provides convenience methods for comparing collections of Infos.
type ResourceList []*resource.Info

// Append adds an Info to the Result.
func (r *ResourceList) Append(val *resource.Info) {
	*r = append(*r, val)
}

// Visit implements resource.Visitor.
func (r ResourceList) Visits(fns []resource.VisitorFunc) error {
	for _, i := range r {
		for _, fn := range fns {
			if err := fn(i, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// Visit implements resource.Visitor.
func (r ResourceList) Visit(fn resource.VisitorFunc) error {
	for _, i := range r {
		if err := fn(i, nil); err != nil {
			return err
		}
	}
	return nil
}

// Filter returns a new Result with Infos that satisfy the predicate fn.
func (r ResourceList) Filter(fn func(*resource.Info) bool) ResourceList {
	var result ResourceList
	for _, i := range r {
		if fn(i) {
			result.Append(i)
		}
	}
	return result
}

// Get returns the Info from the result that matches the name and kind.
func (r ResourceList) Get(info *resource.Info) *resource.Info {
	for _, i := range r {
		if isMatchingInfo(i, info) {
			return i
		}
	}
	return nil
}

// Contains checks to see if an object exists.
func (r ResourceList) Contains(info *resource.Info) bool {
	for _, i := range r {
		if isMatchingInfo(i, info) {
			return true
		}
	}
	return false
}

// Difference will return a new Result with objects not contained in rs.
func (r ResourceList) Difference(rs ResourceList) ResourceList {
	return r.Filter(
		func(info *resource.Info) bool {
			return !rs.Contains(info)
		},
	)
}

// Intersect will return a new Result with objects contained in both Results.
func (r ResourceList) Intersect(rs ResourceList) ResourceList {
	return r.Filter(rs.Contains)
}

// isMatchingInfo returns true if infos match on Name and GroupVersionKind.
func isMatchingInfo(a, b *resource.Info) bool {
	return a.Name == b.Name && a.Namespace == b.Namespace &&
		a.Mapping.GroupVersionKind.Kind == b.Mapping.GroupVersionKind.Kind
}

func LoadValidManifest(path []string, ignorePaths ...[]string) []string {
	var ignorePath []string
	for _, ignorePathItem := range ignorePaths {
		ignorePath = append(ignorePath, ignorePathItem...)
	}

	result := make([]string, 0)
	resourcePaths := path
	for _, eachPath := range resourcePaths {
		files, _, err := GetYamlFilesAndDirs(eachPath, ignorePath)
		if err != nil {
			log.WarnE(err, fmt.Sprintf("Fail to load manifest in %s", eachPath))
			continue
		}

		for _, file := range files {
			if _, err2 := os.Stat(file); err2 != nil {
				log.WarnE(errors.Wrap(err2, ""), fmt.Sprintf("%s can not be installed", file))
				continue
			}
			result = append(result, file)
		}
	}

	return result
}

// Path can be a file or a dir
func GetYamlFilesAndDirs(path string, ignorePaths []string) ([]string, []string, error) {

	if isFileIgnored(path, ignorePaths) {
		log.Infof("Ignoring file: %s", path)
		return nil, nil, nil
	}

	dirs := make([]string, 0)
	files := make([]string, 0)
	var err error
	stat, err := os.Stat(path)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}

	// If path is a file, return it directly
	if !stat.IsDir() {
		return append(files, path), append(dirs, path), nil
	}
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, nil, err
	}

	for _, fi := range dir {
		fPath := filepath.Join(path, fi.Name())
		if isFileIgnored(fPath, ignorePaths) {
			log.Logf("Ignoring file: %s", fPath)
			continue
		}
		if fi.IsDir() {
			dirs = append(dirs, fPath)
			fs, ds, err := GetYamlFilesAndDirs(fPath, ignorePaths)
			if err != nil {
				return files, dirs, err
			}
			dirs = append(dirs, ds...)
			files = append(files, fs...)
		} else if strings.HasSuffix(fi.Name(), ".yaml") || strings.HasSuffix(fi.Name(), ".yml") {
			files = append(files, fPath)
		}
	}
	return files, dirs, nil
}

func isFileIgnored(fileName string, ignorePaths []string) bool {
	for _, iFile := range ignorePaths {
		if iFile == fileName {
			return true
		}
	}
	return false
}
