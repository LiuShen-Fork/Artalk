package core

import (
	"strconv"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/entity"
	"github.com/artalkjs/artalk/v2/internal/log"
	"github.com/artalkjs/artalk/v2/internal/utils"
)

func (app *App) syncFromConf() {
	// Initialize default site
	siteDefault := app.Dao().FindCreateSite(app.Conf().SiteDefault, app.Conf().SiteURL)
	if app.Conf().SiteURL != "" && siteDefault.Urls != app.Conf().SiteURL {
		siteDefault.Urls = app.Conf().SiteURL
		app.Dao().UpdateSite(&siteDefault)
		log.Info("Default Site ", strconv.Quote(app.Conf().SiteDefault),
			" URL has been updated to: ", strconv.Quote(app.Conf().SiteURL))
	}

	// 导入配置文件的管理员用户
	for _, admin := range app.Conf().AdminUsers {
		user := app.Dao().FindUser(admin.Name, admin.Email)
		receiveEmail := true // 默认允许接收邮件
		if admin.ReceiveEmail != nil {
			receiveEmail = *admin.ReceiveEmail
		}

		// A link written here bypasses every handler that validates one, so it
		// must be normalized before it reaches the database. Otherwise an admin
		// who wrote a bare host gets a link the update endpoints refuse, and
		// approving one of their own comments fails with a 400.
		adminLink := strings.TrimSpace(admin.Link)
		link := utils.NormalizeURL(adminLink)
		if link != adminLink {
			log.Warn("Admin ", strconv.Quote(admin.Name), " has a link that is not a valid URL: ",
				strconv.Quote(adminLink), ", using ", strconv.Quote(link), " instead")
		}

		if user.IsEmpty() {
			// create
			user = entity.User{
				Name:         admin.Name,
				Email:        admin.Email,
				Link:         link,
				Password:     admin.Password,
				BadgeName:    admin.BadgeName,
				BadgeColor:   admin.BadgeColor,
				IsAdmin:      true,
				IsInConf:     true,
				ReceiveEmail: receiveEmail,
			}
			app.dao.CreateUser(&user)
		} else {
			// update
			user.Name = admin.Name
			user.Email = admin.Email
			user.Link = link
			user.Password = admin.Password
			user.BadgeName = admin.BadgeName
			user.BadgeColor = admin.BadgeColor
			user.IsAdmin = true
			user.IsInConf = true
			user.ReceiveEmail = receiveEmail
			app.dao.UpdateUser(&user)
		}
	}

	// 清理配置文件中不存在的用户
	// var dbAdminUsers []User
	// lib.DB.Model(&User{}).Where(&User{IsInConf: true}).Find(&dbAdminUsers)
	// for _, dbU := range dbAdminUsers {
	// 	isUserExist := func() bool {
	// 		for _, confU := range app.Conf().AdminUsers {
	// 			// 忽略大小写比较
	// 			if strings.EqualFold(confU.Name, dbU.Name) && strings.EqualFold(confU.Email, dbU.Email) {
	// 				return true
	// 			}
	// 		}
	// 		return false
	// 	}

	// 	if !isUserExist() {
	// 		DelUser(&dbU)
	// 	}
	// }
}
