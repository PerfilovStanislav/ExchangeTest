package main

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
)

var tgBot TgBot

type TgBot struct {
	*tg.BotAPI
	Channel int64
}

func (bot *TgBot) init() {
	bot.BotAPI, _ = tg.NewBotAPI(os.Getenv("tg.token"))
	bot.Channel = s2i(os.Getenv("tg.channel"))
	bot.Debug = false
}

func (bot *TgBot) sendTestFile(file string) {
	msg := tg.NewMediaGroup(tgBot.Channel, []interface{}{
		tg.NewInputMediaDocument(tg.FilePath(file)),
		tg.NewInputMediaDocument(tg.FilePath(".env")),
	})

	_, _ = tgBot.Send(msg)
}
