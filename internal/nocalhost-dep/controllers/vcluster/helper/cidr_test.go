/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func Test_getCIDR(t *testing.T) {
	c, err := config.GetConfig()
	if err != nil {
		t.Error(err)
	}

	type args struct {
		config    *rest.Config
		namespace string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get cidr form k8s",
			args: args{
				config:    c,
				namespace: "default",
			},
			want: "172.18.252.0/22",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCIDR(tt.args.config, tt.args.namespace); got != tt.want {
				t.Errorf("getCIDR() = %v, want %v", got, tt.want)
			}
		})
	}
}
