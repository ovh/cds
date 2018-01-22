package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat"
	groupDB "github.com/ovh/tat/api/group"
	messageDB "github.com/ovh/tat/api/message"
	presenceDB "github.com/ovh/tat/api/presence"
	topicDB "github.com/ovh/tat/api/topic"
	userDB "github.com/ovh/tat/api/user"
	"github.com/spf13/viper"
)

// UsersController contains all methods about users manipulation
type UsersController struct{}

func (*UsersController) buildCriteria(ctx *gin.Context) *tat.UserCriteria {
	c := tat.UserCriteria{}
	skip, e := strconv.Atoi(ctx.DefaultQuery("skip", "0"))
	if e != nil {
		skip = 0
	}
	c.Skip = skip
	limit, e2 := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	if e2 != nil {
		limit = 100
	}
	withGroups, e := strconv.ParseBool(ctx.DefaultQuery("withGroups", "false"))
	if e != nil {
		withGroups = false
	}
	c.Limit = limit
	c.WithGroups = withGroups
	c.IDUser = ctx.Query("idUser")

	c.Username = ctx.Query("username")
	c.Fullname = ctx.Query("fullname")
	c.DateMinCreation = ctx.Query("dateMinCreation")
	c.DateMaxCreation = ctx.Query("dateMaxCreation")
	if c.SortBy == "" {
		c.SortBy = "-dateCreation"
	}
	return &c
}

// List list all users matching Criteria
func (u *UsersController) List(ctx *gin.Context) {
	criteria := u.buildCriteria(ctx)

	var listAsAdmin bool
	if isTatAdmin(ctx) {
		listAsAdmin = true
	} else {
		user, e := PreCheckUser(ctx)
		if e != nil {
			ctx.AbortWithError(http.StatusInternalServerError, e)
			return
		}
		listAsAdmin = user.CanListUsersAsAdmin
	}
	count, users, err := userDB.ListUsers(criteria, listAsAdmin)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	out := &tat.UsersJSON{
		Count: count,
		Users: users,
	}
	ctx.JSON(http.StatusOK, out)
}

// Create a new user, record Username, Fullname and Email
// A mail is sent to ask user for validation
func (u *UsersController) Create(ctx *gin.Context) {
	var userJSON tat.UserCreateJSON
	ctx.Bind(&userJSON)
	var userIn tat.User
	userIn.Username = u.computeUsername(userJSON)
	userIn.Fullname = strings.TrimSpace(userJSON.Fullname)
	userIn.Email = strings.TrimSpace(userJSON.Email)
	callback := strings.TrimSpace(userJSON.Callback)

	if len(userIn.Username) < 3 || len(userIn.Fullname) < 3 || len(userIn.Email) < 7 {
		err := fmt.Errorf("Invalid username (%s) or fullname (%s) or email (%s)", userIn.Username, userIn.Fullname, userIn.Email)
		AbortWithReturnError(ctx, http.StatusInternalServerError, err)
		return
	}

	if err := u.checkAllowedDomains(userJSON); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	user := tat.User{}
	foundEmail, errEmail := userDB.FindByEmail(&user, userJSON.Email)
	foundUsername, errUsername := userDB.FindByUsername(&user, userJSON.Username)
	foundFullname, errFullname := userDB.FindByFullname(&user, userJSON.Fullname)

	if foundEmail || foundUsername || foundFullname || errEmail != nil || errUsername != nil || errFullname != nil {
		e := fmt.Errorf("Please check your username, email or fullname. If you are already registered, please reset your password")
		AbortWithReturnError(ctx, http.StatusBadRequest, e)
		return
	}

	tokenVerify, err := userDB.Insert(&userIn)
	if err != nil {
		log.Errorf("Error while InsertUser %s", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	go userDB.SendVerifyEmail(userIn.Username, userIn.Email, tokenVerify, callback)

	info := ""
	if viper.GetBool("username_from_email") {
		info = fmt.Sprintf(" Note that configuration of Tat forced your username to %s", userIn.Username)
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("please check your mail to validate your account.%s", info)})
}

func (u *UsersController) checkAllowedDomains(userJSON tat.UserCreateJSON) error {
	if viper.GetString("allowed_domains") != "" {
		allowedDomains := strings.Split(viper.GetString("allowed_domains"), ",")
		for _, domain := range allowedDomains {
			if strings.HasSuffix(userJSON.Email, "@"+domain) {
				return nil
			}
		}
		return fmt.Errorf("Your email domain is not allowed on this instance of Tat.")
	}
	return nil
}

// computeUsername returns first.lastname for first.lastname@domainA.com if
// parameter username_from_email=true on tat binary
func (u *UsersController) computeUsername(userJSON tat.UserCreateJSON) string {
	if viper.GetBool("username_from_email") {
		i := strings.Index(userJSON.Email, "@")
		if i > 0 {
			return userJSON.Email[0:i]
		}
	}
	return userJSON.Username
}

// Verify is called by user, after receive email to validate his account
func (u *UsersController) Verify(ctx *gin.Context) {
	var user = &tat.User{}
	username, err := GetParam(ctx, "username")
	if err != nil {
		return
	}
	tokenVerify, err := GetParam(ctx, "tokenVerify")
	if err != nil {
		return
	}
	if username != "" && tokenVerify != "" {
		_, password, err := userDB.Verify(user, username, tokenVerify)
		if err != nil {
			e := fmt.Sprintf("Error on verify token for username %s", username)
			log.Errorf("%s %s", e, err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"info": e})
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"message":  "Verification successful",
				"username": username,
				"password": password,
				"url":      fmt.Sprintf("%s://%s:%s%s", viper.GetString("exposed_scheme"), viper.GetString("exposed_host"), viper.GetString("exposed_port"), viper.GetString("exposed_path")),
			})
		}
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"info": fmt.Sprintf("username %s or token empty", username)})
	}
}

type userResetJSON struct {
	Username string `json:"username"  binding:"required"`
	Email    string `json:"email"     binding:"required"`
	Callback string `json:"callback"`
}

// Reset send a mail asking user to confirm reset password
func (u *UsersController) Reset(ctx *gin.Context) {
	var userJSON userResetJSON
	ctx.Bind(&userJSON)
	var userIn tat.User
	userIn.Username = strings.TrimSpace(userJSON.Username)
	userIn.Email = strings.TrimSpace(userJSON.Email)
	callback := strings.TrimSpace(userJSON.Callback)

	if len(userIn.Username) < 3 || len(userIn.Email) < 7 {
		err := fmt.Errorf("Invalid username (%s) or email (%s)", userIn.Username, userIn.Email)
		AbortWithReturnError(ctx, http.StatusInternalServerError, err)
		return
	}

	tokenVerify, err := userDB.AskReset(&userIn)
	if err != nil {
		log.Errorf("Error while AskReset %s", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	go userDB.SendAskResetEmail(userIn.Username, userIn.Email, tokenVerify, callback)
	ctx.JSON(http.StatusCreated, gin.H{"info": "please check your mail to validate your account"})
}

// Me retrieves all information about me (exception information about Authentication)
func (*UsersController) Me(ctx *gin.Context) {
	var user = tat.User{}
	found, err := userDB.FindByUsername(&user, getCtxUsername(ctx))
	if !found {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching user"})
		return
	}
	gs, errGetGroupsOnlyName := groupDB.GetUserGroupsOnlyName(user.Username)
	if errGetGroupsOnlyName != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while getting groups"})
		return
	}
	user.Groups = gs
	out := &tat.UserJSON{User: user}
	ctx.JSON(http.StatusOK, out)
}

// Contacts retrieves contacts presences since n seconds
func (*UsersController) Contacts(ctx *gin.Context) {
	sinceSeconds, err := GetParam(ctx, "sinceSeconds")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error while getting seconds parameter"})
		return
	}
	seconds, err := strconv.ParseInt(sinceSeconds, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid since parameter : must be an interger"})
		return
	}

	var user = tat.User{}
	found, err := userDB.FindByUsername(&user, getCtxUsername(ctx))
	if !found {
		ctx.JSON(http.StatusInternalServerError, errors.New("User unknown"))
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, errors.New("Error while fetching user"))
		return
	}
	criteria := tat.PresenceCriteria{}
	for _, contact := range user.Contacts {
		criteria.Username = criteria.Username + "," + contact.Username
	}
	criteria.DateMinPresence = strconv.FormatInt(time.Now().Unix()-seconds, 10)
	count, presences, _ := presenceDB.ListPresences(&criteria)

	out := &tat.ContactsJSON{
		Contacts:               user.Contacts,
		CountContactsPresences: count,
		ContactsPresences:      &presences,
	}
	ctx.JSON(http.StatusOK, out)
}

// AddContact add a contact to user
func (*UsersController) AddContact(ctx *gin.Context) {
	contactIn, err := GetParam(ctx, "username")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	var contact = tat.User{}
	found, err := userDB.FindByUsername(&contact, contactIn)
	if !found {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s does not exist", contactIn))
		return
	} else if err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while fetching user with username %s", contactIn))
		return
	}

	if err := userDB.AddContact(&user, contact.Username, contact.Fullname); err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while add contact %s to user:%s", contact.Username, user.Username))
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// RemoveContact removes a contact from user
func (*UsersController) RemoveContact(ctx *gin.Context) {
	contactIn, err := GetParam(ctx, "username")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	if err := userDB.RemoveContact(&user, contactIn); err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while remove contact %s to user:%s", contactIn, user.Username))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": "Contact " + contactIn + " is removed"})
}

// AddFavoriteTopic add a favorite topic to user
func (*UsersController) AddFavoriteTopic(ctx *gin.Context) {
	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	topic, err := topicDB.FindByTopic(topicIn, true, false, false, &user)
	if err != nil {
		AbortWithReturnError(ctx, http.StatusBadRequest, errors.New("topic "+topicIn+" does not exist or you have no Read Access on it"))
		return
	}

	if err := userDB.AddFavoriteTopic(&user, topic.Topic); err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while add favorite topic to user:%s", user.Username))
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("Topic %s added to favorites", topic.Topic)})
}

// RemoveFavoriteTopic removes favorite topic from user
func (*UsersController) RemoveFavoriteTopic(ctx *gin.Context) {
	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	if err := userDB.RemoveFavoriteTopic(&user, topicIn); err != nil {
		e := fmt.Errorf("Error while remove favorite topic %s to user:%s err:%s", topicIn, user.Username, err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"info": fmt.Sprintf("Topic %s removed from favorites", topicIn)})
}

// EnableNotificationsTopic enable notication on one topic
func (*UsersController) EnableNotificationsTopic(ctx *gin.Context) {
	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	topic, err := topicDB.FindByTopic(topicIn, true, false, false, &user)
	if err != nil {
		AbortWithReturnError(ctx, http.StatusBadRequest, errors.New("topic "+topicIn+" does not exist or you have no Read Access on it"))
		return
	}

	if err := userDB.EnableNotificationsTopic(&user, topic.Topic); err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while enable notication on topic %s to user:%s", topic.Topic, user.Username))
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("Notications enabled on Topic %s", topic.Topic)})
}

// DisableNotificationsTopic disable notifications on one topic, except /Private/*
func (*UsersController) DisableNotificationsTopic(ctx *gin.Context) {
	topicIn, err := GetParam(ctx, "topic")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	if err := userDB.DisableNotificationsTopic(&user, topicIn); err != nil {
		e := fmt.Errorf("Error while disable notications on topic %s to user:%s err:%s", topicIn, user.Username, err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("Notications disabled on topic %s", topicIn)})
}

// EnableNotificationsAllTopics enables notifications on all topics
func (*UsersController) EnableNotificationsAllTopics(ctx *gin.Context) {
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	if err := userDB.EnableNotificationsAllTopics(&user); err != nil {
		e := fmt.Errorf("Error while enable notications on all topics to user:%s err:%s", user.Username, err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("Notications enabled on all topics")})
}

// DisableNotificationsAllTopics disables notifications on all topics
func (*UsersController) DisableNotificationsAllTopics(ctx *gin.Context) {
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	if err := userDB.DisableNotificationsAllTopics(&user); err != nil {
		e := fmt.Errorf("Error while disable notications on all topics to user:%s err:%s", user.Username, err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": fmt.Sprintf("Notications disabled on all topics")})
}

// AddFavoriteTag add a favorite tag to user
func (*UsersController) AddFavoriteTag(ctx *gin.Context) {
	tagIn, err := GetParam(ctx, "tag")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	if err = userDB.AddFavoriteTag(&user, tagIn); err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while add favorite tag to user:%s", user.Username))
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// RemoveFavoriteTag removes a favorite tag from user
func (*UsersController) RemoveFavoriteTag(ctx *gin.Context) {
	tagIn, err := GetParam(ctx, "tag")
	if err != nil {
		return
	}
	user, err := PreCheckUser(ctx)
	if err != nil {
		return
	}

	if err := userDB.RemoveFavoriteTag(&user, tagIn); err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while remove favorite tag to user:%s", user.Username))
		return
	}
	ctx.JSON(http.StatusOK, "")
}

// Convert a "normal" user to a "system" user
func (*UsersController) Convert(ctx *gin.Context) {
	var convertJSON tat.ConvertUserJSON
	ctx.Bind(&convertJSON)

	if !strings.HasPrefix(convertJSON.Username, "tat.system") {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("Username does not begin with tat.system (%s), it's not possible to convert this user", convertJSON.Username))
		return
	}

	var userToConvert = tat.User{}
	found, err := userDB.FindByUsername(&userToConvert, convertJSON.Username)
	if !found {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s does not exist", convertJSON.Username))
		return
	} else if err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while fetching user with username %s", convertJSON.Username))
		return
	}

	if userToConvert.IsSystem {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s is already a system user", convertJSON.Username))
		return
	}

	newPassword, err := userDB.ConvertToSystem(&userToConvert, getCtxUsername(ctx), convertJSON.CanWriteNotifications, convertJSON.CanListUsersAsAdmin)
	if err != nil {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("Convert %s to system user failed", convertJSON.Username))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":  "Verification successful",
		"username": userToConvert.Username,
		"password": newPassword,
		"url":      fmt.Sprintf("%s://%s:%s%s", viper.GetString("exposed_scheme"), viper.GetString("exposed_host"), viper.GetString("exposed_port"), viper.GetString("exposed_path")),
	})
}

type resetSystemUserJSON struct {
	Username string `json:"username"  binding:"required"`
}

// UpdateSystemUser updates flags CanWriteNotifications and CanListUsersAsAdmin
func (*UsersController) UpdateSystemUser(ctx *gin.Context) {
	var convertJSON tat.ConvertUserJSON
	ctx.Bind(&convertJSON)

	if !strings.HasPrefix(convertJSON.Username, "tat.system") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Username does not begin with tat.system (%s), it's not possible to update this user", convertJSON.Username)})
		return
	}

	var userToConvert = tat.User{}
	found, err := userDB.FindByUsername(&userToConvert, convertJSON.Username)
	if !found {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("user with username %s does not exist", convertJSON.Username)})
		return
	} else if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Error while fetching user with username %s", convertJSON.Username)})
		return
	}

	if !userToConvert.IsSystem {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("user with username %s is not a system user", convertJSON.Username)})
		return
	}

	err2 := userDB.UpdateSystemUser(&userToConvert, convertJSON.CanWriteNotifications, convertJSON.CanListUsersAsAdmin)
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Error while update system user %s", convertJSON.Username)})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Update successful"})
}

// ResetSystemUser reset password for a system user
func (*UsersController) ResetSystemUser(ctx *gin.Context) {
	var systemUserJSON resetSystemUserJSON
	ctx.Bind(&systemUserJSON)

	if !strings.HasPrefix(systemUserJSON.Username, "tat.system") {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("Username does not begin with tat.system (%s), it's not possible to reset password for this user", systemUserJSON.Username))
		return
	}

	var systemUserToReset = tat.User{}
	found, err := userDB.FindByUsername(&systemUserToReset, systemUserJSON.Username)
	if !found {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s does not exist", systemUserJSON.Username))
		return
	} else if err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while fetching user with username %s", systemUserJSON.Username))
		return
	}

	if !systemUserToReset.IsSystem {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s is not a system user", systemUserJSON.Username))
		return
	}

	newPassword, err := userDB.ResetSystemUserPassword(&systemUserToReset)
	if err != nil {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("Reset password for %s (system user) failed", systemUserJSON.Username))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":  "Reset password successful",
		"username": systemUserToReset.Username,
		"password": newPassword,
		"url":      fmt.Sprintf("%s://%s:%s%s", viper.GetString("exposed_scheme"), viper.GetString("exposed_host"), viper.GetString("exposed_port"), viper.GetString("exposed_path")),
	})
}

// SetAdmin a "normal" user to an admin user
func (*UsersController) SetAdmin(ctx *gin.Context) {
	var convertJSON tat.UsernameUserJSON
	ctx.Bind(&convertJSON)

	var userToGrant = tat.User{}
	found, err := userDB.FindByUsername(&userToGrant, convertJSON.Username)
	if !found {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s does not exist", convertJSON.Username))
		return
	} else if err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error while fetching user with username %s", convertJSON.Username))
		return
	}

	if userToGrant.IsAdmin {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s is already an admin user", convertJSON.Username))
		return
	}

	if err := userDB.ConvertToAdmin(&userToGrant, getCtxUsername(ctx)); err != nil {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("Convert %s to admin user failed", convertJSON.Username))
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// Archive a user
func (*UsersController) Archive(ctx *gin.Context) {
	var archiveJSON tat.UsernameUserJSON
	ctx.Bind(&archiveJSON)

	var userToArchive = tat.User{}
	found, err := userDB.FindByUsername(&userToArchive, archiveJSON.Username)
	if !found {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s does not exist", archiveJSON.Username))
		return
	} else if err != nil {
		AbortWithReturnError(ctx, http.StatusInternalServerError, fmt.Errorf("Error whil fetching user user with username %s", archiveJSON.Username))
		return
	}

	if userToArchive.IsArchived {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("user with username %s is already archived", archiveJSON.Username))
		return
	}

	if err := userDB.Archive(&userToArchive, getCtxUsername(ctx)); err != nil {
		AbortWithReturnError(ctx, http.StatusBadRequest, fmt.Errorf("archive user %s failed", archiveJSON.Username))
		return
	}

	ctx.JSON(http.StatusCreated, "")
}

// Rename a username of one user
func (*UsersController) Rename(ctx *gin.Context) {
	var renameJSON tat.RenameUserJSON
	ctx.Bind(&renameJSON)

	var userToRename = tat.User{}
	found, err := userDB.FindByUsername(&userToRename, renameJSON.Username)
	if !found {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("user with username %s does not exist", renameJSON.Username)})
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("Error while fetching user with username %s", renameJSON.Username)})
		return
	}

	if err := userDB.Rename(&userToRename, renameJSON.NewUsername); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("Rename %s user to %s failed", renameJSON.Username, renameJSON.NewUsername)})
		return
	}

	if err := messageDB.ChangeUsernameOnMessages(userToRename.Username, renameJSON.NewUsername); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("Rename %s user to %s failed", renameJSON.Username, renameJSON.NewUsername)})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"info": "user is renamed"})
}

// Update changes fullname and email
func (*UsersController) Update(ctx *gin.Context) {
	var updateJSON tat.UpdateUserJSON
	ctx.Bind(&updateJSON)

	var userToUpdate = tat.User{}
	found, err := userDB.FindByUsername(&userToUpdate, updateJSON.Username)
	if !found {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("user with username %s does not exist", updateJSON.Username)})
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("Error while fetching user with username %s", updateJSON.Username)})
		return
	}

	if strings.TrimSpace(updateJSON.NewFullname) == "" || strings.TrimSpace(updateJSON.NewEmail) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("Invalid Fullname %s or Email %s", updateJSON.NewFullname, updateJSON.NewEmail)})
		return
	}

	err2 := userDB.Update(&userToUpdate, strings.TrimSpace(updateJSON.NewFullname), strings.TrimSpace(updateJSON.NewEmail))
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Update %s user to fullname %s and email %s failed : %s", updateJSON.Username, updateJSON.NewFullname, updateJSON.NewEmail, err2.Error())})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"info": "user updated"})
}

// Check if user have his Private topics
// /Private/username, /Private/username/Tasks
func (u *UsersController) Check(ctx *gin.Context) {

	var userJSON tat.CheckTopicsUserJSON
	ctx.Bind(&userJSON)

	var userToCheck = tat.User{}
	found, err := userDB.FindByUsername(&userToCheck, userJSON.Username)
	if !found {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("user with username %s does not exist", userJSON.Username)})
		return
	} else if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("Error while fetching user with username %s", userJSON.Username)})
		return
	}

	topicsInfo := userDB.CheckTopics(&userToCheck, userJSON.FixPrivateTopics)
	defaultGroupInfo := userDB.CheckDefaultGroup(&userToCheck, userJSON.FixDefaultGroup)

	ctx.JSON(http.StatusCreated, gin.H{"topics": topicsInfo, "defaultGroup": defaultGroupInfo})
}
