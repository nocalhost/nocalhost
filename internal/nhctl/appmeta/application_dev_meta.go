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

package appmeta

import (
	"encoding/json"
	"nocalhost/pkg/nhctl/log"
)

type SvcType string

const (
	Deployment  SvcType = "deployment"
	StatefulSet SvcType = "statefulset"
	DaemonSet   SvcType = "daemonSet"
	Job         SvcType = "job"
	CronJob     SvcType = "cronJob"

	DEPLOYMENT SvcType = "D"
	DEV_STA    EVENT   = "DEV_STA"
	DEV_END    EVENT   = "DEV_END"
)

// Alias For compatibility with meta
func (s SvcType) Alias() SvcType {
	if s == Deployment {
		return DEPLOYMENT
	}
	return s
}

func (s SvcType) Origin() SvcType {
	if s == DEPLOYMENT {
		return Deployment
	}
	return s
}

func (s SvcType) String() string {
	return string(s)
}

type ApplicationDevMeta map[SvcType]map[ /* resource name */ string] /* identifier */ string

//type ApplicationDevType string
type EVENT string
type ApplicationEvent struct {
	Identifier   string
	ResourceName string
	EventType    EVENT
	DevType      SvcType
}

func (from *ApplicationDevMeta) copy() ApplicationDevMeta {
	m := map[SvcType]map[ /* resource name */ string]string{}
	for k, v := range *from {
		im := map[ /* resource name */ string]string{}
		for ik, iv := range v {
			im[ik] = iv
		}
		m[k] = im
	}
	return m
}

func (from *ApplicationDevMeta) Events(to ApplicationDevMeta) *[]*ApplicationEvent {
	to = to.copy()

	var result []*ApplicationEvent

	marshalFrom, err := json.Marshal(from)
	if err != nil {
		log.Error("Error while marshal 'From ApplicationDevMeta': %s", err.Error())
	}
	marshalTo, err := json.Marshal(to)
	if err != nil {
		log.Error("Error while marshal 'To ApplicationDevMeta': %s", err.Error())
	}

	if string(marshalTo) == string(marshalFrom) {
		return &result
	}

	for devType, resourceNameIdentifierMap := range *from {
		toResourceNameIdentifierMap := to[devType]
		for resourceName, identifier := range resourceNameIdentifierMap {
			if toResourceNameIdentifierMap == nil {
				result = append(
					result, &ApplicationEvent{EventType: DEV_END, ResourceName: resourceName, Identifier: identifier},
				)
			} else {
				var toIdentifier, ok = toResourceNameIdentifierMap[resourceName]

				if ok {
					// means some resource dev end then dev start
					if identifier != toIdentifier {
						result = append(
							result, &ApplicationEvent{
								EventType: DEV_END, ResourceName: resourceName, Identifier: identifier, DevType: devType,
							},
						)
						result = append(
							result, &ApplicationEvent{
								EventType: DEV_STA, ResourceName: resourceName, Identifier: toIdentifier,
								DevType: devType,
							},
						)
					}
				} else {
					result = append(
						result, &ApplicationEvent{
							EventType: DEV_END, ResourceName: resourceName, Identifier: identifier, DevType: devType,
						},
					)
				}

				delete(toResourceNameIdentifierMap, resourceName)
			}
		}
	}

	for devType, resourceNameIdentifierMap := range to {
		for resourceName, identifier := range resourceNameIdentifierMap {
			result = append(
				result, &ApplicationEvent{
					EventType: DEV_STA, ResourceName: resourceName, Identifier: identifier, DevType: devType,
				},
			)
		}
	}

	log.Infof("Event: %v", result)
	return &result
}
