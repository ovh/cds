package main

import (
	"github.com/gin-gonic/gin"
)

// initRoutesGroups initialized routes for Groups Controller
func initRoutesGroups(router *gin.RouterGroup, checkPassword gin.HandlerFunc) {
	groupsCtrl := &GroupsController{}

	g := router.Group("/")
	g.Use(checkPassword)
	{
		g.GET("/groups", groupsCtrl.List)

		g.PUT("/group/add/user", groupsCtrl.AddUser)
		g.PUT("/group/remove/user", groupsCtrl.RemoveUser)
		g.PUT("/group/add/adminuser", groupsCtrl.AddAdminUser)
		g.PUT("/group/remove/adminuser", groupsCtrl.RemoveAdminUser)

		admin := router.Group("/group")
		admin.Use(checkPassword, CheckAdmin())
		{
			admin.POST("", groupsCtrl.Create)
			admin.DELETE("edit/:group", groupsCtrl.Delete)
			admin.PUT("edit/:group", groupsCtrl.Update)
		}
	}
}

// initRoutesMessages initialized routes for Messages Controller
func initRoutesMessages(router *gin.RouterGroup, checkPassword gin.HandlerFunc) {
	messagesCtrl := &MessagesController{}

	g := router.Group("/messages")
	g.Use(checkPassword)
	{
		g.POST("/*topic", messagesCtrl.CreateBulk)
		g.GET("/*topic", messagesCtrl.List)
		g.DELETE("/nocascade/*topic", messagesCtrl.DeleteBulk)
		g.DELETE("/cascade/*topic", messagesCtrl.DeleteBulkCascade)
		g.DELETE("/cascadeforce/*topic", messagesCtrl.DeleteBulkCascadeForce)
	}

	r := router.Group("/read")
	r.Use()
	{
		r.GET("/*topic", messagesCtrl.List)
	}

	gm := router.Group("/message")
	gm.Use(checkPassword)
	{
		//Create a message, a reply
		gm.POST("/*topic", messagesCtrl.Create)

		// Like, Unlike, Label, Unlabel a message, mark as task, voteup, votedown, unvoteup, unvotedown
		gm.PUT("/*topic", messagesCtrl.Update)

		// Delete a message
		gm.DELETE("/nocascade/:idMessage/*topic", messagesCtrl.Delete)

		// Delete a message and its replies
		gm.DELETE("/cascade/:idMessage/*topic", messagesCtrl.DeleteCascade)

		// Delete a message and its replies, event if it's in a Tasks Topic of one user
		gm.DELETE("/cascadeforce/:idMessage/*topic", messagesCtrl.DeleteCascadeForce)
	}
}

// initRoutesPresences initialized routes for Presences Controller
func initRoutesPresences(router *gin.RouterGroup, checkPassword gin.HandlerFunc) {
	presencesCtrl := &PresencesController{}
	g := router.Group("/")
	g.Use(checkPassword)
	{
		// List Presences
		g.GET("presences", presencesCtrl.List)
		g.GET("presences/*topic", presencesCtrl.List)
		// Add a presence and get list
		g.POST("presenceget/*topic", presencesCtrl.CreateAndGet)
		// delete a presence
		g.DELETE("presences/*topic", presencesCtrl.Delete)
	}
	admin := router.Group("/presencesadmin")
	admin.Use(checkPassword, CheckAdmin())
	{
		admin.GET("/checkall", presencesCtrl.CheckAllPresences)
	}
}

// initRoutesStats initialized routes for Stats Controller
func initRoutesStats(router *gin.RouterGroup, checkPassword gin.HandlerFunc) {
	statsCtrl := &StatsController{}

	admin := router.Group("/stats")
	admin.Use(checkPassword, CheckAdmin())
	{
		admin.GET("/count", statsCtrl.Count)
		admin.GET("/instance", statsCtrl.Instance)
		admin.GET("/distribution/topics", statsCtrl.DistributionTopics)
		admin.GET("/db/stats", statsCtrl.DBStats)
		admin.GET("/db/replSetGetConfig", statsCtrl.DBReplSetGetConfig)
		admin.GET("/db/serverStatus", statsCtrl.DBServerStatus)
		admin.GET("/db/replSetGetStatus", statsCtrl.DBReplSetGetStatus)
		admin.GET("/db/collections", statsCtrl.DBStatsCollections)
		admin.GET("/db/slowestQueries", statsCtrl.DBGetSlowestQueries)
		admin.GET("/checkHeaders", statsCtrl.CheckHeaders)
	}
}

// initRoutesSystem initialized routes for System Controller
func initRoutesSystem(router *gin.RouterGroup, checkPassword gin.HandlerFunc) {
	systemCtrl := &SystemController{}
	router.GET("/version", systemCtrl.GetVersion)
	router.GET("/capabilities", systemCtrl.GetCapabilites)
	admin := router.Group("/system")
	admin.Use(checkPassword, CheckAdmin())
	{
		admin.GET("/cache/clean", systemCtrl.CleanCache)
		admin.GET("/cache/info", systemCtrl.CleanInfo)
	}
}

// initRoutesTopics initialized routes for Topics Controller
func initRoutesTopics(router *gin.RouterGroup, checkPassword gin.HandlerFunc) {
	topicsCtrl := &TopicsController{}

	g := router.Group("/")
	g.Use(checkPassword)
	{
		g.GET("/topics", topicsCtrl.List)
		g.POST("/topic", topicsCtrl.Create)
		g.DELETE("/topic/*topic", topicsCtrl.Delete)
		g.GET("/topic/*topic", topicsCtrl.OneTopic)

		g.PUT("/topic/add/parameter", topicsCtrl.AddParameter)
		g.PUT("/topic/remove/parameter", topicsCtrl.RemoveParameter)

		g.PUT("/topic/add/filter", topicsCtrl.AddFilter)
		g.PUT("/topic/remove/filter", topicsCtrl.RemoveFilter)
		g.PUT("/topic/update/filter", topicsCtrl.UpdateFilter)

		g.PUT("/topic/add/rouser", topicsCtrl.AddRoUser)
		g.PUT("/topic/remove/rouser", topicsCtrl.RemoveRoUser)
		g.PUT("/topic/add/rwuser", topicsCtrl.AddRwUser)
		g.PUT("/topic/remove/rwuser", topicsCtrl.RemoveRwUser)
		g.PUT("/topic/add/adminuser", topicsCtrl.AddAdminUser)
		g.PUT("/topic/remove/adminuser", topicsCtrl.RemoveAdminUser)

		g.PUT("/topic/compute/tags", topicsCtrl.ComputeTags)
		g.PUT("/topic/truncate/tags", topicsCtrl.TruncateTags)
		g.PUT("/topic/compute/labels", topicsCtrl.ComputeLabels)
		g.PUT("/topic/truncate/labels", topicsCtrl.TruncateLabels)
		g.PUT("/topic/truncate", topicsCtrl.Truncate)
		g.PUT("/topic/add/rogroup", topicsCtrl.AddRoGroup)
		g.PUT("/topic/remove/rogroup", topicsCtrl.RemoveRoGroup)
		g.PUT("/topic/add/rwgroup", topicsCtrl.AddRwGroup)
		g.PUT("/topic/remove/rwgroup", topicsCtrl.RemoveRwGroup)
		g.PUT("/topic/add/admingroup", topicsCtrl.AddAdminGroup)
		g.PUT("/topic/remove/admingroup", topicsCtrl.RemoveAdminGroup)
		g.PUT("/topic/param", topicsCtrl.SetParam)
	}

	admin := router.Group("/topics")
	admin.Use(checkPassword, CheckAdmin())
	{
		admin.PUT("/compute/tags", topicsCtrl.AllComputeTags)
		admin.PUT("/compute/labels", topicsCtrl.AllComputeLabels)
		admin.PUT("/compute/replies", topicsCtrl.AllComputeReplies)
		admin.PUT("/migrate/dedicated/*topic", topicsCtrl.MigrateToDedicatedTopic)
		admin.PUT("/migrate/dedicatedmessages/:limit/*topic", topicsCtrl.MigrateMessagesForDedicatedTopic)
		admin.PUT("/param", topicsCtrl.AllSetParam)
	}
}

// initRoutesUsers initialized routes for Users Controller
func initRoutesUsers(router *gin.RouterGroup, checkPassword gin.HandlerFunc) {
	usersCtrl := &UsersController{}

	gs := router.Group("/users")
	gs.Use(checkPassword)
	{
		gs.GET("", usersCtrl.List)
	}
	g := router.Group("/user")
	g.Use(checkPassword)
	{
		g.GET("/me", usersCtrl.Me)
		g.GET("/me/contacts/:sinceSeconds", usersCtrl.Contacts)
		g.POST("/me/contacts/:username", usersCtrl.AddContact)
		g.DELETE("/me/contacts/:username", usersCtrl.RemoveContact)
		g.POST("/me/topics/*topic", usersCtrl.AddFavoriteTopic)
		g.DELETE("/me/topics/*topic", usersCtrl.RemoveFavoriteTopic)
		g.POST("/me/tags/:tag", usersCtrl.AddFavoriteTag)
		g.DELETE("/me/tags/:tag", usersCtrl.RemoveFavoriteTag)

		g.POST("/me/enable/notifications/topics/*topic", usersCtrl.EnableNotificationsTopic)
		g.POST("/me/disable/notifications/topics/*topic", usersCtrl.DisableNotificationsTopic)

		g.POST("/me/enable/notifications/alltopics", usersCtrl.EnableNotificationsAllTopics)
		g.POST("/me/disable/notifications/alltopics", usersCtrl.DisableNotificationsAllTopics)
	}

	admin := router.Group("/user")
	admin.Use(checkPassword, CheckAdmin())
	{
		admin.PUT("/convert", usersCtrl.Convert)
		admin.PUT("/archive", usersCtrl.Archive)
		admin.PUT("/rename", usersCtrl.Rename)
		admin.PUT("/update", usersCtrl.Update)
		admin.PUT("/setadmin", usersCtrl.SetAdmin)
		admin.PUT("/resetsystem", usersCtrl.ResetSystemUser)
		admin.PUT("/updatesystem", usersCtrl.UpdateSystemUser)
		admin.PUT("/check", usersCtrl.Check)
	}

	router.GET("/user/verify/:username/:tokenVerify", usersCtrl.Verify)
	router.POST("/user/reset", usersCtrl.Reset)
	router.POST("/user", usersCtrl.Create)
}
