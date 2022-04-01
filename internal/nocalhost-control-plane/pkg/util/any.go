package util

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// MessageToAny marshals src into a new Any instance.
func MessageToAny(src proto.Message) *anypb.Any {
	dst := new(anypb.Any)
	if err := dst.MarshalFrom(src); err != nil {
		fmt.Println("marshals src into a new any instance error:", err)
		return nil
	}
	return dst
}
