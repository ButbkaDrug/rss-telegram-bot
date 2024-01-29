package parser

import (
	"github.com/mmcdole/gofeed"
)

type Storer interface{
    Save(*gofeed.Item)
    Contains(*gofeed.Item) bool
}

type Checker struct {
    Store Storer
    Parser *gofeed.Parser
}

func NewChecker(s Storer) *Checker {
    return &Checker{
        Store: s,
        Parser: gofeed.NewParser(),
    }
}

func(c *Checker) CheckFeed(url string) ([]*gofeed.Item, error) {

	var new []*gofeed.Item

	feed, err := c.Parser.ParseURL(url)

	if err != nil {
		return new, err
	}

	for _, item := range feed.Items {
        if c.Store.Contains(item) {
            continue
        }

        c.Store.Save(item)

		new = append(new, item)
	}

	return new, nil
}
