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

type Storage interface {
	Read() ([]byte, error)
	Write([]byte) error
}

// Core of the application. Ties all the components together
type Bot struct {
    // Telegram bot API key. In case we would need it for some reason
	key         string
    // Telegram bot API client
	api         *tgbotapi.BotAPI
    // Logger for logging what's going on
	logger      *slog.Logger
    // Storage interface for saving users data
	store       Storage
    // RSS parser for parsing feeds. Generic parser by default
	parser      *gofeed.Parser
    // List of active users
	activeUsers models.Users
}

// Creates a new bot instance with given API key and logger
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

// Starts the main loop and handles all the updates from the Telegram API
func (b *Bot) Serve() {

	u := tgbotapi.NewUpdate(0)

	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

    // Load users from the storage before starting the main loop
	b.LoadUsers()

	for update := range updates {
		b.updateHandler(update)
	}
}

// Helper function that prints out the memory usage
// TODO: remove this function
func MemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf("Alloc = %v MiB", m.Alloc/1024/1024) +
		fmt.Sprintf("HeapAlloc = %v MiB", m.HeapAlloc/1024/1024) +
		fmt.Sprintf("\tTotalAlloc = %v MiB", m.TotalAlloc/1024/1024) +
		fmt.Sprintf("\tSys = %v MiB", m.Sys/1024/1024) +
		fmt.Sprintf("\tNumGC = %v\n", m.NumGC)
}

// Loads users from the storage and unmarshals them into the activeUsers list
func (b *Bot) LoadUsers() {

	data, err := b.store.Read()
	err = json.Unmarshal(data, &b.activeUsers)

	if err != nil {
		b.logger.Error("failed to unmarshal users", "err", err)
		return
	}
}

// Marshals users data ad saves to the storage
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

// Helper function to send a text message to the user
func (b *Bot) SendTextMessage(id int64, s string) {

	msg := tgbotapi.NewMessage(id, s)

	b.api.Send(msg)

}

// Helper function to send an error message to the user
//TODO: check if I even end up using this function
func (b *Bot) SendErrorMessage(id int64, args ...interface{}) {
	b.logger.Error("error accured", args...)

	b.SendTextMessage(id, "I'm sorry, something went wrong and we can't procces your request for now. Please, try again later. If problem persists, contact bot administration.")
}


// Starts fetching feeds updates for the user
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

// Checks the timeout and fetches updates for the user
// Also saves the user data to the storage after fetching
func (b *Bot) CheckLink(user *models.User, link *models.Link) {
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

// Fetching feed implementation
//TODO: apply Uwork formatter only when updates come from upwork
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

// Platform specific formatter for Upwork RSS feed
// It will get rid of all the HTML tags
// And should cleanup the text a bit
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
