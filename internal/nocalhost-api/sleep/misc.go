package sleep

import (
	"encoding/json"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"strconv"
	"time"
)

var zero int32 = 0
var exactly = true
var falsely = false

func timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func stringify(v interface{}) string {
	result, _ := json.Marshal(v)
	return string(result)
}

func ternary(a bool, b, c interface{}) interface{} {
	if a {
		return b
	}
	return c
}

// Calc Calculate the percentage of sleep time in a week,
// need to pay attention to the intersection of time
func percent(items *[]model.ByWeek) float32 {
	var week [10080]uint8
	for _, it := range *items {
		a := it.ToIndex(it.SleepDay, it.SleepTime)
		b := it.ToIndex(it.WakeupDay, it.WakeupTime)
		// extend into next week
		if b < a {
			for i := a; i < 10080; i++ {
				week[i] = 1
			}
			for i := 0; i < b; i++ {
				week[i] = 1
			}
		} else {
			for i := a; i < b; i++ {
				week[i] = 1
			}
		}
	}

	var used float32 = 0
	for _, v := range week {
		if v == 1 {
			used++
		}
	}
	return used / 10080
}

func ignorable(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	val, ok := annotations["dev.nocalhost/dev-mode-count"]
	if ok {
		return cast.ToInt(val) > 0
	}
	return false
}
