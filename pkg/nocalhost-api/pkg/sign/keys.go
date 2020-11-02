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

const (
	KeyNameTimeStamp = "timestamp"
	KeyNameNonceStr  = "nonce_str"
	KeyNameAppID     = "app_id"
	KeyNameSign      = "sign"
)

// DefaultKeyName 签名需要用到的字段
type DefaultKeyName struct {
	Timestamp string
	NonceStr  string
	AppID     string
	Sign      string
}

func newDefaultKeyName() *DefaultKeyName {
	return &DefaultKeyName{
		Timestamp: KeyNameTimeStamp,
		NonceStr:  KeyNameNonceStr,
		AppID:     KeyNameAppID,
		Sign:      KeyNameSign,
	}
}

// SetKeyNameTimestamp 设定时间戳
func (d *DefaultKeyName) SetKeyNameTimestamp(name string) {
	d.Timestamp = name
}

// SetKeyNameNonceStr 设定随机字符串
func (d *DefaultKeyName) SetKeyNameNonceStr(name string) {
	d.NonceStr = name
}

// SetKeyNameAppID 设定app id
func (d *DefaultKeyName) SetKeyNameAppID(name string) {
	d.AppID = name
}

// SetKeyNameSign 设定签名
func (d *DefaultKeyName) SetKeyNameSign(name string) {
	d.Sign = name
}
