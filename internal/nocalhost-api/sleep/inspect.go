package sleep

import (
	"encoding/json"
	"github.com/spf13/cast"
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nocalhost-api/model"
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
	if len(ns.Annotations[KForceSleep]) > 0 {
		now := time.Now().UTC()
		t := time.Unix(cast.ToInt64(ns.Annotations[KForceSleep]), 0).UTC()
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
	if len(ns.Annotations[KConfig]) == 0 {
		if ns.Annotations[KStatus] == KAsleep {
			return ToBeWakeup, nil
		}
		return ToBeIgnore, nil
	}
	// 5. parse sleep config
	var conf model.SleepConfig
	err := json.Unmarshal([]byte(ns.Annotations[KConfig]), &conf)
	if err != nil {
		return ToBeIgnore, err
	}
	if len(conf.ByWeek) == 0 {
		return ToBeWakeup, nil
	}

	// 6. match sleep config
	for _, f := range conf.ByWeek {
		now := time.Now().In(f.TimeZone())
		d1 := time.Duration(*f.SleepDay - now.Weekday())
		d2 := time.Duration(*f.WakeupDay - now.Weekday())

		if *f.WakeupDay < *f.SleepDay {
			d2 = time.Duration(time.Saturday - *f.SleepDay + *f.WakeupDay + 1)
		}
		// sleep time
		t1 := now.Add(d1 * 24 * time.Hour)
		t1 = time.Date(t1.Year(), t1.Month(), t1.Day(), f.Hour(f.SleepTime), f.Minute(f.SleepTime), 0, 0, f.TimeZone())
		// wakeup time
		t2 := now.Add(d2 * 24 * time.Hour)
		t2 = time.Date(t2.Year(), t2.Month(), t2.Day(), f.Hour(f.WakeupTime), f.Minute(f.WakeupTime), 0, 0, f.TimeZone())

		println(ns.Name, " Sleep:【"+t1.String()+"】", "Wakeup:【"+t2.String()+"】")

		if now.After(t1) && now.Before(t2) {
			if ns.Annotations[KStatus] == KAsleep {
				return ToBeIgnore, nil
			}
			return ToBeAsleep, nil
		}
	}
	// 7. there are no matching rules, then dev space need to be woken up.
	if ns.Annotations[KStatus] == KAsleep {
		return ToBeWakeup, nil
	}
	return ToBeIgnore, nil
}
