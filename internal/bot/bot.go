package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"telebot/internal/user"

	tblib "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//TODO: Persistant storage. So that i don't send messages in case of restart.
//TODO: Also, save users data, so that after restart I don't have to add all the links back
//TODO: user.feed should be a map?

type Bot struct {
	key         string
	api         *tblib.BotAPI
	logger      *slog.Logger
	activeUsers map[int64]*user.User
}

func NewBot(key string, l *slog.Logger) (*Bot, error) {
	api, err := tblib.NewBotAPI(key)

	if err != nil {
		return nil, fmt.Errorf("Cannot initialize bot: %w", err)
	}

	return &Bot{
		key:         key,
		api:         api,
		logger:      l,
		activeUsers: make(map[int64]*user.User),
	}, nil
}

func (b *Bot) Serve() {

	u := tblib.NewUpdate(0)

	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

    b.LoadUser()

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

func (b *Bot) updateHandler(update tgbotapi.Update) {
    log.Printf("NEW UPDATE:%+v\n", update)

	m := update.Message

	if m != nil && m.IsCommand() {
		switch m.Command() {
		case "start":
			b.StartCommandHandler(update)
		case "users":
			b.UsersCommandHandler(update)
		case "links":
			b.LinksCommandHandler(update)
		case "add":
			b.AddLinkCommandHandler(update)
		case "remove":
			b.RemoveLinkCommandHandler(update)
		case "usage":
            b.UsageCommandHandler(update)
        case "save":
            b.SaveUserDataHandler(update)
		}
	}
}

func(b *Bot) SaveUserDataHandler(update tgbotapi.Update) {
    type temp struct{
            // We will store by user id
            ID int64
            // We will map the feed url and fetching duration. We can secrifice the rest
            Links map[string]time.Duration
    }

    for _, user := range b.activeUsers {
        var links = make(map[string]time.Duration)

        for _, link := range user.Feed {
            links[link.URL] = link.Timer
        }

        //struct will temporarly store the most nessesery data.
        s := temp {
            ID: user.ID,
            Links: links,
        }

        err := StupidStore(strconv.FormatInt(user.ID, 10), s)

        if err != nil {
            b.logger.Error("faild to save users data", "err", err)
        }
    }
}

func(b *Bot) LoadUser(){
    type temp struct{
            // We will store by user id
            ID int64
            // We will map the feed url and fetching duration. We can secrifice the rest
            Links map[string]time.Duration
    }

    var users []temp

    entries, err := os.ReadDir(".users")

    if err != nil {
        b.logger.Error("failed to load users", "err", err)
    }

    for _, entry := range entries {
        if entry.IsDir(){
            continue
        }

        file := path.Join(".users", entry.Name())
        var user temp

        err := StupidRestore(file, &user)

        if err != nil{
            b.logger.Error("failed to restore user", "err", err)
        }


        users = append(users, user)

    }


    for _, entry := range users {

        user := b.GetUser(entry.ID)


        for link, duration := range entry.Links {
            user.AddLinkWithTimer(link, duration)
        }

        b.logger.Info("users loaded", "id", user.ID)

        b.fetchUpdates(user)
    }



}

func StupidStore(filename string, data any) error {
    //TODO: Figure out the proper location for the files and not to store them
    // wherever script runs

    file := filepath.Join(".users", filename)
    f, err := os.Create(file)

    if err != nil {
        return err
    }

    err = json.NewEncoder(f).Encode(data)

    if err != nil {
        return err
    }

    return nil
}

func StupidRestore(filename string, dest any) error {

    f, err := os.Open(filename)

    if err != nil {
        return err
    }

    err = json.NewDecoder(f).Decode(dest)

    if err != nil {
        return err
    }

    return nil
}

func (b *Bot) UsageCommandHandler(update tgbotapi.Update) {
	msg := MemUsage()

	b.SendTextMessage(update.FromChat().ID, msg)
}

// Creates a new user and adds to a userlist
func(b *Bot) newUser(id int64) *user.User {

    u := user.NewUser(id)
    b.activeUsers[id] = u

    return u
}

// Get New user from the users list or creates a new one if user do not exist
func (b *Bot) GetUser(id int64) *user.User {
	usr, ok := b.activeUsers[id]

	if !ok {
        usr = b.newUser(id)
	}

	return usr
}

func (b *Bot) RemoveLinkCommandHandler(update tgbotapi.Update) {
	id, err := strconv.Atoi(update.Message.CommandArguments())

	if err != nil {
		b.SendTextMessage(update.FromChat().ID, "# URL should be a nubmer. user /links to find out")
		return
	}

	user, ok := b.activeUsers[update.FromChat().ID]

	if !ok {
		b.SendTextMessage(update.FromChat().ID, "You're not in the userlist. Please, run a /start command")
		return
	}

	user.RemoveLink(id)

	b.SendTextMessage(update.FromChat().ID, "Link successfully removed")

	b.LinksCommandHandler(update)
    b.SaveUserDataHandler(update)
}

func (b *Bot) AddLinkCommandHandler(update tgbotapi.Update) {
    b.logger.Info("adding new link")
    id := update.FromChat().ID
	args := strings.Split(update.Message.CommandArguments(), " ")

	if len(args) < 1 {
        msg := "Seems like you forgot about to past a like. Please, try again"
		b.SendTextMessage(id, msg)
		return
	}

	link := args[0]

    user := b.GetUser(id)

	if len(args) == 2 {
		timer, err := strconv.Atoi(args[1])

		if err != nil {
            msg := "timer(as second argument) should be a number"
			b.SendTextMessage(id, msg)
			return
		}
		user.AddLinkWithTimer(link, time.Duration(timer)*time.Minute)
	} else {
		user.AddLink(link)
	}

	b.SendTextMessage(id, "Link, successfully added! You links are: ")
	b.LinksCommandHandler(update)

	b.fetchUpdates(user)
    b.SaveUserDataHandler(update)
}

func (b *Bot) LinksCommandHandler(update tgbotapi.Update) {

	var text string
	user, ok := b.activeUsers[update.FromChat().ID]

	if !ok {
		err := errors.New("User is not in a userlist")
		b.SendErrorMessage(update.FromChat().ID, err)
		return
	}

	for _, link := range user.Feed {
		if link == nil {
			continue
		}
		text += fmt.Sprintf("```%+v```\n", link)
	}

	if text == "" {
		text = "You havan't add any links yet"
	}

	b.SendTextMessage(update.FromChat().ID, text)
}

func (b *Bot) UsersCommandHandler(update tgbotapi.Update) {

	var text string

	for _, user := range b.activeUsers {
		text += fmt.Sprintf("%d: %v", user.ID, user.Feed)
	}

	if text == "" {
		text = "No active users yet"
	}

	b.SendTextMessage(update.FromChat().ID, text)

}

func (b *Bot) StartCommandHandler(update tgbotapi.Update) {
	var id = update.FromChat().ID

	if _, ok := b.activeUsers[id]; !ok {
        b.newUser(id)
        msg := "You've been added to a list of active users. Add RSS or Atom URLS to start receiving updates"
        b.SendTextMessage(id, msg)
        return
	}

    msg := "already on the list"
    b.SendTextMessage(id, msg)
    return
}

func (b *Bot) SendTextMessage(id int64, s string) {

	msg := tgbotapi.NewMessage(id, s)

	b.api.Send(msg)

}

func (b *Bot) SendErrorMessage(id int64, args ...interface{}) {
	b.logger.Error("error accured", args...)

	b.SendTextMessage(id, "I'm sorry, something went wrong and we can't procces your request for now. Please, try again later. If problem persists, contact bot administration.")
}

func (b *Bot) fetchUpdates(user *user.User) {
	b.logger.Info("Starting fetching for", "user", user)

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

func (b *Bot) CheckLink(user *user.User, link *user.Link) {
	b.logger.Info("CheckLink initialized", "url", link.URL)

	ticker := time.NewTicker(link.Timer)
	done := make(chan struct{})

    b.fetchLink(user, link)

	for {

		select {
		case <-done:
			return
        case <-ticker.C:
            b.fetchLink(user, link)
		}

	}
}

func (b *Bot) fetchLink(user *user.User, link *user.Link) {
    var msg string

    items, err := link.Fetch()
    b.logger.Info("Items found", "count", len(items))

    if err != nil {
        errMsg := `failed to check for an update for the url + ` + link.URL
        b.SendTextMessage(user.ID, errMsg)
        return
    }

    for _, item := range items {



        sanitized := strings.ReplaceAll(item.Description, "<br />", "\n")
        sanitized = strings.ReplaceAll(sanitized, "    ", "")

        idx := strings.LastIndex(item.Title, " - ")

        title := item.Title[:idx]
        title = fmt.Sprintf("<b><u>%s</u></b>", title)




        msg = fmt.Sprintf("%s\n\n%s\n%s",
            title,
            sanitized,
            item.Link,
        )


        m := tgbotapi.NewMessage(user.ID, msg)
        m.DisableWebPagePreview = true
        m.ParseMode = "HTML"

        _, err := b.api.Send(m)

        if err != nil {
            b.logger.Error("failed to send update", "error", err.Error())
        }

    }

}
