package anti_spam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAkismetReqParamsUsesUnifiedReviewText(t *testing.T) {
	params := &CheckerParams{
		BlogURL:    "https://example.com",
		RawContent: "raw body",
		ReviewText: "昵称: 测试用户\n评论: visible body",
		UserName:   "测试用户",
		UserEmail:  "user@example.com",
		UserIP:     "127.0.0.1",
		UserAgent:  "test-agent",
	}

	req := newAkismetReqParams(params)

	assert.Equal(t, params.ReviewText, req.CommentContent)
	assert.Equal(t, params.UserName, req.CommentAuthor)
	assert.NotEqual(t, params.RawContent, req.CommentContent)
}
