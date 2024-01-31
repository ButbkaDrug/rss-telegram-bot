package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)


// Handles updates that comes from telegram.
func (b *Bot) updateHandler(update tgbotapi.Update) {

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
        case "set":
            b.SetCommandHandler(update)
		}
	}

}

// Sets new timeout on existing feed
func(b *Bot) SetCommandHandler(update tgbotapi.Update){
    id := update.Message.Chat.ID
    args := strings.Split(update.Message.CommandArguments(), " ")

    if len(args) < 2 {
        b.SendTextMessage(id, `
Please, provide 2 intagers:
feedID - to know which feed you want to edit, N for update timer in minues.
In this example we will set feed 0 to update every 10 minutes

EXAMPLE:
/set 0 10
`)

        return
    }

    feedID, feedErr := strconv.Atoi(args[0])
    timeout, timeErr := strconv.Atoi(args[1])

    if feedErr != nil && timeErr != nil {
        b.SendTextMessage(id, "FeedID and Timer should be a number")
    }

    user := b.GetUser(id)

    if feedID >= len(user.Feed) || feedID < 0 {
        b.SendTextMessage(id, "there is no feed with id: "+ args[0])
        return
    }

    if timeout < 1 {
        b.SendTextMessage(id, "timeout should be at least a minute")
        return
    }

    user.Feed[feedID].SetTimeout(time.Duration(timeout)*time.Minute)

    b.SendTextMessage(id, "Your feed has been successfully set!")
}

// Saves user data using. You shoul'd have to use it manually, but you can
func(b *Bot) SaveUserDataHandler(update tgbotapi.Update) {
    b.logger.Info("saving users data")
    b.saveUsers()
}

// Prints out system resources consumed by the app. Why? Well... I think it's
// helpful
func (b *Bot) UsageCommandHandler(update tgbotapi.Update) {
	msg := MemUsage()

	b.SendTextMessage(update.FromChat().ID, msg)
}

// Removes link from the list of feeds
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

// Add's a new feed to a list
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
		user.AddLinkWithTimeout(link, time.Duration(timer)*time.Minute)
	} else {
		user.AddLink(link)
	}

	b.SendTextMessage(id, "Link, successfully added! You links are: ")
	b.LinksCommandHandler(update)

	b.fetchUpdates(user)
    b.SaveUserDataHandler(update)
}

// Prints out feeds list
func (b *Bot) LinksCommandHandler(update tgbotapi.Update) {

	var text string
	user, ok := b.activeUsers[update.FromChat().ID]

	if !ok {
		err := fmt.Errorf("User is not in a userlist")
		b.SendErrorMessage(update.FromChat().ID, err)
		return
	}

	for id, link := range user.Feed {
        text += fmt.Sprintf("FeedID: %d\n", id)
        text += fmt.Sprintf("URL: %s\n", link.URL)
        text += fmt.Sprintf("Checked once every %v mins\n", link.Timeout.Minutes())
        text += fmt.Sprintf("Last Check: %d min %d s ago\n\n",
            int(time.Since(link.LastCheck).Minutes()),
            int(time.Since(link.LastCheck).Seconds()) % 60,
        )
	}

	if text == "" {
		text = "You havan't add any links yet"
	}

	b.SendTextMessage(update.FromChat().ID, text)
}

// List's all users who using the bot.
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

// Prints wellcome message and should tell about capabilities
func (b *Bot) StartCommandHandler(update tgbotapi.Update) {
	var id = update.FromChat().ID
    msg := `
Hi, I'm gonna help you to agrigate your rss/atom/json feeds:

/links - will show you all your feeds

/add URL - will add RSS, Atom or JSON feed URLs to start receiving updates.

/add URL NUMBER - if you want custom timeout(default 10min), where URL is your feed url and NUMBER is an intager that represents an interval

/remove feedID - will remove url with specified ID from feeds(for feedID call /links)
`

	if _, ok := b.activeUsers[id]; !ok {
        b.newUser(id)
        return
	}

    b.SendTextMessage(id, msg)
    return
}
