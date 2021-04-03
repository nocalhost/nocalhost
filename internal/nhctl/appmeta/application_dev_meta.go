package appmeta

import (
	"encoding/json"
	"nocalhost/pkg/nhctl/log"
)

const (
	DEPLOYMENT ApplicationDevType = "D"
	DEV_STA    EVENT              = "DEV_STA"
	DEV_END    EVENT              = "DEV_END"
)

type ApplicationDevMeta map[ApplicationDevType]map[ /* resource name */ string] /* identifier */ string
type ApplicationDevType string
type EVENT string
type ApplicationEvent struct {
	Identifier   string
	ResourceName string
	EventType    EVENT
	DevType      ApplicationDevType
}

func (from *ApplicationDevMeta) Events(to ApplicationDevMeta) *[]*ApplicationEvent {
	var result []*ApplicationEvent

	marshalFrom, err := json.Marshal(from)
	if err != nil {
		log.Error("Error while marshal 'From ApplicationDevMeta': %s", err.Error())
		err = nil
	}
	marshalTo, err := json.Marshal(to)
	if err != nil {
		log.Error("Error while marshal 'To ApplicationDevMeta': %s", err.Error())
		err = nil
	}

	if string(marshalTo) == string(marshalFrom) {
		return &result
	}

	for devType, resourceNameIdentifierMap := range *from {
		toResourceNameIdentifierMap := to[devType]
		for resourceName, identifier := range resourceNameIdentifierMap {
			if toResourceNameIdentifierMap == nil {
				result = append(result, &ApplicationEvent{EventType: DEV_END, ResourceName: resourceName, Identifier: identifier})
			} else {
				var toIdentifier, ok = toResourceNameIdentifierMap[resourceName]

				if ok {
					// means some resource dev end then dev start
					if identifier != toIdentifier {
						result = append(result, &ApplicationEvent{EventType: DEV_END, ResourceName: resourceName, Identifier: identifier, DevType: devType})
						result = append(result, &ApplicationEvent{EventType: DEV_STA, ResourceName: resourceName, Identifier: toIdentifier, DevType: devType})
					}
				} else {
					result = append(result, &ApplicationEvent{EventType: DEV_END, ResourceName: resourceName, Identifier: identifier, DevType: devType})
				}

				delete(toResourceNameIdentifierMap, resourceName)
			}
		}
	}

	for devType, resourceNameIdentifierMap := range to {
		for resourceName, identifier := range resourceNameIdentifierMap {
			result = append(result, &ApplicationEvent{EventType: DEV_STA, ResourceName: resourceName, Identifier: identifier, DevType: devType})
		}
	}

	log.Infof("Event: %v", result)
	return &result
}
