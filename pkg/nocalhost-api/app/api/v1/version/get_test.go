/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package version

import (
	"fmt"
	"testing"

	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func Test_getVersionInfo(t *testing.T) {
	log.NewLogger(&log.Config{}, log.InstanceZapLogger)
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name: "get nocalhost version info from docker registry",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := getVersionInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("getVersionInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(v)
		})
	}
}
