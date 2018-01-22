package main

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat"
)

var (

	// tatHeaderUsernameLower is a Header in lowercase
	tatHeaderUsernameLower = strings.ToLower(tat.TatHeaderUsername)

	// TatHeaderXTatRefererLower contains tat microservice name & version "X-TAT-FROM"
	TatHeaderXTatRefererLower = strings.ToLower("X-Tat-Referer")

	// tatHeaderUsernameLowerDash is a Header in lowercase, and dash : tat-username
	tatHeaderUsernameLowerDash = strings.Replace(tatHeaderUsernameLower, "_", "-", -1)

	// tatCtxIsAdmin is used in Gin Context True if user is admin
	tatCtxIsAdmin = "Tat_isAdmin"

	// TatHeaderPassword contains tat password
	tatHeaderPasswordLower     = "tat_password"
	tatHeaderPasswordLowerDash = "tat-password"
)

// isTatAdmin return true if user is admin. Get value in gin.Context
func isTatAdmin(ctx *gin.Context) bool {
	value, exist := ctx.Get(tatCtxIsAdmin)
	if value != nil && exist && value.(bool) == true {
		return true
	}
	return false
}

// getCtxUsername return username, getting value in gin.Context
func getCtxUsername(ctx *gin.Context) string {
	username, exist := ctx.Get(tat.TatHeaderUsername)
	if username == nil || !exist {
		log.Debug("Username is null in context !")
		return ""
	}
	return username.(string)
}
