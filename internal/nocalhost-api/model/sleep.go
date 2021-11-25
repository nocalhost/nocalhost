package model

import (
	"encoding/json"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"time"
)

type Schedule struct {
	// minutes east of UTC
	UtcOffset  *int          `json:"utc_offset" binding:"required"`
	SleepDay   *time.Weekday `json:"sleep_day" binding:"gte=0,max=6,required"`
	SleepTime  string        `json:"sleep_time" binding:"required"`
	WakeupDay  *time.Weekday `json:"wakeup_day" binding:"gte=0,max=6,required"`
	WakeupTime string        `json:"wakeup_time" binding:"required"`
}

func (s *Schedule) Hour(fmt string) int {
	v, _ := strconv.Atoi(strings.Split(fmt, ":")[0])
	return v
}

func (s *Schedule) Minute(fmt string) int {
	v, _ := strconv.Atoi(strings.Split(fmt, ":")[1])
	return v
}

func (s *Schedule) TimeZone() *time.Location {
	return time.FixedZone(strconv.Itoa(*s.UtcOffset), *s.UtcOffset * 60)
}

type SleepConfig struct {
	Schedules []Schedule `json:"schedules" binding:"required,dive"`
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
