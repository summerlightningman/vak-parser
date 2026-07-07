package bot

import (
	"context"
	"fmt"
	"log"
	"os"

	"vak-parser/common"
	"vak-parser/database"

	"github.com/go-telegram/bot"
	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var logBot = log.New(os.Stderr, "[bot] ", log.LstdFlags)

func botInListener(b *tg.Bot, botIn <-chan common.BotMsg, db *database.DbAdapter) {
	for msg := range botIn {
		switch msg.Type {
		case common.BotMsgTypeSuccess:
			subscribers, err := db.ListSubscribers()
			if err != nil {
				logBot.Printf("ошибка чтения подписчиков: %v", err)
				continue
			}

			text := formatSuccessMessage(msg.SuccessPayload)
			for _, chatID := range subscribers {
				_, err := b.SendMessage(context.Background(), &tg.SendMessageParams{
					ChatID: chatID,
					Text:   text,
				})
				if err != nil {
					logBot.Printf("ошибка отправки в chat %d: %v", chatID, err)
				}
			}
		}
	}
}

func formatSuccessMessage(payload common.SuccessPayload) string {
	if payload.Page >= 0 {
		return fmt.Sprintf("Найдено совпадение:\n%s\nстраница %d", payload.Url, payload.Page)
	}
	return fmt.Sprintf("Найдено совпадение в тегах:\n%s", payload.Url)
}

func RunBot(botIn <-chan common.BotMsg, botOut chan<- common.BotMsg, db *database.DbAdapter) {
	opts := []tg.Option{
		tg.WithDefaultHandler(handler(botOut, db)),
	}

	b, err := bot.New(os.Getenv("BOT_TOKEN"), opts...)
	if err != nil {
		logBot.Printf("ошибка создания бота: %v", err)
		return
	}

	b.RegisterHandler(tg.HandlerTypeMessageText, "/unsubscribe", tg.MatchTypeExact, unsubscribeHandler(db))

	go botInListener(b, botIn, db)

	b.Start(context.TODO())
}

func unsubscribeHandler(db *database.DbAdapter) tg.HandlerFunc {
	return func(ctx context.Context, b *tg.Bot, update *models.Update) {
		db.RemoveSubscriber(update.Message.Chat.ID)

		b.SendMessage(ctx, &tg.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Вы отписались от рассылки",
		})
	}
}

func handler(botOut chan<- common.BotMsg, db *database.DbAdapter) tg.HandlerFunc {
	return func(ctx context.Context, b *tg.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		if err := db.AddSubscriber(update.Message.Chat.ID); err != nil {
			logBot.Printf("ошибка сохранения подписчика %d: %v", update.Message.Chat.ID, err)
		}

		botOut <- common.BotMsg{
			Type: common.BotMsgTypeParse,
		}
		logBot.Println("получено сообщение о парсинге")
	}
}
