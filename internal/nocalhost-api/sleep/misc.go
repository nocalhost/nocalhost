package sleep

import (
	"encoding/json"
	"github.com/spf13/cast"
	"strconv"
	"strings"
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

func toHour(hm string) int {
	v, _ := strconv.Atoi(strings.Split(hm, ":")[0])
	return v
}

func toMinute(hm string) int {
	v, _ := strconv.Atoi(strings.Split(hm, ":")[1])
	return v
}

func toIndex(day time.Weekday, h int, m int) int {
	return int(day)*24*60 + h*60 + m
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
