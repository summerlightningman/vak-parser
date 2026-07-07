package scheduler

import (
	"fmt"
	"log"
	"os"
	"time"
)

var russianMonths = [...]string{
	"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}

func formatRunTime(t time.Time) string {
	return fmt.Sprintf("%d %s в %02d:%02d", t.Day(), russianMonths[t.Month()-1], t.Hour(), t.Minute())
}

// Смотрим приказы в 9:30 и 13:45

func Scheduler(schedCh chan<- struct{}) {
	tz, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return
	}

	log := log.New(os.Stderr, "[scheduler]", log.LstdFlags)

	for {
		now := time.Now().In(tz)

		dt1 := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, tz)
		dt2 := time.Date(now.Year(), now.Month(), now.Day(), 13, 45, 0, 0, tz)
		dt3 := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 30, 0, 0, tz)

		sub1 := dt1.Sub(now)
		sub2 := dt2.Sub(now)
		sub3 := dt3.Sub(now)

		var sub time.Duration
		var dt time.Time
		if sub1 < 0 && sub2 < 0 {
			sub = sub3
			dt = dt3
		} else if (sub1 > 0 && sub2 < 0) || (sub1 > 0 && sub2 > 0 && sub1 < sub2) {
			sub = sub1
			dt = dt1
		} else {
			sub = sub2
			dt = dt2
		}

		log.Printf("Ближайший запуск %s", formatRunTime(dt))

		time.AfterFunc(sub, func() {
			schedCh <- struct{}{}
		})
		time.Sleep(sub)
	}
}
