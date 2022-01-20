/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package registry

import (
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"testing"
)

func TestNew(t *testing.T) {
	log.NewLogger(&log.Config{}, log.InstanceZapLogger)
	type args struct {
		repository  string
		registryURL string
		user        string
		password    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "get tags from coding",
			args: args{
				registryURL: "https://nocalhost-docker.pkg.coding.net",
				repository:  "nocalhost/public/nocalhost-api",
				user:        "",
				password:    "",
			},
		},
		{
			name: "get tag from docker hub",
			args: args{
				registryURL: "https://registry-1.docker.io/",
				repository:  "library/nginx",
				user:        "",
				password:    "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub, err := New(tt.args.registryURL, tt.args.user, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tags, err := hub.Tags(tt.args.repository)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(tags)
		})
	}
}
