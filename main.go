package main

import (
	"fmt"
	"os"
	"vak-parser/bot"
	"vak-parser/common"
	"vak-parser/database"
	"vak-parser/parser"
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
	go parser.Parse(botIn, botOut, db)
	go bot.RunBot(botIn, botOut, db)

	fmt.Scanln()
}
