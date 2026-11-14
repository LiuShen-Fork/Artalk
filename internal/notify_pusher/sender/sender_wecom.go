package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/artalkjs/artalk/v2/internal/i18n"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/artalkjs/artalk/v2/internal/utils"
)

// 企业微信群机器人 markdown 内容的最大字节数
// @see https://developer.work.weixin.qq.com/document/path/91770
const WecomMarkdownContentLimit = 4096

// 企业微信群机器人请求体
type wecomReqBody struct {
	MsgType  string `json:"msgtype"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

// 企业微信群机器人响应体
type wecomRespBody struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

var (
	// 表情包图片标签 (含 atk-emoticon 属性)
	wecomEmoticonImgRe = regexp.MustCompile(`<img\s[^>]*?atk-emoticon=["]([^"]*?)["][^>]*?>`)
	// 其余图片标签
	wecomImgTagRe = regexp.MustCompile(`<img\s[^>]*?>`)
	// Markdown 图片语法
	wecomImgMdRe = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
)

// 企业微信发送 (群机器人 WebHook)
func SendWecom(webhookURL string, content string) {
	if strings.TrimSpace(webhookURL) == "" {
		return
	}

	var reqBody wecomReqBody
	reqBody.MsgType = "markdown"
	reqBody.Markdown.Content = truncateWecomMarkdown(content, WecomMarkdownContentLimit)

	jsonByte, err := json.Marshal(reqBody)
	if err != nil {
		log.Error("[企业微信] Failed to marshal msg:", err)
		return
	}

	result, err := http.Post(webhookURL, "application/json", bytes.NewReader(jsonByte))
	if err != nil {
		log.Error("[企业微信] Failed to send msg:", err)
		return
	}
	defer result.Body.Close()

	body, _ := io.ReadAll(result.Body)

	if result.StatusCode != 200 {
		log.Error("[企业微信] Failed to send msg:", result.StatusCode, string(body))
		return
	}

	// 企业微信即使出错也返回 HTTP 200，需要检查响应体中的 errcode
	var respBody wecomRespBody
	if err := json.Unmarshal(body, &respBody); err != nil {
		log.Error("[企业微信] Failed to parse response:", err, string(body))
		return
	}

	if respBody.ErrCode != 0 {
		log.Error("[企业微信] Failed to send msg:", respBody.ErrCode, respBody.ErrMsg, string(jsonByte))
	}
}

// 生成企业微信群机器人 markdown 消息内容
//
// 格式：
//
//	来自 {{nick}} 的新评论
//	> 文章： [标题](链接)
//	> 评论内容
//
//	[查看详情](链接)
func BuildWecomMarkdown(nick, pageTitle, pageURL, content, detailsURL string) string {
	var sb strings.Builder
	sb.WriteString(i18n.T("New comment from {{nick}}", map[string]interface{}{"nick": nick}))
	sb.WriteString("\n")

	// 文章标题 (带超链接)
	title := strings.TrimSpace(pageTitle)
	pageURL = strings.TrimSpace(pageURL)
	if title == "" {
		title = pageURL
	}
	if title != "" {
		title = escapeWecomLinkText(title)
		if utils.ValidateURL(pageURL) {
			sb.WriteString(fmt.Sprintf("> %s： [%s](%s)\n", i18n.T("Post"), title, pageURL))
		} else {
			sb.WriteString(fmt.Sprintf("> %s： %s\n", i18n.T("Post"), title))
		}
	}

	// 评论内容 (每行都加上引用前缀，保持引用块连续)
	for _, line := range strings.Split(CommentContentToText(content), "\n") {
		sb.WriteString("> ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// 查看详情
	if detailsURL = strings.TrimSpace(detailsURL); detailsURL != "" {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("[%s](%s)", i18n.T("View details"), detailsURL))
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

// 将评论内容转换为企业微信 markdown 文本
//
// 评论内容本身就是 Markdown，因此仅做两处替换，其余语法原样保留：
//   - 表情包 (含 atk-emoticon 属性的 img) 提取主干主题，例如 `liushen-迷惑` -> `[迷惑]`
//   - 其他图片 (img 标签或 Markdown 图片语法) 转换为 `[图片]`
func CommentContentToText(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	// 表情包 (须先于普通图片处理)
	content = wecomEmoticonImgRe.ReplaceAllStringFunc(content, func(m string) string {
		ms := wecomEmoticonImgRe.FindStringSubmatch(m)
		if len(ms) < 2 {
			return "[" + i18n.T("Emoticon") + "]"
		}
		return "[" + emoticonMainTheme(ms[1]) + "]"
	})

	// 其他 HTML 图片标签
	content = wecomImgTagRe.ReplaceAllString(content, "["+i18n.T("Image")+"]")

	// Markdown 图片语法
	content = wecomImgMdRe.ReplaceAllString(content, "["+i18n.T("Image")+"]")

	return normalizeWecomText(content)
}

// 提取表情包主干主题
//
// 表情包 key 形如 `liushen-迷惑`，取最后一个 `-` 之后的部分作为主干主题
func emoticonMainTheme(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return i18n.T("Emoticon")
	}

	if idx := strings.LastIndex(key, "-"); idx != -1 {
		if theme := strings.TrimSpace(key[idx+1:]); theme != "" {
			return theme
		}
	}

	return key
}

// 规范化文本：统一换行符、去除行尾空格、丢弃空白行
//
// 通知消息追求紧凑，空白行在引用块中会形成空洞，因此直接丢弃
func normalizeWecomText(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			continue // 丢弃空白行，避免引用块出现空洞
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// 转义企业微信 markdown 链接文案
//
// 企业微信 markdown 不支持转义方括号，替换为全角符号避免链接语法被破坏
func escapeWecomLinkText(text string) string {
	text = strings.ReplaceAll(text, "[", "【")
	text = strings.ReplaceAll(text, "]", "】")
	return text
}

// 按字节长度安全截断 (不破坏 UTF-8 字符)
func truncateWecomMarkdown(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}

	const ellipsis = "..."
	cut := limit - len(ellipsis)

	var sb strings.Builder
	for _, r := range s {
		if sb.Len()+utf8.RuneLen(r) > cut {
			break
		}
		sb.WriteRune(r)
	}

	return sb.String() + ellipsis
}
