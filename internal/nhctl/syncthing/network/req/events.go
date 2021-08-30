package req

import (
	"encoding/json"
	"strconv"
	"time"
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
	eventList := make([]event, 0)
	if err = json.Unmarshal(resp, &eventList); err != nil {
		return nil, err
	}
	return eventList[0:], err
}

type event struct {
	Id        int64     `json:"id"`
	GlobalID  int64     `json:"globalID"`
	EventType string    `json:"type"`
	Time      time.Time `json:"time"`
	Data      data      `json:"data"`
}

type data struct {
	Completion float64 `json:"completion"`
	Device     string  `json:"device"`
	Folder     string  `json:"folder"`
}

func FolderCompletionDistinct(events []event) []event {
	result := make([]event, len(events))
	for _, e := range events {
		if e.Data.Completion == 100 {
			result = append(result, e)
		}
	}
	return result[0:]
}
