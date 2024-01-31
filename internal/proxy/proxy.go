package proxy

import "fmt"


// Proxy interface is a a wrapper for a super simple http proxy that routes
// your request through their server. Idea is that you'll feed you link to a thing
// and it will return you a link that already goes through a proxy.
type Proxy interface {
    Route()string
}

type SimpleProxy struct{
    key string
}

func NewProxy(key string) *SimpleProxy{
    return &SimpleProxy{
        key: key,
    }
}

func(s SimpleProxy) Route(url string) string {
	return fmt.Sprintf("http://api.scraperapi.com/?api_key=%s&url=%s", s.key, url)
}
