/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
	cmd := patch.NewCmdPatch(c.NewFactory(), ioStreams)
	if err := o.Complete(c.NewFactory(), cmd, []string{resourceType, name}); err != nil {
		return errors.Wrap(err, "")
	}
	o.Patch = jsonStr
	return errors.Wrap(o.RunPatch(), "")
}
