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

package controller

import (
	"encoding/json"
	"fmt"
	"nocalhost/internal/nhctl/profile"
	"testing"
)

func TestIsResourcesLimitToLow(t *testing.T) {
	r := &profile.ResourceQuota{
		Limits:   &profile.QuotaList{Memory: "1.5Gi", Cpu: "1"},
		Requests: &profile.QuotaList{Memory: "50Mi", Cpu: "100m"},
	}
	rq, _ := convertResourceQuota(r)
	fmt.Println(rq.Limits.Cpu().String())
	//bys, _ := json.Marshal(rq)
	//fmt.Printf("%v\n", string(bys))
	bys, _ := json.Marshal(rq.Limits)
	fmt.Println(string(bys))
	fmt.Println(IsResourcesLimitTooLow(rq))
	fmt.Println(IsResourcesLimitTooLow(nil))
	r.Limits = nil
	rq, _ = convertResourceQuota(r)
	fmt.Println(IsResourcesLimitTooLow(rq))
}
