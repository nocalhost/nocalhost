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

package clientgoutils

import (
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/patch"
	"os"
)

func (c *ClientGoUtils) Patch(resourceType string, name string, jsonStr string) error {
	ioStreams := genericclioptions.IOStreams{
		In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr,
	} // don't print log to stderr
	o := patch.NewPatchOptions(ioStreams)
	cmd := patch.NewCmdPatch(c.newFactory(), ioStreams)
	//cmd.Flags()
	if err := o.Complete(c.newFactory(), cmd, []string{resourceType, name}); err != nil {
		return errors.Wrap(err, "")
	}
	o.Patch = jsonStr
	return errors.Wrap(o.RunPatch(), "")
}
