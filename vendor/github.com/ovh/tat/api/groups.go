package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat"
	groupDB "github.com/ovh/tat/api/group"
	topicDB "github.com/ovh/tat/api/topic"
	userDB "github.com/ovh/tat/api/user"
)

// GroupsController contains all methods about groups manipulation
type GroupsController struct{}

func (*GroupsController) buildCriteria(ctx *gin.Context) *tat.GroupCriteria {
	c := tat.GroupCriteria{}
	skip, e := strconv.Atoi(ctx.DefaultQuery("skip", "0"))
	if e != nil {
		skip = 0
	}
	c.Skip = skip

	limit, e2 := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	if e2 != nil {
		limit = 100
	}
	c.Limit = limit
	c.IDGroup = ctx.Query("idGroup")
	c.Name = ctx.Query("name")
	c.Description = ctx.Query("description")
	c.DateMinCreation = ctx.Query("dateMinCreation")
	c.DateMaxCreation = ctx.Query("dateMaxCreation")
	return &c
}

// List list groups with given criteria
func (g *GroupsController) List(ctx *gin.Context) {
	var criteria tat.GroupCriteria
	ctx.Bind(&criteria)

	user, err := PreCheckUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching current user"})
		return
	}
	count, groups, err := groupDB.ListGroups(g.buildCriteria(ctx), &user, isTatAdmin(ctx))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	out := &tat.GroupsJSON{
		Count:  count,
		Groups: groups,
	}
	ctx.JSON(http.StatusOK, out)
}

// Create creates a new group
func (*GroupsController) Create(ctx *gin.Context) {
	var groupBind tat.GroupJSON
	ctx.Bind(&groupBind)

	var groupIn tat.Group
	groupIn.Name = groupBind.Name
	groupIn.Description = groupBind.Description

	err := groupDB.Insert(&groupIn)
	if err != nil {
		log.Errorf("Error while InsertGroup %s", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusCreated, groupIn)
}

func (*GroupsController) preCheckUser(ctx *gin.Context, paramJSON *tat.ParamGroupUserJSON) (tat.Group, error) {
	user := tat.User{}
	found, err := userDB.FindByUsername(&user, paramJSON.Username)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return tat.Group{}, err
	}

	if !found {
		e := errors.New("username " + paramJSON.Username + " does not exist")
		ctx.AbortWithError(http.StatusInternalServerError, e)
		return tat.Group{}, e
	}

	group, errfinding := groupDB.FindByName(paramJSON.Groupname)
	if errfinding != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errfinding)
		return tat.Group{}, errfinding
	}

	if isTatAdmin(ctx) { // if Tat admin, ok
		return *group, nil
	}

	if !groupDB.IsUserAdmin(group, getCtxUsername(ctx)) {
		e := fmt.Errorf("user %s is not admin on group %s", user.Username, group.Name)
		ctx.AbortWithError(http.StatusInternalServerError, e)
		return tat.Group{}, e
	}

	return *group, nil
}

type groupUpdateJSON struct {
	Name        string `json:"newName" binding:"required"`
	Description string `json:"newDescription" binding:"required"`
}

// Update a group
// only for Tat admin
func (g *GroupsController) Update(ctx *gin.Context) {
	var paramJSON groupUpdateJSON
	ctx.Bind(&paramJSON)

	group, err := GetParam(ctx, "group")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Group in query"})
		return
	}

	groupToUpdate, err := groupDB.FindByName(group)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Group"})
		return
	}

	if paramJSON.Name != groupToUpdate.Name {
		groupnameExists := groupDB.IsGroupnameExists(paramJSON.Name)

		if groupnameExists {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Group Name already exists"})
			return
		}
	}

	user, err := PreCheckUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error while fetching user"})
		return
	}

	err = groupDB.Update(groupToUpdate, paramJSON.Name, paramJSON.Description, &user)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error while update group"})
		return
	}

	if paramJSON.Name != groupToUpdate.Name {
		if err := topicDB.ChangeGroupnameOnTopics(groupToUpdate.Name, paramJSON.Name); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error while update group on topics"})
			return
		}
	}

	ctx.JSON(http.StatusCreated, "")
}

// Delete deletes requested group
// only for Tat admin
func (g *GroupsController) Delete(ctx *gin.Context) {
	groupName, err := GetParam(ctx, "group")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Group in query"})
		return
	}

	groupToDelete, err := groupDB.FindByName(groupName)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Invalid Group"})
		return
	}

	user, err := PreCheckUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Error while fetching user"})
		return
	}

	c := tat.TopicCriteria{}
	c.Skip = 0
	c.Limit = 10
	c.Group = groupToDelete.Name

	count, topics, err := topicDB.ListTopics(&c, &user, false, false, false)
	if err != nil {
		log.Errorf("Error while getting topics associated to group %s:%s", groupToDelete.Name, err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf("Error while getting topics associated to group")})
		return
	}

	if len(topics) > 0 {
		e := fmt.Sprintf("Group %s associated to %d topic, you can't delete it", groupToDelete.Name, count)
		log.Errorf(e)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Errorf(e)})
		return
	}

	if err = groupDB.Delete(groupToDelete, &user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error while deleting Group: %s", err.Error())})
		return
	}
	ctx.JSON(http.StatusOK, "")
}

// AddUser add a user to a group
func (g *GroupsController) AddUser(ctx *gin.Context) {
	var paramJSON tat.ParamGroupUserJSON
	ctx.Bind(&paramJSON)
	group, e := g.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	if err := groupDB.AddUser(&group, getCtxUsername(ctx), paramJSON.Username); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error while add user to group: %s", err)})
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// RemoveUser removes user from a group
func (g *GroupsController) RemoveUser(ctx *gin.Context) {
	var paramJSON tat.ParamGroupUserJSON
	ctx.Bind(&paramJSON)
	group, e := g.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	if err := groupDB.RemoveUser(&group, getCtxUsername(ctx), paramJSON.Username); err != nil {
		return
	}
	ctx.JSON(http.StatusOK, "")
}

// AddAdminUser add a user to a group
func (g *GroupsController) AddAdminUser(ctx *gin.Context) {
	var paramJSON tat.ParamGroupUserJSON
	ctx.Bind(&paramJSON)
	group, e := g.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	if err := groupDB.AddAdminUser(&group, getCtxUsername(ctx), paramJSON.Username); err != nil {
		return
	}
	ctx.JSON(http.StatusCreated, "")
}

// RemoveAdminUser removes user from a group
func (g *GroupsController) RemoveAdminUser(ctx *gin.Context) {
	var paramJSON tat.ParamGroupUserJSON
	ctx.Bind(&paramJSON)
	group, e := g.preCheckUser(ctx, &paramJSON)
	if e != nil {
		return
	}

	if err := groupDB.RemoveAdminUser(&group, getCtxUsername(ctx), paramJSON.Username); err != nil {
		return
	}
	ctx.JSON(http.StatusOK, "")
}
