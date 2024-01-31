package models

import "time"

// Represents a link object
type Link struct {
    // Feed url will be used to fetch new items
	URL        string
    // Timeout will state how often we want to check particular feed
	Timeout    time.Duration
    // Will be mark true to be deleted.
	Expired    bool
    // Will be mark true, when link will be processed
	InProgress bool
    // Last time we fetched updates
	LastCheck  time.Time
}

// Returns pointer to a new link object with default timer(10min)
func NewLink(url string) *Link {
	return &Link{
		URL:       url,
		Timeout:     10 * time.Minute,
		LastCheck: time.Now().Add(time.Duration(-1) * time.Minute),
	}
}

// Returs pointer to a new link object and let's set timeout different to default(10min)
func NewLinkWithTimeout(url string, d time.Duration) *Link {
	return NewLink(url).SetTimeout(d)
}

// Let's you set timeout(update intervar) for the link. And returns pointer to
// itself
func (l *Link) SetTimeout(d time.Duration) *Link {
	l.Timeout = d
	return l
}
