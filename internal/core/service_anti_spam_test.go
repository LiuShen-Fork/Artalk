package core

import (
	"testing"

	"github.com/artalkjs/artalk/v2/internal/anti_spam"
	"github.com/stretchr/testify/assert"
)

func TestShouldRecordModerationResult(t *testing.T) {
	tests := []struct {
		name   string
		result anti_spam.CheckResult
		want   bool
	}{
		{name: "normal pass", result: anti_spam.CheckResult{Status: anti_spam.CheckStatusPass, Action: anti_spam.CheckActionAllow}, want: false},
		{name: "blocked", result: anti_spam.CheckResult{Status: anti_spam.CheckStatusBlock, Action: anti_spam.CheckActionPending}, want: true},
		{name: "checker error", result: anti_spam.CheckResult{Status: anti_spam.CheckStatusError, Action: anti_spam.CheckActionPending}, want: true},
		{name: "replaced keyword", result: anti_spam.CheckResult{Status: anti_spam.CheckStatusPass, Action: anti_spam.CheckActionReplace}, want: true},
		{name: "synchronous rejection", result: anti_spam.CheckResult{Status: anti_spam.CheckStatusBlock, Action: anti_spam.CheckActionReject}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldRecordModerationResult(tt.result))
		})
	}
}
