/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
)

func (c *Controller) DevEnd(reset bool) error {
	if err := c.StopSyncAndPortForwardProcess(true); err != nil {
		if !reset && !os.IsNotExist(err) {
			return err // `dev end` must make sure syncthing is terminated
		}
		log.WarnE(err, "StopSyncAndPortForwardProcess failed")
	}

	if err := c.BuildPodController().RollBack(reset); err != nil {
		//if !reset {
		//	return err
		//}
		log.WarnE(err, "something incorrect occurs when rolling back")
	}

	utils.ShouldI(c.AppMeta.SvcDevEnd(c.Name, c.Identifier, c.Type, c.DevModeType), "something incorrect occurs when updating secret")

	return nil
}
