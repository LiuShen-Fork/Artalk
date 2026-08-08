package anti_spam

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/samber/lo"
)

const LOG_TAG = "[AntiSpam] "

// -------------------------------------------------------------------
//  AntiSpam
// -------------------------------------------------------------------

type AntiSpamConf struct {
	config.ModeratorConf

	OnBlockComment  func(commentID uint)
	OnUpdateComment func(commentID uint, content string)
	OnCheckResult   func(result CheckResult)
}

type AntiSpam struct {
	conf *AntiSpamConf
}

// Create new AntiSpam instance
func NewAntiSpam(conf *AntiSpamConf) *AntiSpam {
	return &AntiSpam{
		conf: conf,
	}
}

// Check and block comment if it is spam,
// the function is exposed and can be called by other modules
func (as AntiSpam) CheckAndBlock(params *CheckerParams) {
	as.checkAndBlockWithCheckers(params, as.getEnabledCheckers())
}

func (as AntiSpam) checkAndBlockWithCheckers(params *CheckerParams, checkers []Checker) {
	shouldBlock := false

	// Execute check one by one
	// Multiple checkers can be enabled at the same time
	// All enabled checkers are executed so every stage can be recorded.
	for _, checker := range checkers {
		pass := as.checkerTrigger(checker, params)
		if !pass {
			shouldBlock = true
		}
	}

	if !shouldBlock {
		return
	}

	if as.conf.OnBlockComment != nil {
		as.conf.OnBlockComment(params.CommentID)
	}

	log.Debug(LOG_TAG, fmt.Sprintf("Successful blocking of comments ID=%d CONT=%s",
		params.CommentID, strconv.Quote(params.RawContent)))
}

// Checker trigger function
func (as AntiSpam) checkerTrigger(checker Checker, params *CheckerParams) bool {
	params.UpdatedContent = ""
	params.ResultReason = ""

	pass, err := checker.Check(params)
	status := CheckStatusPass
	action := CheckActionAllow
	message := params.ResultReason

	if err != nil {
		log.Error(LOG_TAG, fmt.Sprintf("%s checker comment=%d error:",
			checker.Name(), params.CommentID), err)

		pass = lo.If(as.conf.ApiFailBlock, false).Else(true) // block if api fail
		status = CheckStatusError
		message = err.Error()
		if !pass {
			action = CheckActionPending
		}
	} else if !pass {
		status = CheckStatusBlock
		action = CheckActionPending
	} else if params.UpdatedContent != "" {
		action = CheckActionReplace
		if message == "" {
			message = "keyword matched and comment content was replaced"
		}
	}

	if as.conf.OnCheckResult != nil {
		as.conf.OnCheckResult(CheckResult{
			CommentID: params.CommentID,
			SiteName:  params.SiteName,
			PageKey:   params.PageKey,
			UserID:    params.UserID,
			Checker:   checker.Name(),
			Status:    status,
			Action:    action,
			Message:   message,
		})
	}

	return pass
}

// Get enabled checkers by config
func (as AntiSpam) getEnabledCheckers() []Checker {
	checkers := []Checker{}

	// Akismet
	akismetKey := strings.TrimSpace(as.conf.AkismetKey)
	if akismetKey != "" {
		checkers = append(checkers, NewAkismetChecker(akismetKey))
	}

	// Tencent Cloud
	tencentConf := as.conf.Tencent
	if tencentConf.Enabled {
		checkers = append(checkers, NewTencentChecker(
			tencentConf.SecretID, tencentConf.SecretKey, tencentConf.Region))
	}

	// Aliyun
	aliyunConf := as.conf.Aliyun
	if aliyunConf.Enabled {
		checkers = append(checkers, NewAliyunChecker(
			aliyunConf.AccessKeyID, aliyunConf.AccessKeySecret, aliyunConf.Region))
	}

	// AI
	aiConf := as.conf.AI
	if aiConf.Enabled {
		checkers = append(checkers, NewAIChecker(AICheckerConf{
			APIType: AIAPIType(aiConf.APIType),
			BaseURL: aiConf.BaseURL,
			APIKey:  aiConf.APIKey,
			Model:   aiConf.Model,
			Prompt:  aiConf.Prompt,
		}))
	}
	// Keywords Checker
	keywordsConf := as.conf.Keywords
	if keywordsConf.Enabled {

		var kwCheckerMode KwCheckerMode
		if as.conf.Keywords.Pending {
			kwCheckerMode = KwCheckerModeBlock
		} else {
			kwCheckerMode = KwCheckerModeReplace
		}

		checkers = append(checkers, NewKeywordsChecker(&KeywordsCheckerConf{
			Files:     as.conf.Keywords.Files,
			FileSep:   as.conf.Keywords.FileSep,
			ReplaceTo: as.conf.Keywords.ReplaceTo,
			Mode:      kwCheckerMode,
			OnUpdateComment: func(commentID uint, content string) {
				if as.conf.OnUpdateComment != nil {
					as.conf.OnUpdateComment(commentID, content)
				}
			},
		}))

	}

	return checkers
}

// -------------------------------------------------------------------
//  Checker
// -------------------------------------------------------------------

type CheckerParams struct {
	BlogURL string

	CommentID      uint
	SiteName       string
	PageKey        string
	RawContent     string
	ReviewContent  string
	ReviewText     string
	UpdatedContent string
	ResultReason   string

	UserName  string
	UserEmail string
	UserID    uint
	UserIP    string
	UserAgent string
}

type Checker interface {
	Name() string
	Check(p *CheckerParams) (bool, error)
}

type CheckStatus string

const (
	CheckStatusPass  CheckStatus = "pass"
	CheckStatusBlock CheckStatus = "block"
	CheckStatusError CheckStatus = "error"
)

type CheckAction string

const (
	CheckActionAllow   CheckAction = "allow"
	CheckActionPending CheckAction = "pending"
	CheckActionReplace CheckAction = "replace"
)

type CheckResult struct {
	CommentID uint
	SiteName  string
	PageKey   string
	UserID    uint
	Checker   string
	Status    CheckStatus
	Action    CheckAction
	Message   string
}
