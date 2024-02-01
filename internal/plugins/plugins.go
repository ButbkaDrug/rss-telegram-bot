package plugins

import (
	"fmt"
	"strings"

	"github.com/mmcdole/gofeed"
)

type Plugin interface {
    Parse(*gofeed.Item) string
}

type UpworkParser struct {
}

func(u UpworkParser) Parse(i *gofeed.Item) string {
    var title string

	body := strings.ReplaceAll(i.Content, "<br />", "\n")
	body = strings.ReplaceAll(body, "    ", "")
	body = strings.ReplaceAll(body, "&quot;", "\"")
	body = strings.ReplaceAll(body, "&rsquo;", "'")
	body = strings.ReplaceAll(body, "&ndash;", "-")

	idx := strings.LastIndex(i.Title, " - ")

    if idx > 1 {
        title = i.Title[:idx]
    }

	title = fmt.Sprintf("<b><u>%s</u></b>", title)

	msg := fmt.Sprintf("%s\n\n%s\n%s",
		title,
		body,
		i.Link,
	)

	return msg
}
