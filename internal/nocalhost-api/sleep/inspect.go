package sleep

import (
	"encoding/json"
	"github.com/spf13/cast"
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nocalhost-api/model"
	"strconv"
	"time"
)

type ToBe int

const (
	ToBeIgnore ToBe = iota
	ToBeWakeup
	ToBeAsleep
)

func Inspect(ns *v1.Namespace) (ToBe, error) {
	// 1. check annotations
	if ns.Annotations == nil {
		return ToBeIgnore, nil
	}
	// 2. check force sleep
	if len(ns.Annotations[KForceAsleep]) > 0 {
		now := time.Now().UTC()
		t := time.Unix(cast.ToInt64(ns.Annotations[KForceAsleep]), 0).UTC()
		if t.Month() == now.Month() && t.Day() == now.Day() {
			return ToBeIgnore, nil
		}
	}
	// 3. check force wakeup
	if len(ns.Annotations[KForceWakeup]) > 0 {
		now := time.Now().UTC()
		t := time.Unix(cast.ToInt64(ns.Annotations[KForceWakeup]), 0).UTC()
		if t.Month() == now.Month() && t.Day() == now.Day() {
			return ToBeIgnore, nil
		}
	}
	// 4. check sleep config
	if len(ns.Annotations[KSleepConfig]) == 0 {
		if ns.Annotations[KSleepStatus] == KAsleep {
			return ToBeWakeup, nil
		}
		return ToBeIgnore, nil
	}
	// 5. parse sleep config
	var conf model.SleepConfig
	err := json.Unmarshal([]byte(ns.Annotations[KSleepConfig]), &conf)
	if err != nil {
		return ToBeIgnore, err
	}
	if len(conf.ByWeek) == 0 {
		return ToBeWakeup, nil
	}

	// 6. match sleep config
	for _, f := range conf.ByWeek {
		now := time.Now().In(f.TimeZone())
		cursor := toIndex(now.Weekday(), now.Hour(), now.Minute())
		index0 := toIndex(*f.SleepDay, toHour(f.SleepTime), toMinute(f.SleepTime))
		index1 := toIndex(*f.WakeupDay, toHour(f.WakeupTime), toMinute(f.WakeupTime))

		println(ns.Name, "Cursor: 【"+strconv.Itoa(cursor)+"】", " Asleep:【"+strconv.Itoa(index0)+"】", "Wakeup:【"+strconv.Itoa(index1)+"】")

		// case: sleep at Monday 20:00, wakeup at Tuesday 09:00
		if index0 < index1 && cursor > index0 && cursor < index1 {
			if ns.Annotations[KSleepStatus] == KAsleep {
				return ToBeIgnore, nil
			}
			return ToBeAsleep, nil
		}

		// case: sleep at Friday 20:00, wakeup at Monday 09:00
		if index1 < index0 && (cursor > index0 || cursor < index1) {
			if ns.Annotations[KSleepStatus] == KAsleep {
				return ToBeIgnore, nil
			}
			return ToBeAsleep, nil
		}
	}
	// 7. No rules were matched, then dev space should be wakeup.
	if ns.Annotations[KSleepStatus] == KAsleep {
		return ToBeWakeup, nil
	}
	return ToBeIgnore, nil
}
