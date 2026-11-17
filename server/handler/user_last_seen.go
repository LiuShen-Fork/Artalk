package handler

import (
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/gofiber/fiber/v2"
)

// updateUserLastSeen stores the request IP / UA on the user record.
// It reports whether anything changed, so callers can skip a needless write.
func updateUserLastSeen(user *entity.User, ip string, ua string) bool {
	if user == nil || user.IsEmpty() {
		return false
	}
	if user.LastIP == ip && user.LastUA == ua {
		return false
	}
	user.LastIP = ip
	user.LastUA = ua
	return true
}

// touchUserLastSeen records where the current request came from.
// Used on login and on comment creation so the admin panel can show the
// most recent IP / UA of every user, not only of anonymous commenters.
func touchUserLastSeen(app *core.App, c *fiber.Ctx, user *entity.User) {
	if !updateUserLastSeen(user, clientIP(c), c.Get("User-Agent")) {
		return
	}
	if err := app.Dao().UpdateUserLastSeen(user); err != nil {
		log.Error("[TouchUserLastSeen] ", err)
	}
}
