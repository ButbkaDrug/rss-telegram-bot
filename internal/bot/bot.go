package bot

import (
	"fmt"
	"log/slog"

	tblib "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"telebot/internal/handler"
)

type Bot struct {
	key      string
	api      *tblib.BotAPI
	logger   *slog.Logger
	handlers *handler.Handler
}

func NewBot(key string, l *slog.Logger) (*Bot, error) {
	api, err := tblib.NewBotAPI(key)

	if err != nil {
		return nil, fmt.Errorf("Cannot initialize bot: %w", err)
	}

	return &Bot{
		key:     key,
		api:     api,
		handler: handler.NewHandler(),
		logger:  l,
	}, nil
}

func (b *Bot) Serve() {

	u := tblib.NewUpdate(0)

	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		fmt.Printf("%+v\n", update)
	}
}
