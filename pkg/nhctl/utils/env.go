/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
