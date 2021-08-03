/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

//
//func TestClientGoUtils_Apply(t *testing.T) {
//	client := getClient()
//	err := client.Apply("/Users/xinxinhuang/.nh/nhctl/application/bookinfo-coding/resources/manifest/templates/ratings.yaml",)
//	if err != nil {
//		panic(err)
//	}
//}
//
//func TestClientGoUtils_CreateResourceInfo(t *testing.T) {
//	client := getClient()
//
//	infos, err := client.GetResourceInfoFromFiles([]string{"/Users/xinxinhuang/.nh/nhctl/application/bookinfo-coding/resources/manifest/templates/ratings.yaml"}, true)
//	if err != nil {
//		panic(err)
//	}
//
//	for _, info := range infos {
//		fmt.Println(info.Object.GetObjectKind().GroupVersionKind().Kind)
//		if info.Object.GetObjectKind().GroupVersionKind().Kind == "Deployment" {
//			//err = client.UpdateResourceInfoByClientSide(info)
//			err = client.ApplyResourceInfo(info)
//			if err != nil {
//				panic(err)
//			}
//		}
//	}
//}
//
//func TestClientGoUtils_UpdateResourceInfo(t *testing.T) {
//	client, err := NewClientGoUtils("", "")
//	if err != nil {
//		panic(err)
//	}
//
//	infos, err := client.GetResourceInfoFromFiles([]string{"/tmp/yaml/ubuntu.yaml"}, true)
//	if err != nil {
//		panic(err)
//	}
//
//	for _, info := range infos {
//		err = client.UpdateResourceInfoByServerSide(info)
//		if err != nil {
//			fmt.Println(err.Error())
//		}
//	}
//}
