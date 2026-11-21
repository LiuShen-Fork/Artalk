package handler

import (
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/entity"
)

// preferredEmailAccount chooses which row a code-based login or password reset
// should act on when several accounts share one email.
//
// Anonymous commenting creates a user row per (name, email), so one email can
// own a guest profile and a registered account under different names. A code
// proves ownership of the email, not of a particular row; landing on the
// account that has a password or a linked identity is what the person means to
// reach, and keeps that credential alive if they merge the accounts afterwards.
// Without any credentialed row the oldest row keeps the previous behavior.
func preferredEmailAccount(app *core.App, users []entity.User) entity.User {
	for _, user := range users {
		if userHasLoginCredentials(app, user) {
			return user
		}
	}
	if len(users) == 0 {
		return entity.User{}
	}
	return users[0]
}
