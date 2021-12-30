/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package utils

import (
	"testing"
)

func TestGetVClusterVersionList(t *testing.T) {
	type args struct {
		repo string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "get version",
			args: args{repo: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetVClusterVersionList(tt.args.repo)
			t.Log(got)
		})
	}
}
