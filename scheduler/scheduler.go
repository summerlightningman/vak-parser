package scheduler

import (
	"fmt"
	"time"
)

// Смотрим приказы в 9:30 и 13:45

func Scheduler(schedCh chan<- struct{}) {
	tz, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return
	}

	for {
		now := time.Now().In(tz)

		dt1 := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, tz)
		dt2 := time.Date(now.Year(), now.Month(), now.Day(), 13, 45, 0, 0, tz)
		dt3 := time.Date(now.Year(), now.Month(), now.Day() + 1, 9, 30, 0, 0, tz)

		sub1 := dt1.Sub(now)
		sub2 := dt2.Sub(now)
		sub3 := dt3.Sub(now)

		var sub time.Duration
		if sub1 == 0 || sub2 == 0 {
			schedCh<-struct{}{}
		} else if (sub1 < 0 && sub2 < 0) {
			sub = sub3
		} else if (sub1 > 0 && sub2 < 0) || (sub1 > 0 && sub2 > 0 && sub1 < sub2) {
			sub = sub1
		} else {
			sub = sub2
		}
		fmt.Printf("До ближайшего запроса %fs\n",  sub.Seconds() / 3600)
		time.AfterFunc(sub, func() {
			schedCh<-struct{}{}
		})
		time.Sleep(sub)
	}
}
