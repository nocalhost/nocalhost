package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/qiniu/api.v7"
	"github.com/qiniu/api.v7/conf"
)

//  七牛鉴权类，用于生成Qbox, Qiniu, Upload签名
// AK/SK可以从 https://portal.qiniu.com/user/key 获取。
type Credentials struct {
	AccessKey string
	SecretKey []byte
}

// 构建一个Credentials对象
func New(accessKey, secretKey string) *Credentials {
	return &Credentials{accessKey, []byte(secretKey)}
}

// Sign 对数据进行签名，一般用于私有空间下载用途
func (ath *Credentials) Sign(data []byte) (token string) {
	h := hmac.New(sha1.New, ath.SecretKey)
	h.Write(data)

	sign := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s:%s", ath.AccessKey, sign)
}

// SignToken 根据t的类型对请求进行签名，并把token加入req中
func (ath *Credentials) AddToken(t TokenType, req *http.Request) error {
	switch t {
	case TokenQiniu:
		token, sErr := ath.SignRequestV2(req)
		if sErr != nil {
			return sErr
		}
		req.Header.Add("Authorization", "Qiniu "+token)
	default:
		token, err := ath.SignRequest(req)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "QBox "+token)
	}
	return nil
}

// SignWithData 对数据进行签名，一般用于上传凭证的生成用途
func (ath *Credentials) SignWithData(b []byte) (token string) {
	encodedData := base64.URLEncoding.EncodeToString(b)
	sign := ath.Sign([]byte(encodedData))
	return fmt.Sprintf("%s:%s", sign, encodedData)
}

func collectData(req *http.Request) (data []byte, err error) {
	u := req.URL
	s := u.Path
	if u.RawQuery != "" {
		s += "?"
		s += u.RawQuery
	}
	s += "\n"

	data = []byte(s)
	if incBody(req) {
		s2, rErr := api.BytesFromRequest(req)
		if rErr != nil {
			err = rErr
			return
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(s2))
		data = append(data, s2...)
	}
	return
}

func collectDataV2(req *http.Request) (data []byte, err error) {
	u := req.URL

	//write method path?query
	s := fmt.Sprintf("%s %s", req.Method, u.Path)
	if u.RawQuery != "" {
		s += "?"
		s += u.RawQuery
	}

	//write host and post
	s += "\nHost: "
	s += req.Host

	//write content type
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		s += "\n"
		s += fmt.Sprintf("Content-Type: %s", contentType)
	}
	s += "\n\n"

	data = []byte(s)
	//write body
	if incBodyV2(req) {
		s2, rErr := api.BytesFromRequest(req)
		if rErr != nil {
			err = rErr
			return
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(s2))
		data = append(data, s2...)
	}
	return
}

// SignRequest 对数据进行签名，一般用于管理凭证的生成
func (ath *Credentials) SignRequest(req *http.Request) (token string, err error) {
	data, err := collectData(req)
	if err != nil {
		return
	}
	token = ath.Sign(data)
	return
}

// SignRequestV2 对数据进行签名，一般用于高级管理凭证的生成
func (ath *Credentials) SignRequestV2(req *http.Request) (token string, err error) {

	data, err := collectDataV2(req)
	if err != nil {
		return
	}
	token = ath.Sign(data)
	return
}

// 管理凭证生成时，是否同时对request body进行签名
func incBody(req *http.Request) bool {
	return req.Body != nil && req.Header.Get("Content-Type") == conf.CONTENT_TYPE_FORM
}

func incBodyV2(req *http.Request) bool {
	contentType := req.Header.Get("Content-Type")
	return req.Body != nil && (contentType == conf.CONTENT_TYPE_FORM || contentType == conf.CONTENT_TYPE_JSON)
}

// VerifyCallback 验证上传回调请求是否来自七牛
func (ath *Credentials) VerifyCallback(req *http.Request) (bool, error) {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return false, nil
	}

	token, err := ath.SignRequest(req)
	if err != nil {
		return false, err
	}

	return auth == "QBox "+token, nil
}
