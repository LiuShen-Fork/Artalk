package anti_spam

import "strings"

// CommentInterceptor performs a synchronous local keyword check before a comment is saved.
type CommentInterceptor struct {
	keywords []string
}

func NewCommentInterceptor(keywords string) *CommentInterceptor {
	items := strings.Split(keywords, ",")
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if keyword := strings.TrimSpace(item); keyword != "" {
			filtered = append(filtered, keyword)
		}
	}
	return &CommentInterceptor{keywords: filtered}
}

func (c *CommentInterceptor) Check(p *CheckerParams) (bool, string) {
	for _, keyword := range c.keywords {
		if strings.Contains(p.UserName, keyword) || strings.Contains(p.ReviewContent, keyword) {
			return true, keyword
		}
	}
	return false, ""
}

func (c *CommentInterceptor) Keywords() []string {
	return append([]string(nil), c.keywords...)
}
