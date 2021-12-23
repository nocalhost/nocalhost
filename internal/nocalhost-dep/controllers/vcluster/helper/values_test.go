/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"testing"
)

func Test_getDefaultValues(t *testing.T) {
	tests := []struct {
		name    string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get default values",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDefaultValues()
			if (err != nil) != tt.wantErr {
				t.Errorf("getDefaultValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(got)
		})
	}
}
