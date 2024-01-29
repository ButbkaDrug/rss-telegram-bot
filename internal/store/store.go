package store

import "github.com/mmcdole/gofeed"

type Simplestore struct {
    store map[string]*gofeed.Item
}

func NewSimplestore()Simplestore {
    return Simplestore{
        store: make(map[string]*gofeed.Item),
    }
}

func(s Simplestore) Save(i *gofeed.Item) {
    s.store[i.GUID] = i
}

func(s Simplestore) Contains(i *gofeed.Item) bool {
    _, ok := s.store[i.GUID]

    return ok
}
