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

var IoStreams = &genericclioptions.IOStreams{
	In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr,
}

func (c *ClientGoUtils) Patch(resourceType string, name string, patchContent string, pathType string) error {
	o := patch.NewPatchOptions(*IoStreams)
	cmd := patch.NewCmdPatch(c.NewFactory(), *IoStreams)
	if err := o.Complete(c.NewFactory(), cmd, []string{resourceType, name}); err != nil {
		return errors.WithStack(err)
	}
	o.Patch = patchContent
	o.PatchType = pathType
	return errors.WithStack(o.RunPatch())
}
