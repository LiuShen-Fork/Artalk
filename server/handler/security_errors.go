package handler

import "errors"

var (
	errInvalidSecurityAction = errors.New("invalid security action, must be captcha / reject / ban")
	errInvalidAutoBan        = errors.New("invalid auto ban config, threshold / window / duration must not be negative")
	errInvalidRetention      = errors.New("invalid event retention days")
	errInvalidCaptchaType    = errors.New("invalid captcha type")
	errInvalidCaptchaLimit   = errors.New("invalid captcha action limit or reset")
	errInvalidExcludePath    = errors.New("invalid exclude path, must start with /")
	errInvalidIPTarget       = errors.New("invalid ip or cidr target")
)
