package anti_spam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeReviewContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "plain", input: "普通评论", expected: "普通评论"},
		{name: "markdown formatting", input: "# 标题\n\n**加粗**和普通文字", expected: "标题 加粗和普通文字"},
		{name: "markdown link", input: "[点击领取](https://spam.example/ad)", expected: "点击领取"},
		{name: "html link", input: `<a href="https://spam.example/ad">点击领取</a>`, expected: "点击领取"},
		{name: "markdown image", input: `![](https://spam.example/ad.jpg)`, expected: "[图片]"},
		{name: "html image", input: `<img src="https://spam.example/ad.jpg" alt="广告">`, expected: "[图片]"},
		{name: "emoticon", input: `<img src="https://owo.example/a.png" atk-emoticon="blobcat">`, expected: ""},
		{name: "split tag", input: `广<strong>告</strong>`, expected: "广告"},
		{name: "entity", input: `敏感&amp;内容`, expected: "敏感&内容"},
		{name: "invisible", input: `<script>alert(1)</script><style>.x{}</style><!--x-->正文`, expected: "正文"},
		{name: "whitespace", input: "第一行\n\n   第二行", expected: "第一行 第二行"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := NormalizeReviewContent(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestBuildReviewText(t *testing.T) {
	assert.Equal(t, "昵称: 张三\n评论: 正文", BuildReviewText("张三", "正文"))
}
