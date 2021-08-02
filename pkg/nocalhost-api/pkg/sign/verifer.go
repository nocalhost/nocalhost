/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package sign

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"nocalhost/pkg/nocalhost-api/pkg/utils"
)

// Verifier define struct
type Verifier struct {
	*DefaultKeyName
	body url.Values

	timeout time.Duration
}

// NewVerifier  Verifier
func NewVerifier() *Verifier {
	return &Verifier{
		DefaultKeyName: newDefaultKeyName(),
		body:           make(url.Values),
		timeout:        time.Minute * 5,
	}
}

// ParseQuery
func (v *Verifier) ParseQuery(requestUri string) error {
	requestQuery := ""
	idx := strings.Index(requestUri, "?")
	if idx > 0 {
		requestQuery = requestUri[idx+1:]
	}
	query, err := url.ParseQuery(requestQuery)
	if nil != err {
		return err
	}
	v.ParseValues(query)
	return nil
}

// ParseValues
func (v *Verifier) ParseValues(values url.Values) {
	for key, value := range values {
		v.body[key] = value
	}
}

// SetTimeout
func (v *Verifier) SetTimeout(timeout time.Duration) *Verifier {
	v.timeout = timeout
	return v
}

// MustString
func (v *Verifier) MustString(key string) string {
	ss := v.MustStrings(key)
	if len(ss) == 0 {
		return ""
	}
	return ss[0]
}

// MustString
func (v *Verifier) MustStrings(key string) []string {
	return v.body[key]
}

// MustInt64
func (v *Verifier) MustInt64(key string) int64 {
	n, _ := utils.StringToInt64(v.MustString(key))
	return n
}

// MustHasKeys
func (v *Verifier) MustHasKeys(keys ...string) error {
	for _, key := range keys {
		if _, hit := v.body[key]; !hit {
			return fmt.Errorf("KEY_MISSED:<%s>", key)
		}
	}
	return nil
}

// MustHasKeys
func (v *Verifier) MustHasOtherKeys(keys ...string) error {
	fields := []string{v.Timestamp, v.NonceStr, v.Sign, v.AppID}
	if len(keys) > 0 {
		fields = append(fields, keys...)
	}
	return v.MustHasKeys(fields...)
}

// CheckTimeStamp
func (v *Verifier) CheckTimeStamp() error {
	timestamp := v.GetTimestamp()
	thatTime := time.Unix(timestamp, 0)
	if time.Since(thatTime) > v.timeout {
		return fmt.Errorf("TIMESTAMP_TIMEOUT:<%d>", timestamp)
	}
	return nil
}

// GetAppID
func (v *Verifier) GetAppID() string {
	return v.MustString(v.AppID)
}

// GetNonceStr
func (v *Verifier) GetNonceStr() string {
	return v.MustString(v.NonceStr)
}

// GetSign
func (v *Verifier) GetSign() string {
	return v.MustString(v.Sign)
}

// GetTimestamp
func (v *Verifier) GetTimestamp() int64 {
	return v.MustInt64(v.Timestamp)
}

// GetBodyWithoutSign
func (v *Verifier) GetBodyWithoutSign() url.Values {
	out := make(url.Values)
	for k, val := range v.body {
		if k != v.Sign {
			out[k] = val
		}
	}
	return out
}

// GetBody
func (v *Verifier) GetBody() url.Values {
	out := make(url.Values)
	for k, val := range v.body {
		out[k] = val
	}
	return out
}
