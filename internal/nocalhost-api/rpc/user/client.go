/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	pb "nocalhost/internal/nocalhost-api/rpc/user/v0"
)

func main() {
	serviceAddress := "127.0.0.1:1234"
	conn, err := grpc.Dial(serviceAddress, grpc.WithInsecure())
	if err != nil {
		panic("connect error")
	}
	defer conn.Close()

	userClient := pb.NewUserServiceClient(conn)
	userReq := &pb.PhoneLoginRequest{
		Phone:      130,
		VerifyCode: 123,
	}
	reply, _ := userClient.LoginByPhone(context.Background(), userReq)
	fmt.Printf("UserService LoginByPhone : %+v", reply)
}
