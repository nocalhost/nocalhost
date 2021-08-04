/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package envsubst

import (
	"bufio"
	"nocalhost/internal/nhctl/envsubst/parse"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"
)

func Render(fp, envFile *fp.FilePathEnhance) (string, error) {
	envs := [][]string{os.Environ()}
	envs = append(envs, readEnvFile(envFile)[:])

	return parse.New("string", envs,
		&parse.Restrictions{NoUnset: false, NoEmpty: false}).
		Parse(fp.ReadFile(), fp.Abs(), []string{})
}

func readEnvFile(envFile *fp.FilePathEnhance) []string {
	var envFiles []string

	if envFile == nil {
		return envFiles
	}

	if err := envFile.CheckExist(); err != nil {
		return envFiles
	}

	file, err := os.Open(envFile.Abs())
	if err != nil {
		log.ErrorE(err, "Can't not reading env file from "+envFile.Path)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()
		if !strings.ContainsAny(text, "=") || strings.HasPrefix(text, "#") {
			continue
		}
		envFiles = append(envFiles, text)
	}

	return envFiles
}
