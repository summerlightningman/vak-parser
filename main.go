package main

import (
	"fmt"
	"os"

	"vak-parser/bot"
	"vak-parser/common"
	"vak-parser/database"
	"vak-parser/parser"
	"vak-parser/scheduler"
)

func main() {
	db, err := database.Open("./.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка БД: %v\n", err)
		return
	}
	defer db.Close()

	botIn := make(chan common.BotMsg, 1)
	botOut := make(chan common.BotMsg, 1)
	schedCh := make(chan struct{})
	go parser.Parse(botIn, botOut, schedCh, db)
	go bot.RunBot(botIn, botOut, db)
	go scheduler.Scheduler(schedCh)

	fmt.Scanln()
}
