/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
