package req

import (
	"encoding/json"
	"strconv"
)

func (p *SyncthingHttpClient) Events(since int64) ([]event, error) {
	resp, err := p.get("rest/events?since=" + strconv.FormatInt(since, 10))
	if err != nil {
		return nil, err
	}
	var eventList []event
	if err = json.Unmarshal(resp, &eventList); err != nil {
		return nil, err
	}
	return eventList, err
}

type EventType string

const (
	EventFolderCompletion EventType = "FolderCompletion"
)

type event struct {
	Id        int64     `json:"id"`
	GlobalID  int64     `json:"globalID"`
	EventType EventType `json:"type"`
	Time      string    `json:"time"`
	Data      data      `json:"data"`
}

type data struct {
	Completion float64 `json:"completion"`
	Device     string  `json:"device"`
	Folder     string  `json:"folder"`
}
