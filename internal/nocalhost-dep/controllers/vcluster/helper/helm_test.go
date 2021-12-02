/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"testing"

	"k8s.io/client-go/rest"
)

func TestClient_LoadChart(t *testing.T) {
	c, err := NewClient(&rest.Config{}, "")
	if err != nil {
		t.Error(err)
	}
	type args struct {
		chartRef string
		repo     string
	}
	tests := []struct {
		name    string
		fields  *Client
		args    args
		want    string
		wantErr bool
	}{
		{
			name:   "download",
			fields: c,
			args: args{
				chartRef: "nocalhost",
				repo:     "https://nocalhost-helm.pkg.coding.net/nocalhost/nocalhost",
			},
			want: "nocalhost",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ChartPathOptions: tt.fields.ChartPathOptions,
				Config:           tt.fields.Config,
				Settings:         tt.fields.Settings,
			}
			got, err := c.loadChart(tt.args.chartRef, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadChart() error = %+v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Name() != tt.want {
				t.Errorf("loadChart() get.Name = %v, want %v", got.Name(), tt.want)
				return
			}
		})
	}
}
