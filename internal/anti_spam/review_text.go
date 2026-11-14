package anti_spam

import (
	"fmt"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/utils"
	"golang.org/x/net/html"
)

var reviewTextBlockElements = map[string]struct{}{
	"address": {}, "article": {}, "aside": {}, "blockquote": {}, "br": {},
	"dd": {}, "div": {}, "dl": {}, "dt": {}, "fieldset": {}, "figcaption": {},
	"figure": {}, "footer": {}, "form": {}, "h1": {}, "h2": {}, "h3": {},
	"h4": {}, "h5": {}, "h6": {}, "header": {}, "hr": {}, "li": {}, "main": {},
	"nav": {}, "ol": {}, "p": {}, "pre": {}, "section": {}, "table": {},
	"tbody": {}, "td": {}, "tfoot": {}, "th": {}, "thead": {}, "tr": {}, "ul": {},
}

// NormalizeReviewContent converts Markdown/HTML into the visible semantic text used by
// moderation checkers. It deliberately excludes URLs and Artalk emoticon metadata.
func NormalizeReviewContent(content string) (string, error) {
	marked, err := utils.Marked(content)
	if err != nil {
		return "", err
	}

	doc, err := html.Parse(strings.NewReader(marked))
	if err != nil {
		return "", err
	}

	var out strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		switch node.Type {
		case html.TextNode:
			out.WriteString(node.Data)
			return
		case html.CommentNode:
			return
		case html.ElementNode:
			tag := strings.ToLower(node.Data)
			if tag == "script" || tag == "style" {
				return
			}
			if tag == "img" {
				for _, attr := range node.Attr {
					if strings.EqualFold(attr.Key, "atk-emoticon") {
						return
					}
				}
				out.WriteString("[图片]")
				return
			}
			if _, isBlock := reviewTextBlockElements[tag]; isBlock {
				out.WriteByte(' ')
				defer out.WriteByte(' ')
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(doc)
	return strings.Join(strings.Fields(out.String()), " "), nil
}

func BuildReviewText(userName, reviewContent string) string {
	return fmt.Sprintf("昵称: %s\n评论: %s", userName, reviewContent)
}
