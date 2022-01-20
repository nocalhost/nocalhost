/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package syncthing

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"

	"nocalhost/pkg/nhctl/log"
)

type addAPIKeyTransport struct {
	T http.RoundTripper
}

func (akt *addAPIKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-API-Key", "cnd")
	return akt.T.RoundTrip(req)
}

//NewAPIClient returns a new syncthing api client configured to call the syncthing api
func NewAPIClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &addAPIKeyTransport{http.DefaultTransport},
	}
}

// APICall calls the syncthing API and returns the parsed json or an error
func (s *Syncthing) APICall(
	ctx context.Context, url, method string, code int, params map[string]string, local bool,
	body []byte, readBody bool,
	maxRetries int,
) ([]byte, error) {
	retries := 0
	for {
		result, err := s.callWithRetry(ctx, url, method, code, params, local, body, readBody)
		if err == nil {
			return result, nil
		}
		if retries == maxRetries {
			return nil, err
		}
		log.Debugf("retrying syncthing call[%s] local=%t: %s", url, local, err.Error())
		time.Sleep(200 * time.Millisecond)
		retries++
	}
}

func (s *Syncthing) callWithRetry(
	ctx context.Context, url, method string, code int, params map[string]string, local bool, body []byte, readBody bool,
) ([]byte, error) {
	var urlPath string
	if local {
		urlPath = filepath.Join(s.GUIAddress, url)
		s.Client.Timeout = 3 * time.Second
	} else {
		urlPath = filepath.Join(s.RemoteGUIAddress, url)
		s.Client.Timeout = 25 * time.Second
		if url == "rest/db/ignores" || url == "rest/system/ping" {
			s.Client.Timeout = 5 * time.Second
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("http://%s", urlPath), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize syncthing API request: %w", err)
	}

	req = req.WithContext(ctx)

	q := req.URL.Query()
	q.Add("limit", "30")

	for key, value := range params {
		q.Add(key, value)
	}

	req.URL.RawQuery = q.Encode()

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call syncthing [%s]: %w", url, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != code {
		return nil, fmt.Errorf(
			"unexpected response from syncthing [%s | %d]: %s", req.URL.String(), resp.StatusCode, string(body),
		)
	}

	if !readBody {
		return nil, nil
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from syncthing [%s]: %w", url, err)
	}

	return body, nil
}
