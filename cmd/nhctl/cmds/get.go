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
	"sigs.k8s.io/yaml"
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
		if data != nil && err == nil {
			switch outputType {
			case YAML:
				if bytes, err = yaml.Marshal(data); err == nil {
					fmt.Print(string(bytes))
				}
			case JSON:
				if bytes, err = json.Marshal(data); err == nil {
					fmt.Print(string(bytes))
				}
			default:
				bytes, err = json.Marshal(data)
				if err != nil {
					return
				}
				switch resourceType {
				case "all":
					var results []daemon_handler.Result
					if err = json.Unmarshal(bytes, &results); err != nil {
						var result daemon_handler.Result
						if err = json.Unmarshal(bytes, &result); err == nil {
							results = append(results, result)
						}
					}
					for _, result := range results {
						var needsToComplete = true
						var rows [][]string
						for _, app := range result.Application {
							for _, group := range app.Groups {
								for _, list := range group.List {
									for _, item := range list.List {
										_, name, err2 := getNsAndName(item.Metadata)
										if err2 == nil {
											needsToComplete = false
											row := []string{result.Namespace, app.Name, group.GroupName,
												list.Name + "/" + name}
											rows = append(rows, row)
										} else {
											log.Error(err2)
										}
									}
								}
							}
						}
						if needsToComplete {
							rows = append(rows, []string{result.Namespace})
						}
						write([]string{"Namespace", "Application", "Group", "Name"}, rows)
					}
				case "app", "application":
					var metas []*appmeta.ApplicationMeta
					if resourceName == "" {
						_ = json.Unmarshal(bytes, &metas)
					} else {
						var meta *appmeta.ApplicationMeta
						if err := json.Unmarshal(bytes, &meta); err == nil {
							metas = append(metas, meta)
						}
					}
					var rows [][]string
					for _, meta := range metas {
						rows = append(rows, []string{
							meta.Ns, meta.Application, string(meta.ApplicationType), string(meta.ApplicationState)})
					}
					write([]string{"Namespace", "Application", "Type", "State"}, rows)
				default:
					var items []daemon_handler.Item
					if resourceName == "" {
						_ = json.Unmarshal(bytes, &items)
					} else {
						var item daemon_handler.Item
						if err = json.Unmarshal(bytes, &item); err == nil {
							items = append(items, item)
						}
					}
					var rows [][]string
					for _, item := range items {
						namespace, name, err2 := getNsAndName(item.Metadata)
						if err2 == nil {
							rows = append(rows, []string{namespace, name})
						} else {
							log.Error(err2)
						}
					}
					write([]string{"namespace", "name"}, rows)
				}
			}
		}
	},
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

func getNsAndName(obj interface{}) (namespace, name string, errs error) {
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
