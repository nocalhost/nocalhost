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

func Render(fp, envFile *fp.FilePathEnhance) (string, error) {
	envs := [][]string{os.Environ()}
	envs = append(envs, envFile.ReadEnvFile()[:])

	return parse.New(
		"string", envs,
		&parse.Restrictions{NoUnset: false, NoEmpty: false},
	).
		Parse(fp.ReadFile(), fp.Abs(), []string{})
}
