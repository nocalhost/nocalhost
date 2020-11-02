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

package sign

import (
	"fmt"
	"net/url"
	"testing"
)

func TestSignMd5(t *testing.T) {
	signer := NewSignerMd5()
	signer.SetAppID("123456")
	signer.SetTimeStamp(1594458195)
	signer.SetNonceStr("supertempstr")
	signer.AddBody("city", "beijing")
	signer.SetAppSecretWrapBody("20200711")

	fmt.Println("生成签名前字符串：" + signer.GetSignBodyString())
	fmt.Println("生成sign：" + signer.GetSignature())
	fmt.Println("输出URL字符串：" + signer.GetSignedQuery())
	if "app_id=123456&city=beijing&nonce_str=supertempstr&timestamp=1594458195&sign=af603c2375aa2265e970737600555d7f" != signer.GetSignedQuery() {
		t.Fatal("Md5校验失败")
	}
}

func TestSigner_AddBody(t *testing.T) {
	body := make(url.Values)
	body["username"] = []string{"1024casts"}
	body["tags"] = []string{"github", "gopher"}

	signer := NewSignerHmac()
	signer.SetAppSecret("20200711")
	signer.SetTimeStamp(1594458195)
	signer.SetAppID("112233")
	signer.SetNonceStr("supertempstr")
	for k, v := range body {
		signer.AddBodies(k, v)
	}

	body.Add(KeyNameTimeStamp, "1594458195")
	body.Add(KeyNameAppID, "snake")
	body.Add(KeyNameNonceStr, "snake_nonce")

	fmt.Println("生成签字字符串：" + signer.GetSignBodyString())
	fmt.Println("输出URL字符串：" + signer.GetSignedQuery())

	verifier := NewVerifier()
	verifier.ParseValues(body)

	resigner := NewSignerHmac()
	resigner.SetAppSecret("snake_key")
	resigner.SetBody(verifier.GetBodyWithoutSign())

	fmt.Println("重新生成签字字符串：" + resigner.GetSignBodyString())
	fmt.Println("重新输出URL字符串：" + resigner.GetSignedQuery())
}
