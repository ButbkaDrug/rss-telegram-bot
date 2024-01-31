package models

import (
	"encoding/json"
	"time"
)

type Users map[int64]*User

type User struct {
	// ID should be unique. Intension is to use telegram's user id. In oreder
	// to avoid managin them by hand.
	ID int64
	// Contains links rss/atom/json feeds.
	Feed []*Link
	// Contains GUID's of already received news. As it allows to avoid receiving
	// same article agaig. Works well for personal use. Can turn into memory hog
	// if used by many users
	Store map[string]struct{}
}

// intermediate structure that allows easy conversion of the user struct to a
// JSON.
type tempUser struct {
    // original user ID with reamin untuched
	ID    int64                    `json:"id"`
    // We will store a link struct as a url:duration pair. everything else is
    // irrelevant
	Feed  map[string]time.Duration `json:"feed"`
    // Will store store as is
	Store map[string]struct{}      `json:"store"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	tf := make(map[string]time.Duration)

	for _, link := range u.Feed {
		tf[link.URL] = link.Timeout
	}
	temp := tempUser{
		ID:    u.ID,
		Feed:  tf,
		Store: u.Store,
	}

	return json.Marshal(temp)
}

// Since my dump ass couldn't be botherd doing everything properly. I have made
// custom marshaler for converting user struct into a JSON object through an
// intermediate structure.
func (u *User) UnmarshalJSON(data []byte) error {
	var temp tempUser

	err := json.Unmarshal(data, &temp)

	if err != nil {
		return err
	}

	u.ID = temp.ID

	for link, d := range temp.Feed {
		u.Feed = append(u.Feed, NewLinkWithTimeout(link, d))
	}

	u.Store = make(map[string]struct{})

	for guid := range temp.Store {
		u.Store[guid] = struct{}{}
	}

	return nil
}

// Creates a new user instance
func NewUser(id int64) *User {
	return &User{
		ID:    id,
		Feed:  make([]*Link, 0),
		Store: make(map[string]struct{}),
	}
}

// Removes a link from a links list by a link number. You can see links using
// /links command(if used within telegram)
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
// Add's a link with a default parametrs  to a list if url is not yet present.
// otherwise no op
func (u *User) AddLink(url string) {

	for _, link := range u.Feed {

		if url == link.URL {
			return
		}
	}

	u.Feed = append(u.Feed, NewLink(url))
}

// Add's a link with a custom timeout. If link already exists updates timeout
// otherwise no op
func (u *User) AddLinkWithTimeout(url string, d time.Duration) {
	for i, link := range u.Feed {

		if url == link.URL && d == link.Timeout {
			return
		}

		if url == link.URL {
			u.Feed[i].Timeout = d
			return
		}
	}

	u.Feed = append(u.Feed, NewLinkWithTimeout(url, d))
}
