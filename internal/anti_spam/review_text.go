package anti_spam

import (
	"fmt"
	"regexp"
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

func BuildReviewText(userName, reviewContent string, links ...string) string {
	text := fmt.Sprintf("昵称: %s\n评论: %s", userName, reviewContent)
	if len(links) > 0 {
		text += "\n链接: " + strings.Join(links, " ")
	}
	return text
}

const (
	// reviewLinkMaxCount caps how many targets are handed to the checkers, so a
	// link flood cannot blow up the review payload.
	reviewLinkMaxCount = 5

	// reviewLinkMaxLen keeps a single hostile link from dominating the payload.
	reviewLinkMaxLen = 200
)

var reviewLinkPattern = regexp.MustCompile("https?://[^\\s<>\"'`]+")

// ExtractReviewLinks returns the outbound targets a comment carries: anchor
// hrefs, image sources and bare URLs left in the text. NormalizeReviewContent
// drops them on purpose because they are not part of the visible prose, but
// advertising spam hides exactly there, so moderation receives them separately.
func ExtractReviewLinks(content string) []string {
	marked, err := utils.Marked(content)
	if err != nil {
		marked = content
	}

	found := []string{}
	text := strings.Builder{}

	if doc, parseErr := html.Parse(strings.NewReader(marked)); parseErr == nil {
		collectLinksOutsideReviewText(doc, &found, &text)
	} else {
		text.WriteString(content)
	}

	// Bare URLs that were never turned into an anchor still matter, so the
	// visible text is scanned for the same pattern.
	found = append(found, reviewLinkPattern.FindAllString(text.String(), -1)...)

	seen := make(map[string]struct{}, len(found))
	links := make([]string, 0, len(found))
	for _, link := range found {
		link = strings.Trim(strings.TrimSpace(link), ".,;:!?)]}>\"'")
		if link == "" {
			continue
		}
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		if r := []rune(link); len(r) > reviewLinkMaxLen {
			link = string(r[:reviewLinkMaxLen])
		}
		links = append(links, link)
		if len(links) >= reviewLinkMaxCount {
			break
		}
	}
	return links
}

func collectLinksOutsideReviewText(node *html.Node, found *[]string, text *strings.Builder) {
	switch node.Type {
	case html.TextNode:
		text.WriteString(node.Data)
		text.WriteByte(' ')
		return
	case html.CommentNode:
		return
	case html.ElementNode:
		tag := strings.ToLower(node.Data)
		if tag == "script" || tag == "style" {
			return
		}
		for _, attr := range node.Attr {
			switch strings.ToLower(attr.Key) {
			case "href", "src":
				if v := strings.TrimSpace(attr.Val); v != "" {
					*found = append(*found, v)
				}
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectLinksOutsideReviewText(child, found, text)
	}
}
