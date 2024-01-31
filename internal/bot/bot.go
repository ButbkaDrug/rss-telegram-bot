package bot

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"telebot/internal/models"
	"telebot/internal/store"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mmcdole/gofeed"
)

//TODO: Persistant storage. So that i don't send messages in case of restart.
//TODO: user.feed should be a map?

type Storage interface {
	Read() ([]byte, error)
	Write([]byte) error
}

type Bot struct {
	key         string
	api         *tgbotapi.BotAPI
	logger      *slog.Logger
	store       Storage
	parser      *gofeed.Parser
	activeUsers models.Users
}

func NewBot(key string, l *slog.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(key)

	s := store.NewSimplestore()
	p := gofeed.NewParser()

	if err != nil {
		return nil, fmt.Errorf("Cannot initialize bot: %w", err)
	}

	return &Bot{
		key:         key,
		api:         api,
		logger:      l,
		store:       s,
		parser:      p,
		activeUsers: make(models.Users),
	}, nil
}

func (b *Bot) Serve() {

	u := tgbotapi.NewUpdate(0)

	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	b.LoadUsers()

	for update := range updates {
		b.updateHandler(update)
	}
}

func MemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf("Alloc = %v MiB", m.Alloc/1024/1024) +
		fmt.Sprintf("HeapAlloc = %v MiB", m.HeapAlloc/1024/1024) +
		fmt.Sprintf("\tTotalAlloc = %v MiB", m.TotalAlloc/1024/1024) +
		fmt.Sprintf("\tSys = %v MiB", m.Sys/1024/1024) +
		fmt.Sprintf("\tNumGC = %v\n", m.NumGC)
}

func (b *Bot) LoadUsers() {

	data, err := b.store.Read()

	err = json.Unmarshal(data, &b.activeUsers)

	if err != nil {
		b.logger.Error("failed to unmarshal users", "err", err)
		return
	}
}

func (b *Bot) saveUsers() {
	data, err := json.Marshal(b.activeUsers)

	if err != nil {
		b.logger.Error("failed to marshal users data", "err", err.Error())
		return
	}

	err = b.store.Write(data)

	if err != nil {
		b.logger.Error("failed to write user data", "err", err.Error())
		return
	}
}

// Creates a new user and adds to a userlist
func (b *Bot) newUser(id int64) *models.User {

	u := models.NewUser(id)
	b.activeUsers[id] = u

	return u
}

// Get New user from the users list or creates a new one if user do not exist
func (b *Bot) GetUser(id int64) *models.User {
	usr, ok := b.activeUsers[id]

	if !ok {
		usr = b.newUser(id)
	}

	return usr
}

func (b *Bot) SendTextMessage(id int64, s string) {

	msg := tgbotapi.NewMessage(id, s)

	b.api.Send(msg)

}

func (b *Bot) SendErrorMessage(id int64, args ...interface{}) {
	b.logger.Error("error accured", args...)

	b.SendTextMessage(id, "I'm sorry, something went wrong and we can't procces your request for now. Please, try again later. If problem persists, contact bot administration.")
}

func (b *Bot) fetchUpdates(user *models.User) {
	b.logger.Info("starting fetching for", "user", user.ID)

	for i, link := range user.Feed {
		if link == nil {
			continue
		}

		if link.Expired {
			user.RemoveLink(i)
		}

		if !link.InProgress {
			link.InProgress = true
			go b.CheckLink(user, link)
		}
	}
}

func (b *Bot) CheckLink(user *models.User, link *models.Link) {
	b.logger.Info("CheckLink initialized", "url", link.URL)

	ticker := time.NewTicker(link.Timeout)
	done := make(chan struct{})

	b.fetchLink(user, link)

	for {

		select {
		case <-done:
			return
		case <-ticker.C:
			b.fetchLink(user, link)
			b.saveUsers()
		}

	}
}

func (b *Bot) fetchLink(user *models.User, link *models.Link) {

	for _, link := range user.Feed {
		feed, err := b.parser.ParseURL(link.URL)

		if err != nil {
			errMsg := `failed to check for an update for the url + ` + link.URL
			b.SendTextMessage(user.ID, errMsg)
			return
		}

		for _, item := range feed.Items {
			if _, ok := user.Store[item.GUID]; ok {
				continue
			}

			user.Store[item.GUID] = struct{}{}

			msg := b.UpworkToTelegramFormatter(item)

			m := tgbotapi.NewMessage(user.ID, msg)
			m.DisableWebPagePreview = true
			m.ParseMode = "HTML"

			_, err := b.api.Send(m)

			if err != nil {
				b.logger.Error("failed to send update", "error", err.Error())
			}
		}

	}

}

func (b *Bot) UpworkToTelegramFormatter(item *gofeed.Item) string {
	sanitized := strings.ReplaceAll(item.Description, "<br />", "\n")
	sanitized = strings.ReplaceAll(sanitized, "    ", "")
	sanitized = strings.ReplaceAll(sanitized, "&quot;", "\"")
	sanitized = strings.ReplaceAll(sanitized, "&rsquo;", "'")
	sanitized = strings.ReplaceAll(sanitized, "&ndash;", "-")

	idx := strings.LastIndex(item.Title, " - ")

	title := item.Title[:idx]
	title = fmt.Sprintf("<b><u>%s</u></b>", title)

	msg := fmt.Sprintf("%s\n\n%s\n%s",
		title,
		sanitized,
		item.Link,
	)

	return msg
}
