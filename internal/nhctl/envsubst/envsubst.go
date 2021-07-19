/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
