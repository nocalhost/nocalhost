/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package envsubst

import (
	"nocalhost/internal/nhctl/envsubst/parse"
	"nocalhost/internal/nhctl/fp"
	"os"
)

type RenderItem interface {
	GetContent() string
	GetLocation() string
}

// LocalFileRenderItem can parse the includation by rel path
// we should use LocalFileRenderItem if a config may contain any other config from local path
type LocalFileRenderItem struct {
	*fp.FilePathEnhance
}

func (lfri LocalFileRenderItem) GetContent() string {
	return lfri.ReadFile()
}

func (lfri LocalFileRenderItem) GetLocation() string {
	return lfri.Abs()
}

type TextRenderItem string

func (tri TextRenderItem) GetContent() string {
	return string(tri)
}

func (tri TextRenderItem) GetLocation() string {
	return ""
}

func Render(fp RenderItem, envFile *fp.FilePathEnhance) (string, error) {
	envs := [][]string{os.Environ()}
	envs = append(envs, envFile.ReadEnvFile()[:])

	return parse.New(
		"string", envs,
		&parse.Restrictions{NoUnset: false, NoEmpty: false},
	).
		Parse(fp.GetContent(), fp.GetLocation(), []string{})
}
