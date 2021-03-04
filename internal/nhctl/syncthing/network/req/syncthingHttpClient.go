package req

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

type SyncthingHttpClient struct {
	guiHost      string
	apiKey       string
	remoteDevice string
	folderName   string
}

func NewSyncthingHttpClient(
	guiHost string,
	apiKey string,
	remoteDevice string,
	folderName string) *SyncthingHttpClient {
	return &SyncthingHttpClient{
		guiHost:      guiHost,
		apiKey:       apiKey,
		remoteDevice: remoteDevice,
		folderName:   folderName,
	}
}

// Get performs an HTTP GET and returns the bytes and/or an error. Any non-200
// return code is returned as an error.
func (s *SyncthingHttpClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/%s", s.guiHost, path), nil)
	if err != nil {
		return nil, err
	}
	return s.do(req)
}

func (s *SyncthingHttpClient) Post(path, body string) ([]byte, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/%s", s.guiHost, path), bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	return s.do(req)
}

func (s *SyncthingHttpClient) do(req *http.Request) ([]byte, error) {
	req.Header.Add("X-API-Key", s.apiKey)

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
