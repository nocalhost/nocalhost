/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package sign

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"nocalhost/pkg/nocalhost-api/pkg/utils"
)

// CryptoFunc
type CryptoFunc func(secretKey string, args string) []byte

// Signer define
type Signer struct {
	*DefaultKeyName

	body       url.Values
	bodyPrefix string
	bodySuffix string
	splitChar  string

	secretKey  string
	cryptoFunc CryptoFunc
}

// NewSigner  Signer
func NewSigner(cryptoFunc CryptoFunc) *Signer {
	return &Signer{
		DefaultKeyName: newDefaultKeyName(),
		body:           make(url.Values),
		bodyPrefix:     "",
		bodySuffix:     "",
		splitChar:      "",
		cryptoFunc:     cryptoFunc,
	}
}

// NewSignerMd5
func NewSignerMd5() *Signer {
	return NewSigner(Md5Sign)
}

// NewSignerHmac
func NewSignerHmac() *Signer {
	return NewSigner(HmacSign)
}

// SetBody
func (s *Signer) SetBody(body url.Values) {
	for k, v := range body {
		s.body[k] = v
	}
}

// GetBody
func (s *Signer) GetBody() url.Values {
	return s.body
}

// AddBody
func (s *Signer) AddBody(key string, value string) *Signer {
	return s.AddBodies(key, []string{value})
}

// AddBodies add value to body
func (s *Signer) AddBodies(key string, value []string) *Signer {
	s.body[key] = value
	return s
}

// SetTimeStamp
func (s *Signer) SetTimeStamp(ts int64) *Signer {
	return s.AddBody(s.Timestamp, strconv.FormatInt(ts, 10))
}

// GetTimeStamp
func (s *Signer) GetTimeStamp() string {
	return s.body.Get(s.Timestamp)
}

// SetNonceStr
func (s *Signer) SetNonceStr(nonce string) *Signer {
	return s.AddBody(s.NonceStr, nonce)
}

// GetNonceStr
func (s *Signer) GetNonceStr() string {
	return s.body.Get(s.NonceStr)
}

// SetAppID
func (s *Signer) SetAppID(appID string) *Signer {
	return s.AddBody(s.AppID, appID)
}

// GetAppID get app id
func (s *Signer) GetAppID() string {
	return s.body.Get(s.AppID)
}

// RandNonceStr
func (s *Signer) RandNonceStr() *Signer {
	return s.SetNonceStr(utils.RandomStr(16))
}

// SetSignBodyPrefix
func (s *Signer) SetSignBodyPrefix(prefix string) *Signer {
	s.bodyPrefix = prefix
	return s
}

// SetSignBodySuffix
func (s *Signer) SetSignBodySuffix(suffix string) *Signer {
	s.bodySuffix = suffix
	return s
}

// SetSplitChar
func (s *Signer) SetSplitChar(split string) *Signer {
	s.splitChar = split
	return s
}

// SetAppSecret
func (s *Signer) SetAppSecret(appSecret string) *Signer {
	s.secretKey = appSecret
	return s
}

// SetAppSecretWrapBody
func (s *Signer) SetAppSecretWrapBody(appSecret string) *Signer {
	s.SetSignBodyPrefix(appSecret)
	s.SetSignBodySuffix(appSecret)
	return s.SetAppSecret(appSecret)
}

// GetSignBodyString
func (s *Signer) GetSignBodyString() string {
	return s.MakeRawBodyString()
}

// MakeRawBodyString
func (s *Signer) MakeRawBodyString() string {
	return s.bodyPrefix + s.splitChar + s.getSortedBodyString() + s.splitChar + s.bodySuffix
}

// GetSignedQuery
func (s *Signer) GetSignedQuery() string {
	return s.MakeSignedQuery()
}

// GetSignedQuery
func (s *Signer) MakeSignedQuery() string {
	body := s.getSortedBodyString()
	sign := s.GetSignature()
	return body + "&" + s.Sign + "=" + sign
}

// GetSignature
func (s *Signer) GetSignature() string {
	return s.MakeSign()
}

// MakeSign
func (s *Signer) MakeSign() string {
	sign := fmt.Sprintf("%x", s.cryptoFunc(s.secretKey, s.GetSignBodyString()))
	return sign
}

func (s *Signer) getSortedBodyString() string {
	return SortKVPairs(s.body)
}

// SortKVPairs
func SortKVPairs(m url.Values) string {
	size := len(m)
	if size == 0 {
		return ""
	}
	keys := make([]string, size)
	idx := 0
	for k := range m {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)
	pairs := make([]string, size)
	for i, key := range keys {
		pairs[i] = key + "=" + strings.Join(m[key], ",")
	}
	return strings.Join(pairs, "&")
}
