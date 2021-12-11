package sleep

import (
	"encoding/json"
	"github.com/spf13/cast"
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
