/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
)

// files is a file slice, dir is not suppored
func GetKVFromEnvFiles(files []string) map[string]string {

	result := make(map[string]string, 0)
	for _, file := range files {
		vip := viper.New()
		log.Infof("Loading env file: %s", file)
		_, err := os.Stat(file)
		if err != nil {
			log.WarnE(errors.Wrap(err, ""), fmt.Sprintf("Failed to read env file: %s", file))
			continue
		}
		vip.AddConfigPath(filepath.Dir(file))
		vip.SetConfigType("env")
		vip.SetConfigName(filepath.Base(file))
		utils.Should(errors.Wrap(vip.ReadInConfig(), "Failed to read env file"))
		for _, key := range vip.AllKeys() {
			result[key] = vip.GetString(key)
		}
	}

	return result
}
