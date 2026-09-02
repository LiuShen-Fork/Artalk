package anti_spam

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommentInterceptor(t *testing.T) {
	interceptor := NewCommentInterceptor(" spam,广告, ,spam ")

	blocked, keyword := interceptor.Check(&CheckerParams{
		UserName:      "Alice",
		ReviewContent: "This comment contains 广告 content",
	})
	assert.True(t, blocked)
	assert.Equal(t, "广告", keyword)

	blocked, keyword = interceptor.Check(&CheckerParams{
		UserName:      "spam-user",
		ReviewContent: "normal content",
	})
	assert.True(t, blocked)
	assert.Equal(t, "spam", keyword)

	blocked, keyword = interceptor.Check(&CheckerParams{
		UserName:      "Alice",
		ReviewContent: "normal content",
	})
	assert.False(t, blocked)
	assert.Empty(t, keyword)

	assert.Equal(t, []string{"spam", "广告", "spam"}, interceptor.Keywords())
}

func TestNewKeywordsChecker(t *testing.T) {
	kwFile1 := fmt.Sprintf("%s/keywords_1.txt", t.TempDir())
	_ = os.WriteFile(kwFile1, []byte("关键词A\n关键词B"), 0644)
	defer os.Remove(kwFile1)

	kwFile2 := fmt.Sprintf("%s/keywords_2.txt", t.TempDir())
	_ = os.WriteFile(kwFile2, []byte("关键词C\n关键词D"), 0644)
	defer os.Remove(kwFile2)

	kwFile3 := fmt.Sprintf("%s/keywords_3.txt", t.TempDir())
	_ = os.WriteFile(kwFile3, []byte("关键词E\n关键词F"), 0644)
	defer os.Remove(kwFile3)

	assert.Equal(t, "keywords", NewKeywordsChecker(&KeywordsCheckerConf{}).Name())

	t.Run("BlockMode", func(t *testing.T) {
		checker := NewKeywordsChecker(&KeywordsCheckerConf{
			Files:     []string{kwFile1, kwFile2},
			FileSep:   "\n",
			ReplaceTo: "*",
			Mode:      KwCheckerModeBlock,
		})

		t.Run("Exist", func(t *testing.T) {
			ok, err := checker.Check(&CheckerParams{
				RawContent:    "dWQDQOIJWO\nABC关键词CEF\nABDIWHDUWH\n\n",
				ReviewContent: "dWQDQOIJWO\nABC关键词CEF\nABDIWHDUWH\n\n",
				CommentID:     1000,
			})

			assert.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("NotExist", func(t *testing.T) {
			ok, err := checker.Check(&CheckerParams{
				RawContent:    "ABCDEFG\nEWFWEOI\nWIEEWOIE\nWDIQJDW",
				ReviewContent: "ABCDEFG\nEWFWEOI\nWIEEWOIE\nWDIQJDW",
				CommentID:     1000,
			})

			assert.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("ReplaceMode", func(t *testing.T) {

		checker := NewKeywordsChecker(&KeywordsCheckerConf{
			Files:     []string{kwFile3},
			FileSep:   "\n",
			ReplaceTo: "*",
			Mode:      KwCheckerModeReplace,
		})

		t.Run("Exist", func(t *testing.T) {
			updated := false
			updatedContent := ""
			checker.conf.OnUpdateComment = func(commentID uint, content string) {
				updated = true
				updatedContent = content
			}

			ok, err := checker.Check(&CheckerParams{
				RawContent:    "ABCDEF\nEWFWEOI\nWIE关键词EWOIE\nWDIQJDW",
				ReviewContent: "ABCDEF\nEWFWEOI\nWIE关键词EWOIE\nWDIQJDW",
				CommentID:     1000,
			})
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.True(t, updated)
			assert.Equal(t, "ABCDEF\nEWFWEOI\nWIE****WOIE\nWDIQJDW", updatedContent)
		})

		t.Run("NotExist", func(t *testing.T) {
			updated := false
			checker.conf.OnUpdateComment = func(commentID uint, content string) {
				updated = true
			}

			ok, err := checker.Check(&CheckerParams{
				RawContent:    "ABCDEFG\nEWFWEOI\nWIEEWOIE\nWDIQJDW",
				ReviewContent: "ABCDEFG\nEWFWEOI\nWIEEWOIE\nWDIQJDW",
				CommentID:     1000,
			})
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.False(t, updated)
		})
	})

	t.Run("ErrorLoad", func(t *testing.T) {
		checker := NewKeywordsChecker(&KeywordsCheckerConf{
			Files:   []string{"not_exist_file"},
			FileSep: "\n",
			Mode:    KwCheckerModeBlock,
		})
		ok, err := checker.Check(&CheckerParams{
			RawContent:    "ABCDEFG\nEWFWEOI\nWIEEWOIE\nWDIQJDW",
			ReviewContent: "ABCDEFG\nEWFWEOI\nWIEEWOIE\nWDIQJDW",
		})
		assert.ErrorContains(t, err, "failed to load")
		assert.False(t, ok)
	})

	t.Run("ErrorEmptySeparator", func(t *testing.T) {
		checker := NewKeywordsChecker(&KeywordsCheckerConf{
			Files:   []string{kwFile1},
			FileSep: "",
			Mode:    KwCheckerModeBlock,
		})
		ok, err := checker.Check(&CheckerParams{
			RawContent: "关键词A",
		})
		assert.ErrorContains(t, err, "separator cannot be empty")
		assert.False(t, ok)
	})

	t.Run("ErrorUnknownMode", func(t *testing.T) {
		checker := NewKeywordsChecker(&KeywordsCheckerConf{
			FileSep: "\n",
			Mode:    999,
		})
		ok, err := checker.Check(&CheckerParams{})
		assert.ErrorContains(t, err, "unknown mode")
		assert.False(t, ok)
	})
}

func TestKeywordsCheckerFieldAware(t *testing.T) {
	keywordFile := fmt.Sprintf("%s/keywords.txt", t.TempDir())
	assert.NoError(t, os.WriteFile(keywordFile, []byte("广告"), 0644))

	newChecker := func(mode KwCheckerMode, onUpdate func(uint, string)) *KeywordsChecker {
		return NewKeywordsChecker(&KeywordsCheckerConf{
			Files:           []string{keywordFile},
			FileSep:         "\n",
			ReplaceTo:       "*",
			Mode:            mode,
			OnUpdateComment: onUpdate,
		})
	}

	t.Run("nickname match always blocks without replacement", func(t *testing.T) {
		updated := false
		checker := newChecker(KwCheckerModeReplace, func(uint, string) { updated = true })

		pass, err := checker.Check(&CheckerParams{
			UserName:      "广告用户",
			RawContent:    "正常正文",
			ReviewContent: "正常正文",
		})

		assert.NoError(t, err)
		assert.False(t, pass)
		assert.False(t, updated)
	})

	t.Run("visible body match replaces contiguous raw text", func(t *testing.T) {
		updatedContent := ""
		checker := newChecker(KwCheckerModeReplace, func(_ uint, content string) {
			updatedContent = content
		})

		pass, err := checker.Check(&CheckerParams{
			RawContent:    "<b>广告</b>",
			ReviewContent: "广告",
		})

		assert.NoError(t, err)
		assert.True(t, pass)
		assert.Equal(t, "<b>**</b>", updatedContent)
	})

	t.Run("split markup blocks instead of corrupting raw content", func(t *testing.T) {
		updated := false
		checker := newChecker(KwCheckerModeReplace, func(uint, string) { updated = true })

		pass, err := checker.Check(&CheckerParams{
			RawContent:    "广<strong>告</strong>",
			ReviewContent: "广告",
		})

		assert.NoError(t, err)
		assert.False(t, pass)
		assert.False(t, updated)
	})

	t.Run("URL and emoticon metadata are ignored", func(t *testing.T) {
		inputs := []string{
			"[正常链接](https://example.com/广告)",
			`<img src="https://example.com/广告.png" atk-emoticon="广告">`,
		}

		for _, input := range inputs {
			reviewContent, err := NormalizeReviewContent(input)
			assert.NoError(t, err)
			checker := newChecker(KwCheckerModeReplace, nil)
			pass, err := checker.Check(&CheckerParams{
				RawContent:    input,
				ReviewContent: reviewContent,
			})
			assert.NoError(t, err)
			assert.True(t, pass)
		}
	})
}
