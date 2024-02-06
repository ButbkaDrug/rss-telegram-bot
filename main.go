package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"github.com/butbkadrug/rss-telegram-bot/internal/bot"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	var bot_api_key string
	var logger *slog.Logger

	data := time.DateTime
	filename := fmt.Sprintf("%s rss-bot-logs.log", data)
	logFile, err := os.Create(filename)

	if err != nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
		logger.Error("failed to creat log file, logging to stdout")
	} else {
		logger = slog.New(slog.NewTextHandler(logFile, nil))
	}

	err = godotenv.Load()

	if err != nil {
		logger.Warn("Can't load inviroment varialbes", "error", err.Error())
	}

	flag.StringVar(
		&bot_api_key,
		"key",
		os.Getenv("BOT_API_KEY"),
		"Provide bot api key. If you don't have one prompt @botfather chatbot",
	)

	flag.Parse()

	bot, err := bot.NewBot(bot_api_key, logger)

	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	bot.Serve()
}
