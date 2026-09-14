package sender

import (
	"strings"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/i18n"
)

func init() {
	// 单元测试中直接载入中文文案，保证断言可读
	i18n.Locales = map[string]string{
		"New comment from {{nick}}": "来自 {{nick}} 的新评论",
		"Post":                      "文章",
		"View details":              "查看详情",
		"Image":                     "图片",
		"Emoticon":                  "表情",
	}
}

func TestEmoticonMainTheme(t *testing.T) {
	tests := []struct {
		name string
		give string
		want string
	}{
		{name: "with prefix", give: "liushen-迷惑", want: "迷惑"},
		{name: "with multi dash", give: "a-b-大哭", want: "大哭"},
		{name: "without dash", give: "blobcat", want: "blobcat"},
		{name: "trailing dash", give: "liushen-", want: "liushen-"},
		{name: "empty", give: "", want: "表情"},
		{name: "spaces", give: "  ", want: "表情"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := emoticonMainTheme(tt.give); got != tt.want {
				t.Errorf("emoticonMainTheme(%q) = %q, want %q", tt.give, got, tt.want)
			}
		})
	}
}

func TestCommentContentToText(t *testing.T) {
	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "plain text",
			give: "Hello World",
			want: "Hello World",
		},
		{
			name: "mixed text and emoticon",
			give: `这字体残缺的有点厉害啊<img src="https://owo.liiiu.cn/liushen/liushen-confused.png" atk-emoticon="liushen-迷惑">`,
			want: "这字体残缺的有点厉害啊[迷惑]",
		},
		{
			name: "regular html image becomes placeholder",
			give: `看这张图 <img src="https://example.com/pic.png">`,
			want: "看这张图 [图片]",
		},
		{
			name: "markdown image becomes placeholder",
			give: `看这张图 ![alt](https://example.com/pic.png)`,
			want: "看这张图 [图片]",
		},
		{
			name: "empty emoticon key falls back to emoticon",
			give: `<img src="https://example.com/pic.png" atk-emoticon="">`,
			want: "[表情]",
		},
		{
			name: "markdown syntax preserved",
			give: "**bold** and `code` and [link](https://artalk.js.org)",
			want: "**bold** and `code` and [link](https://artalk.js.org)",
		},
		{
			name: "unsupported italic kept as-is",
			give: "an *italic* word",
			want: "an *italic* word",
		},
		{
			name: "list preserved",
			give: "- item A\n- item B",
			want: "- item A\n- item B",
		},
		{
			name: "blockquote preserved",
			give: "> quoted text",
			want: "> quoted text",
		},
		{
			name: "paragraphs become lines",
			give: "first paragraph\n\nsecond paragraph",
			want: "first paragraph\nsecond paragraph",
		},
		{
			name: "trailing spaces trimmed",
			give: "line one   \nline two",
			want: "line one\nline two",
		},
		{
			name: "empty",
			give: "   ",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CommentContentToText(tt.give); got != tt.want {
				t.Errorf("CommentContentToText(%q) =\n%q\nwant\n%q", tt.give, got, tt.want)
			}
		})
	}
}

func TestBuildWecomMarkdown(t *testing.T) {
	got := BuildWecomMarkdown(
		"Hexvork",
		"文章名称",
		"https://example.com/post/1",
		"这是评论内容\n第二行\n<img src=\"https://owo.liiiu.cn/liushen/liushen-confused.png\" atk-emoticon=\"liushen-迷惑\"> ![pic](https://example.com/a.png)",
		"https://example.com/post/1?atk_comment=1",
	)

	want := strings.Join([]string{
		"来自 Hexvork 的新评论",
		"> 文章： [文章名称](https://example.com/post/1)",
		"> 这是评论内容",
		"> 第二行",
		"> [迷惑] [图片]",
		"",
		"[查看详情](https://example.com/post/1?atk_comment=1)",
	}, "\n")

	if got != want {
		t.Errorf("BuildWecomMarkdown() =\n%s\nwant\n%s", got, want)
	}
}

func TestBuildWecomMarkdownWithoutPageURL(t *testing.T) {
	got := BuildWecomMarkdown("Hexvork", "文章名称", "", "评论内容", "")

	want := strings.Join([]string{
		"来自 Hexvork 的新评论",
		"> 文章： 文章名称",
		"> 评论内容",
	}, "\n")

	if got != want {
		t.Errorf("BuildWecomMarkdown() =\n%s\nwant\n%s", got, want)
	}
}

func TestBuildWecomMarkdownEscapesLinkText(t *testing.T) {
	got := BuildWecomMarkdown("Nick", "标题[with]brackets", "https://example.com", "hi", "")

	if !strings.Contains(got, "[标题【with】brackets](https://example.com)") {
		t.Errorf("link text should be escaped, got:\n%s", got)
	}
}

func TestTruncateWecomMarkdown(t *testing.T) {
	t.Run("short string untouched", func(t *testing.T) {
		s := "hello 世界"
		if got := truncateWecomMarkdown(s, 100); got != s {
			t.Errorf("got %q, want unchanged", got)
		}
	})

	t.Run("long string truncated by bytes without breaking runes", func(t *testing.T) {
		s := strings.Repeat("中", 100) // 300 bytes
		got := truncateWecomMarkdown(s, 100)

		if len(got) > 100 {
			t.Errorf("result length %d exceeds limit 100", len(got))
		}
		if !strings.HasSuffix(got, "...") {
			t.Errorf("result should end with ellipsis, got %q", got)
		}
		if !strings.HasPrefix(s, strings.TrimSuffix(got, "...")) {
			t.Errorf("result should be a prefix of the source")
		}
	})
}
