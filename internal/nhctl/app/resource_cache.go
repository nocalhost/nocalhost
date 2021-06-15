package app

import (
	"encoding/json"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_handler/item"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"reflect"
)

func (a *Application) GetConfigMap(resourceName string) (*v1.ConfigMap, error) {
	if result, err := a.getResource(resourceName, "configmaps"); err != nil {
		return nil, err
	} else {
		if marshal, err := json.Marshal(result); err != nil {
			return nil, err
		} else {
			var cm v1.ConfigMap

			if err := json.Unmarshal(marshal, &cm); err != nil {
				return nil, err
			}
			return &cm, nil
		}
	}
}

func (a *Application) getResource(resourceName, resourceType string) (interface{}, error) {
	cli, err := daemon_client.NewDaemonClient(utils.IsSudoUser())
	if err != nil {
		return nil, err
	}

	data, err := cli.SendGetResourceInfoCommand(
		a.KubeConfig, a.NameSpace, a.GetAppMeta().Application, resourceType, resourceName, nil,
	)
	if data == nil || err != nil {
		return nil, errors.Wrap(err, "Fail to get resource info from daemon")
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "Fail to get resource")
	}

	multiple := reflect.ValueOf(data).Kind() == reflect.Slice
	var items []item.Item
	var _item item.Item
	if multiple {
		log.Error("Getting resource but receive multiple result")
		_ = json.Unmarshal(bytes, &items)
	} else {
		_ = json.Unmarshal(bytes, &_item)
		items = append(items, _item)
	}

	if len(items) == 0 || items[0].Metadata == nil {
		return nil, errors.New("Fail to get resource, resource may not exist")
	}

	return items[0].Metadata, nil
}
