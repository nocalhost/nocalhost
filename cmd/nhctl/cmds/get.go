/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_handler/item"
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/utils"
	k8sutil "nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"reflect"
)

var outputType string
var label map[string]string

const JSON = "json"
const YAML = "yaml"

func init() {
	getCmd.PersistentFlags().StringVarP(
		&appName, "application", "a", "", "application name",
	)
	getCmd.PersistentFlags().StringVarP(
		&outputType, "outputType", "o", "", "json or yaml",
	)
	getCmd.PersistentFlags().StringToStringVarP(
		&label, "selector", "l", map[string]string{}, "Selector (label query) to filter on, "+
			"only supports '='.(e.g. -l key1=value1,key2=value2)",
	)
	rootCmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get type",
	Short: "Get resource info",
	Long: `
Get resource info
nhctl get service serviceName [-n namespace] --kubeconfig=kubeconfigfile
`,
	Example: `
	# Get all application
	nhctl get application --kubeconfig=kubeconfigfile

	# Get all application in namespace
	nhctl get application -n namespaceName --kubeconfig=kubeoconfigpath
  
	# Get all deployment of application in namespace
	nhctl get deployment -n namespaceName -a bookinfo --kubeconfig=kubeconfigpath
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		resourceName := ""
		if len(args) >= 2 {
			resourceName = args[1]
		}
		if appName != "" {
			go func() {
				if _, err := common.InitAppMutate(appName); err != nil {
					log.Logf("error while init app: %s on namespace: %s, error: %v", appName, common.NameSpace, err)
				}
			}()
		}
		if common.KubeConfig == "" {
			common.KubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
		}
		if abs, err := filepath.Abs(common.KubeConfig); err == nil {
			common.KubeConfig = abs
		}
		if _, err := ioutil.ReadFile(common.KubeConfig); err != nil {
			log.FatalE(err, "")
		}
		cli, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
		if err != nil {
			log.FatalE(err, "")
		}
		data, err := cli.SendGetResourceInfoCommand(
			common.KubeConfig, common.NameSpace, appName, resourceType, resourceName, label, false,
		)
		if err != nil {
			log.Error(err)
			return
		}
		if data == nil {
			return
		}

		switch outputType {
		case JSON:
			out(json.Marshal, data)
		case YAML:
			out(yaml.Marshal, data)
		default:
			bytes, err := json.Marshal(data)
			if err != nil {
				log.Error(err)
				return
			}
			switch resourceType {
			case "all":
				multiple := reflect.ValueOf(data).Kind() == reflect.Slice
				var results []item.Result
				var result item.Result
				if multiple {
					_ = json.Unmarshal(bytes, &results)
				} else {
					_ = json.Unmarshal(bytes, &result)
				}
				if !multiple {
					results = append(results, result)
				}
				printResult(results)
			case "app", "application":
				multiple := reflect.ValueOf(data).Kind() == reflect.Slice
				var metas []*model.Namespace
				var meta *model.Namespace
				if multiple {
					_ = json.Unmarshal(bytes, &metas)
				} else {
					_ = json.Unmarshal(bytes, &meta)
				}
				if !multiple {
					metas = append(metas, meta)
				}
				printMeta(metas)
			default:
				multiple := reflect.ValueOf(data).Kind() == reflect.Slice
				var items []item.Item
				var item item.Item
				if multiple {
					_ = json.Unmarshal(bytes, &items)
				} else {
					_ = json.Unmarshal(bytes, &item)
				}
				if !multiple {
					items = append(items, item)
				}
				printItem(items)
			}
		}
	},
}

func out(f func(interface{}) ([]byte, error), data interface{}) {
	if bytes, err := f(data); err == nil {
		fmt.Print(string(bytes))
	}
}

func write(headers []string, rows [][]string) {
	writer := tablewriter.NewWriter(os.Stdout)
	writer.SetBorder(false)
	writer.SetColumnSeparator("")
	writer.SetRowSeparator("")
	writer.SetCenterSeparator("")
	writer.SetHeaderLine(false)
	writer.SetHeader(headers)
	writer.AppendBulk(rows)
	writer.Render()
}

func printResult(results []item.Result) {
	for _, r := range results {
		var needsToComplete = true
		var rows [][]string
		for _, app := range r.Application {
			for _, group := range app.Groups {
				for _, list := range group.List {
					for _, omItem := range list.List {
						_, name, err := k8sutil.GetNamespaceAndNameFromObjectMeta(omItem.Metadata)
						if err == nil {
							needsToComplete = false
							row := []string{r.Namespace, app.Name, group.GroupName, list.Name + "/" + name}
							rows = append(rows, row)
						}
					}
				}
			}
		}
		if needsToComplete {
			rows = append(rows, []string{r.Namespace})
		}
		write([]string{"namespace", "application", "group", "name"}, rows)
	}
}

func printItem(items []item.Item) {
	var rows [][]string
	for _, i := range items {
		if namespace, name, err2 := k8sutil.GetNamespaceAndNameFromObjectMeta(i.Metadata); err2 == nil {
			rows = append(rows, []string{namespace, name})
		}
	}
	write([]string{"namespace", "name"}, rows)
}

func printMeta(metas []*model.Namespace) {
	var rows [][]string
	for _, e := range metas {
		for _, appInfo := range e.Application {
			rows = append(rows, []string{e.Namespace, appInfo.Name, appInfo.Type})
		}
	}
	write([]string{"namespace", "name", "type"}, rows)
}
