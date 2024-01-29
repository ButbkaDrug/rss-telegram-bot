package user

import (
	"time"

	"github.com/mmcdole/gofeed"
)

type User struct {
	ID   int64
	Feed []*Link
}

func NewUser(id int64) *User {
	return &User{
		ID:   id,
		Feed: make([]*Link, 0),
	}
}

func (u *User) RemoveLink(n int) {

	if len(u.Feed) == 1 {
		u.Feed = make([]*Link, 0)
	}

	if n >= len(u.Feed) {
		return
	}

	new := make([]*Link, 0)
	new = append(new, u.Feed[:n]...)
	new = append(new, u.Feed[n+1:]...)

	u.Feed = new
}

func (u *User) AddLink(url string) {

	for _, link := range u.Feed {

		if url == link.URL {
			return
		}
	}

	u.Feed = append(u.Feed, NewLink(url))
}

func (u *User) AddLinkWithTimer(url string, d time.Duration) {
	for i, link := range u.Feed {

		if url == link.URL && d == link.Timer {
			return
		}

		if url == link.URL {
			u.Feed[i].Timer = d
			return
		}
	}

	u.Feed = append(u.Feed, NewLinkWithTimer(url, d))
}












type Link struct {
	URL        string
	Timer      time.Duration
	Expired    bool
	InProgress bool
	Store      map[string]*gofeed.Item
	Parser     *gofeed.Parser
	LastCheck  time.Time
}

// Returns a new link object with default timeer
func NewLink(url string) *Link {
	return &Link{
		URL:       url,
		Timer:     10 * time.Minute,
		Parser:    gofeed.NewParser(),
		Store:     make(map[string]*gofeed.Item),
		LastCheck: time.Now().Add(time.Duration(-1) * time.Minute),
	}
}

func NewLinkWithTimer(url string, d time.Duration) *Link {
	return NewLink(url).SetTimer(d)
}


func (l *Link) SetTimer(d time.Duration) *Link {
	l.Timer = d
	return l
}

func(l *Link) SetParser(p *gofeed.Parser) *Link {
    l.Parser = p
    return l
}

func(l *Link) Fetch() ([]*gofeed.Item, error) {

    l.LastCheck = time.Now()

    feed, err := l.Parser.ParseURL(l.URL)

    if err != nil {
        return nil, err
    }

    var result []*gofeed.Item

    for _, item := range feed.Items {

        if _, ok := l.Store[item.GUID]; ok {
            continue
        }

        l.Store[item.GUID] = item

        result = append(result, item)
    }

    return result, nil
}

