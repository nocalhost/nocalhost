/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"bytes"
	"fmt"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/kustomize"
	"k8s.io/cli-runtime/pkg/resource"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/pkg/fs"
	"strings"
)

const (
	constSTDINstr = "STDIN"
)

type ResourceReader interface {
	LoadResource() (*Resource, error)
}

type Resource struct {
	resource []string
}

func NewResourceFromStr(manifest string) *Resource {
	return &Resource{strings.Split(manifest, "---")}
}

func (r Resource) Append(s string) {
	r.resource = append(r.resource, s)
}

func (r Resource) String() string {
	return strings.Join(r.resource, "\n---\n")
}

func (r *Resource) arr() []string {
	return r.resource
}

func (r *Resource) GetResourceInfo(c *ClientGoUtils, continueOnError bool) ([]*resource.Info, error) {
	return c.GetResourceInfoFromReader(bytes.NewBufferString(r.String()), continueOnError)
}

// == local file visitor -- Kustomize

func NewKustomizeResourceReader(path string) *kustomizeResourceReader {
	return &kustomizeResourceReader{
		path: path,
	}
}

type kustomizeResourceReader struct {
	path string
}

func (lrv *kustomizeResourceReader) LoadResource() (*Resource, error) {
	fSys := fs.MakeRealFS()
	var out bytes.Buffer
	err := kustomize.RunKustomizeBuild(&out, fSys, lrv.path)
	if err != nil {
		return &Resource{}, err
	}

	return NewResourceFromStr(string(out.Bytes())), nil
}

// == local file visitor -- Manifest

func NewManifestResourceReader(files []string) *manifestResourceReader {
	return &manifestResourceReader{
		files: files,
	}
}

type manifestResourceReader struct {
	files []string
}

func (lrv *manifestResourceReader) LoadResource() (*Resource, error) {
	var (
		manifests []string
		errInfo   string
		e         error
	)

	// todo supports Stdin & network
	for _, file := range lrv.files {
		content, err := loadByPath(file)
		if err != nil {
			errInfo += fmt.Sprintf("\n Err while loading: %s, err: %s", file, err)
		}
		manifests = append(manifests, content)
	}

	if errInfo != "" {
		e = fmt.Errorf(errInfo)
	}
	return &Resource{manifests}, e
}

func loadByPath(p string) (string, error) {
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("the path %q does not exist", p)
	}
	if err != nil {
		return "", fmt.Errorf("the path %q cannot be accessed: %v", p, err)
	}

	result, err := doLoadByPath(p)
	if err != nil {
		return "", fmt.Errorf("error reading %q: %v", p, err)
	}

	return result, nil
}

func doLoadByPath(paths string) (string, error) {

	result := ""
	err := filepath.Walk(paths, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}
		// Don't check extension if the filepath was passed explicitly
		if path != paths {
			return nil
		}

		var f *os.File
		if path == constSTDINstr {
			f = os.Stdin
		} else {
			var err error
			f, err = os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
		}

		// TODO: Consider adding a flag to force to UTF16, apparently some
		// Windows tools don't write the BOM
		utf16bom := unicode.BOMOverride(unicode.UTF8.NewDecoder())
		reader := transform.NewReader(f, utf16bom)

		d := yaml.NewYAMLOrJSONDecoder(reader, 4096)
		for {
			ext := runtime.RawExtension{}
			if err := d.Decode(&ext); err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("error parsing %s: %v", path, err)
			}
			// TODO: This needs to be able to handle object in other encodings and schemas.
			ext.Raw = bytes.TrimSpace(ext.Raw)
			if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
				continue
			}

			result += fmt.Sprintf("---\n%s\n", string(ext.Raw))
		}
	})

	if err != nil {
		return "", err
	}
	return result, nil
}

// == remote secret visitor
