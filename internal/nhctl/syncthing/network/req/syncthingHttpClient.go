/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package req

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type SyncthingHttpClient struct {
	guiHost          string
	apiKey           string
	remoteDevice     string
	folderName       string
	reqTimeoutSecond int
}

func NewSyncthingHttpClient(
	guiHost string,
	apiKey string,
	remoteDevice string,
	folderName string,
	reqTimeoutSecond int) *SyncthingHttpClient {
	return &SyncthingHttpClient{
		guiHost:          guiHost,
		apiKey:           apiKey,
		remoteDevice:     remoteDevice,
		folderName:       folderName,
		reqTimeoutSecond: reqTimeoutSecond,
	}
}

// Get performs an HTTP GET and returns the bytes and/or an error. Any non-200
// return code is returned as an error.
func (s *SyncthingHttpClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/%s", s.guiHost, path), nil)
	if err != nil {
		return nil, err
	}
	return s.do(req, s.reqTimeoutSecond)
}

func (s *SyncthingHttpClient) Post(path, body string) ([]byte, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/%s", s.guiHost, path), bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	return s.do(req, s.reqTimeoutSecond)
}

func (s *SyncthingHttpClient) do(req *http.Request, reqTimeoutSecond int) ([]byte, error) {
	req.Header.Add("X-API-Key", s.apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(reqTimeoutSecond))
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		errStr := fmt.Sprint("Failed to override changes\nStatus code: ", resp.StatusCode)
		bs, err := ResponseToBArray(resp)
		if err != nil {
			return nil, fmt.Errorf(errStr)
		}
		body := string(bs)
		if body != "" {
			errStr += "\nBody: " + body
		}
		return nil, fmt.Errorf(errStr)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return bs, nil
}

func ResponseToBArray(response *http.Response) ([]byte, error) {
	bs, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return bs, response.Body.Close()
}
