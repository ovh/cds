package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ovh/tat"
	"github.com/ovh/tat/api/cache"
	"github.com/ovh/tat/api/hook"
	"github.com/spf13/viper"
)

// SystemController contains all methods about version
type SystemController struct{}

//GetVersion returns version of tat
func (*SystemController) GetVersion(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"version": tat.Version})
}

//GetCapabilites returns version of tat
func (*SystemController) GetCapabilites(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, tat.Capabilities{
		UsernameFromEmail: viper.GetBool("username_from_email"),
		Hooks:             hook.GetCapabilities(),
	})
}

// CleanCache cleans cache...
func (*SystemController) CleanCache(ctx *gin.Context) {
	out, err := cache.FlushDB()
	ctx.JSON(http.StatusOK, gin.H{
		"output": out,
		"error":  err,
	})
}

// CleanInfo returns INFO cmd on redis
func (*SystemController) CleanInfo(ctx *gin.Context) {
	out, err := cache.Info()
	ctx.JSON(http.StatusOK, gin.H{
		"output": out,
		"error":  err,
	})
}
