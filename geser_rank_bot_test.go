package main

import (
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	TestToken = ""
	ChatID    = 1
)

func getBot(t *testing.T) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(TestToken)
	bot.Debug = true

	if err != nil {
		t.Error(err)
		t.Fail()
	}

	return bot, err
}

func TestNormalPlusMessage(t *testing.T) {
	bot, _ := getBot(t)
	logger := configureLogger()
	mutex := &sync.RWMutex{}
	// msg := tgbotapi.NewMessage(ChatID, "A test message from the test library in telegram-bot-api")
	// msg.ParseMode = "markdown"
	// _, err := bot.Send(msg)

	processMessage(logger, bot, mutex, update.Message)

	if err != nil {
		t.Error(err)
		t.Fail()
	}
}
