/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sign

const (
	KeyNameTimeStamp = "timestamp"
	KeyNameNonceStr  = "nonce_str"
	KeyNameAppID     = "app_id"
	KeyNameSign      = "sign"
)

// DefaultKeyName
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

// SetKeyNameTimestamp
func (d *DefaultKeyName) SetKeyNameTimestamp(name string) {
	d.Timestamp = name
}

// SetKeyNameNonceStr
func (d *DefaultKeyName) SetKeyNameNonceStr(name string) {
	d.NonceStr = name
}

// SetKeyNameAppID
func (d *DefaultKeyName) SetKeyNameAppID(name string) {
	d.AppID = name
}

// SetKeyNameSign
func (d *DefaultKeyName) SetKeyNameSign(name string) {
	d.Sign = name
}
