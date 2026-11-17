package anti_spam

import "strings"

// CommentInterceptor performs a synchronous local keyword check before a comment is saved.
type CommentInterceptor struct {
	keywords []string
}

func NewCommentInterceptor(keywords string) *CommentInterceptor {
	items := strings.Split(keywords, ",")
	filtered := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		keyword := strings.TrimSpace(item)
		if keyword == "" {
			continue
		}
		if _, exists := seen[keyword]; exists {
			continue
		}
		seen[keyword] = struct{}{}
		filtered = append(filtered, keyword)
	}
	return &CommentInterceptor{keywords: filtered}
}

func (c *CommentInterceptor) Check(p *CheckerParams) (bool, string) {
	// Links are checked as their own fields: the visible text a keyword filter
	// sees has every URL stripped out of it.
	for _, keyword := range c.keywords {
		if strings.Contains(p.UserName, keyword) || strings.Contains(p.ReviewContent, keyword) {
			return true, keyword
		}
		for _, link := range p.Links {
			if strings.Contains(link, keyword) {
				return true, keyword
			}
		}
	}
	return false, ""
}

func (c *CommentInterceptor) Keywords() []string {
	return append([]string(nil), c.keywords...)
}
