package core

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/anti_spam"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/log"
)

var _ Service = (*AntiSpamService)(nil)

type AntiSpamService struct {
	app    *App
	client *anti_spam.AntiSpam
}

func NewAntiSpamService(app *App) *AntiSpamService {
	return &AntiSpamService{app: app}
}

func (s *AntiSpamService) Init() error {
	s.pruneModerationLogs()

	s.client = anti_spam.NewAntiSpam(&anti_spam.AntiSpamConf{
		ModeratorConf: s.app.Conf().Moderator,
		OnBlockComment: func(commentID uint) {
			comment := s.app.dao.FindComment(commentID)
			if comment.IsPending {
				return // no need to block again
			}

			// update comment status
			comment.IsPending = true
			s.app.dao.UpdateComment(&comment)
		},
		OnUpdateComment: func(commentID uint, content string) {
			comment := s.app.dao.FindComment(commentID)
			comment.Content = content
			s.app.dao.UpdateComment(&comment)
		},
		OnCheckResult: func(result anti_spam.CheckResult) {
			s.recordCheckResult(result)
		},
	})

	return nil
}

func (s *AntiSpamService) Dispose() error {
	s.client = nil

	return nil
}

func (s *AntiSpamService) recordCheckResult(result anti_spam.CheckResult) {
	s.pruneModerationLogs()

	if !shouldRecordModerationResult(result) {
		return
	}

	logRow := entity.ModerationLog{
		CommentID:      result.CommentID,
		SiteName:       result.SiteName,
		PageKey:        result.PageKey,
		UserID:         result.UserID,
		UserName:       result.UserName,
		UserEmail:      result.UserEmail,
		CommentContent: result.CommentContent,
		Checker:        result.Checker,
		Status:         string(result.Status),
		Action:         string(result.Action),
		Message:        result.Message,
	}
	if err := s.app.dao.DB().Create(&logRow).Error; err != nil {
		log.Errorf("[AntiSpam] record moderation log failed: %v", err)
	}
}

func shouldRecordModerationResult(result anti_spam.CheckResult) bool {
	return result.Status == anti_spam.CheckStatusBlock ||
		result.Status == anti_spam.CheckStatusError ||
		result.Action == anti_spam.CheckActionReplace
}

func (s *AntiSpamService) pruneModerationLogs() {
	cutoff := time.Now().AddDate(0, 0, -90)
	if err := s.app.dao.DB().Where("created_at < ?", cutoff).Delete(&entity.ModerationLog{}).Error; err != nil {
		log.Errorf("[AntiSpam] prune moderation logs failed: %v", err)
	}
}

func (s *AntiSpamService) CheckAndBlock(data *AntiSpamCheckPayload) {
	s.client.CheckAndBlock(s.payload2CheckerParams(data))
}

// CheckIntercept synchronously checks a comment before it is persisted.
func (s *AntiSpamService) CheckIntercept(data *AntiSpamCheckPayload) (bool, string) {
	if s.client == nil {
		return false, ""
	}
	return s.client.CheckIntercept(s.payload2CheckerParams(data))
}

// Payload for CheckAndBlock function
type AntiSpamCheckPayload struct {
	Comment      *entity.Comment
	ReqReferer   string
	ReqIP        string
	ReqUserAgent string
}

// Transform `AntiSpamCheckPayload` to `CheckerParams` for `anti_spam.CheckAndBlock` func call
//
//	The `AntiSpamCheckPayload` struct is exposed and can be used by other modules
//	The `CheckerParams` struct is used by `anti_spam.CheckAndBlock` in anti_spam module
func (s *AntiSpamService) payload2CheckerParams(payload *AntiSpamCheckPayload) *anti_spam.CheckerParams {
	user := s.app.dao.FetchUserForComment(payload.Comment)
	siteURL := ""

	if payload.Comment.SiteName != "" {
		site := s.app.dao.FindSite(payload.Comment.SiteName)
		siteURL = s.app.dao.CookSite(&site).FirstUrl
	}
	if siteURL == "" {
		// extract site url from referer
		if pr, err := url.Parse(payload.ReqReferer); err == nil && pr.Scheme != "" && pr.Host != "" {
			siteURL = fmt.Sprintf("%s://%s", pr.Scheme, pr.Host)
		}
	}

	reviewContent, err := anti_spam.NormalizeReviewContent(payload.Comment.Content)
	if err != nil {
		log.Errorf("[AntiSpam] normalize comment=%d review text failed: %v", payload.Comment.ID, err)
		reviewContent = strings.Join(strings.Fields(payload.Comment.Content), " ")
	}

	return &anti_spam.CheckerParams{
		BlogURL: siteURL,

		CommentID:     payload.Comment.ID,
		SiteName:      payload.Comment.SiteName,
		PageKey:       payload.Comment.PageKey,
		RawContent:    payload.Comment.Content,
		ReviewContent: reviewContent,
		ReviewText:    anti_spam.BuildReviewText(user.Name, reviewContent),

		UserName:  user.Name,
		UserEmail: user.Email,
		UserID:    user.ID,
		UserIP:    payload.ReqIP,
		UserAgent: payload.ReqUserAgent,
	}
}
