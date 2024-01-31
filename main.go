package main

import(
    "fmt"
    "runtime"
    "log/slog"
    "os"
    "flag"
    "github.com/joho/godotenv"
    "telebot/internal/bot"
)

func main () {
    var bot_api_key string
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    err := godotenv.Load()

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

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB", m.Alloc / 1024 / 1024)
	fmt.Printf("\tTotalAlloc = %v MiB", m.TotalAlloc / 1024 / 1024)
	fmt.Printf("\tSys = %v MiB", m.Sys / 1024 / 1024)
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

