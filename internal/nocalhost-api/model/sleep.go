package model

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

type ByWeek struct {
	// minutes east of UTC
	UtcOffset *int          `json:"utc_offset" binding:"required"`
	SleepDay  *time.Weekday `json:"sleep_day" binding:"gte=0,max=6,required" swaggertype:"integer" enums:"0,1,2,3,4,5,6"`
	WakeupDay *time.Weekday `json:"wakeup_day" binding:"gte=0,max=6,required" swaggertype:"integer" enums:"0,1,2,3,4,5,6"`

	SleepTime  string `json:"sleep_time" binding:"required,timing"`  // eg. 20:00, 23:55
	WakeupTime string `json:"wakeup_time" binding:"required,timing"` // eg. 08:00, 09:30
}

func (s *ByWeek) TimeZone() *time.Location {
	return time.FixedZone(strconv.Itoa(*s.UtcOffset), *s.UtcOffset*60)
}

type SleepConfig struct {
	ByWeek []ByWeek `json:"by_week" binding:"required,dive"`
}

func (h *SleepConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	// check value for byte slice
	b, ok := value.([]byte)
	if !ok {
		return errors.Errorf("value is not []byte, value: %v", value)
	}
	// check value for empty string
	if len(string(value.([]byte))) == 0 {
		return nil
	}
	return json.Unmarshal(b, h)
}

func (h SleepConfig) Value() (driver.Value, error) {
	return json.Marshal(h)
}
