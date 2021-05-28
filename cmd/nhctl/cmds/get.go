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

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimejson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_handler"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"reflect"
)

var outputType string

const JSON = "json"
const YAML = "yaml"

func init() {
	getCmd.PersistentFlags().StringVarP(
		&appName, "application", "a", "", "application name",
	)
	getCmd.PersistentFlags().StringVarP(
		&outputType, "outputType", "o", "", "json or yaml",
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
			initApp(appName)
		}
		if kubeConfig == "" {
			kubeConfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
		}
		if abs, err := filepath.Abs(kubeConfig); err == nil {
			kubeConfig = abs
		}
		bytes, err := ioutil.ReadFile(kubeConfig)
		if err != nil {
			log.Fatal(err)
		}
		cli, err := daemon_client.NewDaemonClient(utils.IsSudoUser())
		if err != nil {
			log.Fatal(err)
		}
		data, err := cli.SendGetResourceInfoCommand(string(bytes), nameSpace, appName, resourceType, resourceName)
		if data == nil || err != nil {
			return
		}

		switch outputType {
		case JSON:
			out(json.Marshal, data)
		case YAML:
			out(yaml.Marshal, data)
		default:
			bytes, err = json.Marshal(data)
			if err != nil {
				return
			}
			switch resourceType {
			case "all":
				multiple := reflect.ValueOf(data).Kind() == reflect.Slice
				var results []daemon_handler.Result
				var result daemon_handler.Result
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
				var metas []*appmeta.ApplicationMeta
				var meta *appmeta.ApplicationMeta
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
				var items []daemon_handler.Item
				var item daemon_handler.Item
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

func printResult(results []daemon_handler.Result) {
	for _, r := range results {
		var needsToComplete = true
		var rows [][]string
		for _, app := range r.Application {
			for _, group := range app.Groups {
				for _, list := range group.List {
					for _, item := range list.List {
						_, name, err2 := getNamespaceAndName(item.Metadata)
						if err2 == nil {
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

func printItem(items []daemon_handler.Item) {
	var rows [][]string
	for _, i := range items {
		if namespace, name, err2 := getNamespaceAndName(i.Metadata); err2 == nil {
			rows = append(rows, []string{namespace, name})
		}
	}
	write([]string{"namespace", "name"}, rows)
}

func printMeta(metas []*appmeta.ApplicationMeta) {
	var rows [][]string
	for _, e := range metas {
		rows = append(rows, []string{e.Ns, e.Application, string(e.ApplicationType), string(e.ApplicationState)})
	}
	write([]string{"namespace", "application", "type", "state"}, rows)
}

func getNamespaceAndName(obj interface{}) (namespace, name string, errs error) {
	var caseSensitiveJsonIterator = runtimejson.CaseSensitiveJSONIterator()
	marshal, errs := caseSensitiveJsonIterator.Marshal(obj)
	if errs != nil {
		return
	}
	v := &metadataOnlyObject{}
	if errs = caseSensitiveJsonIterator.Unmarshal(marshal, v); errs != nil {
		return
	}
	return v.Namespace, v.Name, nil
}

type metadataOnlyObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}
