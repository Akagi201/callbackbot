package main

import (
	log "github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	bot *tgbotapi.BotAPI
)

func setUpdates() <-chan tgbotapi.Update {
	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = 60
	updates, err := bot.GetUpdatesChan(upd)
	if err != nil {
		log.Fatalf("Telegram bot get update failed, err: %v", err)
	}
	return updates
}

func message(m *tgbotapi.Message) {
}

func inline(iq *tgbotapi.InlineQuery) {
}

func chosenResult(cr *tgbotapi.ChosenInlineResult) {
}

func callback(cq *tgbotapi.CallbackQuery) {
}

func channelPost(m *tgbotapi.Message) {
}

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI(opts.BotToken)
	if err != nil {
		log.Fatalf("Create telegram bot api failed, err: %v", err)
	}
	log.Infof("Authorized telegram robot as @%v", bot.Self.UserName)

	bot.Debug = opts.Debug

	updates := make(<-chan tgbotapi.Update)
	updates = setUpdates()

	for upd := range updates {
		switch {
		case upd.Message != nil:
			go message(upd.Message)
		case upd.InlineQuery != nil && len(upd.InlineQuery.Query) <= 255: // Just don't update results if query exceeds the maximum length
			go inline(upd.InlineQuery)
		case upd.ChosenInlineResult != nil:
			go chosenResult(upd.ChosenInlineResult)
		case upd.CallbackQuery != nil:
			go callback(upd.CallbackQuery)
		case upd.ChannelPost != nil:
			go channelPost(upd.ChannelPost)
		}
	}
}
