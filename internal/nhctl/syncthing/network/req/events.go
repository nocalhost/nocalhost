package req

import (
	"encoding/json"
	"strconv"
)

type EventType string

const (
	EventFolderCompletion EventType = "FolderCompletion"
)

func (p *SyncthingHttpClient) Events(eventType EventType, since int32) ([]event, error) {
	uri := "rest/events"
	if len(eventType) > 0 {
		uri += "?events=" + string(eventType)
	}
	if since > 0 {
		uri += "&since=" + strconv.FormatInt(int64(since), 10)
	}
	resp, err := p.get(uri)
	if err != nil {
		return nil, err
	}
	var eventList []event
	if err = json.Unmarshal(resp, &eventList); err != nil {
		return nil, err
	}
	return eventList, err
}

type event struct {
	Id        int64  `json:"id"`
	GlobalID  int64  `json:"globalID"`
	EventType string `json:"type"`
	Time      string `json:"time"`
	Data      data   `json:"data"`
}

type data struct {
	Completion float64 `json:"completion"`
	Device     string  `json:"device"`
	Folder     string  `json:"folder"`
}
