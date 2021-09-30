/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package setupcluster

import (
	"reflect"
	"testing"

	"nocalhost/internal/nocalhost-api/model"
)

func TestMeshDevInfo_SortApps(t *testing.T) {
	type fields struct {
		BaseNamespace    string
		MeshDevNamespace string
		IsUpdateHeader   bool
		Header           model.Header
		Apps             []MeshDevApp
		resources        meshDevResources
	}
	tests := []struct {
		name   string
		want   MeshDevInfo
		fields fields
	}{
		{
			name: "sort",
			want: MeshDevInfo{
				Apps: []MeshDevApp{
					{
						Name: "bookinfo",
						Workloads: []MeshDevWorkload{
							{
								Name: "details",
								Kind: ConfigMap,
							},
							{
								Name: "productpage",
								Kind: ConfigMap,
							},
							{
								Name: "ratings",
								Kind: ConfigMap,
							},
							{
								Name: "details",
								Kind: Deployment,
							},
							{
								Name: "productpage",
								Kind: Deployment,
							},
							{
								Name: "reviews",
								Kind: Deployment,
							},
							{
								Name: "details",
								Kind: Secret,
							},
							{
								Name: "productpage",
								Kind: Secret,
							},
							{
								Name: "ratings",
								Kind: Secret,
							},
							{
								Name: "details",
								Kind: VirtualService,
							},
						},
					},
					{
						Name: "foo",
						Workloads: []MeshDevWorkload{
							{
								Name: "bar",
								Kind: ConfigMap,
							},
							{
								Name: "bar",
								Kind: Deployment,
							},
							{
								Name: "foo",
								Kind: Deployment,
							},
							{
								Name: "bar",
								Kind: Secret,
							},
							{
								Name: "foo",
								Kind: VirtualService,
							},
						},
					},
				},
			},
			fields: fields{
				Apps: []MeshDevApp{
					{
						Name: "foo",
						Workloads: []MeshDevWorkload{
							{
								Name: "bar",
								Kind: Deployment,
							},
							{
								Name: "foo",
								Kind: VirtualService,
							},
							{
								Name: "bar",
								Kind: Secret,
							},
							{
								Name: "bar",
								Kind: ConfigMap,
							},
							{
								Name: "foo",
								Kind: Deployment,
							},
						},
					},
					{
						Name: "bookinfo",
						Workloads: []MeshDevWorkload{
							{
								Name: "ratings",
								Kind: Secret,
							},
							{
								Name: "ratings",
								Kind: ConfigMap,
							},
							{
								Name: "productpage",
								Kind: Deployment,
							},
							{
								Name: "details",
								Kind: Deployment,
							},
							{
								Name: "details",
								Kind: Secret,
							},
							{
								Name: "reviews",
								Kind: Deployment,
							},
							{
								Name: "details",
								Kind: VirtualService,
							},
							{
								Name: "productpage",
								Kind: Secret,
							},
							{
								Name: "details",
								Kind: ConfigMap,
							},
							{
								Name: "productpage",
								Kind: ConfigMap,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := MeshDevInfo{
				Apps: tt.fields.Apps,
			}
			info.SortApps()
			if !reflect.DeepEqual(info, tt.want) {
				t.Errorf("sort result = %v, want %v", info, tt.want)
			}
		})
	}
}
