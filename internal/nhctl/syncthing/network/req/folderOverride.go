/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package req

func (p *SyncthingHttpClient) FolderOverride() error {
	_, err := p.Post("rest/db/override?folder="+p.folderName, "")
	if err != nil {
		return err
	}
	return nil
}
