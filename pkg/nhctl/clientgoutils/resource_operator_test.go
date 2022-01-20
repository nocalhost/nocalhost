/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"testing"
)

//func TestClientGoUtils_CreateResource(t *testing.T) {
//	client, err := NewClientGoUtils("", "")
//	Must(err)
//	err = client.Apply([]string{"/tmp/pre-install-cm.yaml"}, true)
//	if err != nil {
//		fmt.Printf("%s\n", err.Error())
//	}
//}

func TestClientGoUtils_Exec(t *testing.T) {
	client, err := NewClientGoUtils("", "")
	Must(err)
	err = client.Exec("details-555cc5597f-gx5px", "", []string{"ls"})
	Must(err)
}
