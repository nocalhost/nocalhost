/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
