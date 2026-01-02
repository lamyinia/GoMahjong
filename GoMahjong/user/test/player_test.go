package test

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/proto"
	"testing"
	"user/pb"
)

func TestEasy(t *testing.T) {
	para := pb.RegisterParams{
		Account:       "123",
		Password:      "123",
		LoginPlatform: 0,
		SmsCode:       "123",
	}
	bye1, _ := json.Marshal(&para)
	bye2, _ := proto.Marshal(&para)

	fmt.Println(len(bye1))
	fmt.Println(len(bye2))
}
