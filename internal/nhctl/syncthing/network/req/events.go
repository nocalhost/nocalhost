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
	Time      MyTime `json:"time"`
	Data      data   `json:"data"`
}

type data struct {
	Completion float64 `json:"completion"`
	Device     string  `json:"device"`
	Folder     string  `json:"folder"`
}

type MyTime struct {
	time.Time
}

func (selfies *MyTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)

	// Get rid of the quotes "" around the value.
	// A second option would be to include them
	// in the date format string instead, like so below:
	//   time.Parse(`"`+time.RFC3339Nano+`"`, s)
	s = s[1 : len(s)-1]

	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.999999999Z0700", s)
	}
	selfies.Time = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
	return
}

func FolderCompletionDistinct(events []event) []event {
	maps := make(map[MyTime]event)
	for _, e := range events {
		if e.Data.Completion == 100 {
			if _, found := maps[e.Time]; !found {
				maps[e.Time] = e
			}
		}
	}
	result := make([]event, len(maps))
	for _, e := range maps {
		result = append(result, e)
	}
	return result[0:]
}
