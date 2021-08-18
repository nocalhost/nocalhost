/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package appmeta

import (
	"encoding/json"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/pkg/nhctl/log"
)

const (
	DEV_STA EVENT = "DEV_STA"
	DEV_END EVENT = "DEV_END"

	STARTING DevStartStatus = "STARTING"
	STARTED  DevStartStatus = "STARTED"
	NONE     DevStartStatus = "NONE"
)

type ApplicationDevMeta map[base.SvcType]map[ /* resource name */ string] /* identifier */ string

type DevStartStatus string

//type ApplicationDevType string
type EVENT string
type ApplicationEvent struct {
	Identifier   string
	ResourceName string
	EventType    EVENT
	DevType      base.SvcType
}

func (from *ApplicationDevMeta) copy() ApplicationDevMeta {
	m := map[base.SvcType]map[ /* resource name */ string]string{}
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

	for devType, fromResourceNameIdentifierMap := range *from {
		toResourceNameIdentifierMap := to[devType]
		for resourceName, identifier := range fromResourceNameIdentifierMap {
			if toResourceNameIdentifierMap == nil {
				result = append(
					result, &ApplicationEvent{EventType: DEV_END, ResourceName: resourceName, Identifier: identifier,
						DevType: devType},
				)
			} else {
				var toIdentifier, ok = toResourceNameIdentifierMap[resourceName]

				if ok {
					// means some resource dev end then dev start
					if identifier != toIdentifier {
						result = append(
							result, &ApplicationEvent{
								EventType: DEV_END, ResourceName: resourceName, Identifier: identifier,
								DevType: devType,
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

	return &result
}
