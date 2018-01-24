package tat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Contact User Struct.
type Contact struct {
	Username string `bson:"username" json:"username"`
	Fullname string `bson:"fullname" json:"fullname"`
}

// Auth User Struct
type Auth struct {
	HashedPassword    string `bson:"hashedPassword" json:"-"`
	HashedTokenVerify string `bson:"hashedTokenVerify" json:"-"`
	DateRenewPassword int64  `bson:"dateRenewPassword" json:"dateRenewPassword"`
	DateAskReset      int64  `bson:"dateAskReset" json:"dateAskReset"`
	DateVerify        int64  `bson:"dateVerify" json:"dateVerify"`
	EmailVerified     bool   `bson:"emailVerified" json:"emailVerified"`
}

// User struct
type User struct {
	ID                     string    `bson:"_id" json:"_id"`
	Username               string    `bson:"username" json:"username"`
	Fullname               string    `bson:"fullname" json:"fullname"`
	Email                  string    `bson:"email" json:"email,omitempty"`
	Groups                 []string  `bson:"-" json:"groups,omitempty"`
	IsAdmin                bool      `bson:"isAdmin" json:"isAdmin,omitempty"`
	IsSystem               bool      `bson:"isSystem" json:"isSystem,omitempty"`
	IsArchived             bool      `bson:"isArchived" json:"isArchived,omitempty"`
	CanWriteNotifications  bool      `bson:"canWriteNotifications" json:"canWriteNotifications,omitempty"`
	CanListUsersAsAdmin    bool      `bson:"canListUsersAsAdmin" json:"canListUsersAsAdmin,omitempty"`
	FavoritesTopics        []string  `bson:"favoritesTopics" json:"favoritesTopics,omitempty"`
	OffNotificationsTopics []string  `bson:"offNotificationsTopics" json:"offNotificationsTopics,omitempty"`
	FavoritesTags          []string  `bson:"favoritesTags" json:"favoritesTags,omitempty"`
	DateCreation           int64     `bson:"dateCreation" json:"dateCreation,omitempty"`
	Contacts               []Contact `bson:"contacts" json:"contacts,omitempty"`
	Auth                   Auth      `bson:"auth" json:"-"`
}

// UsersJSON  represents list of users and count for total
type UsersJSON struct {
	Count int    `json:"count"`
	Users []User `json:"users"`
}

// UserCreateJSON is used for create a new user
type UserCreateJSON struct {
	Username string `json:"username"  binding:"required"`
	Fullname string `json:"fullname"  binding:"required"`
	Email    string `json:"email"     binding:"required"`
	// Callback contains command to execute to verify account
	// this command is displayed in ask for confirmation mail
	Callback string `json:"callback"`
}

// UserResetJSON is used for reset a new user
type UserResetJSON struct {
	Username string `json:"username"  binding:"required"`
	Email    string `json:"email"     binding:"required"`
	// Callback contains command to execute to verify account
	// this command is displayed in ask for confirmation mail
	Callback string `json:"callback"`
}

// UpdateUserJSON is used for update user information
type UpdateUserJSON struct {
	Username    string `json:"username" binding:"required"`
	NewFullname string `json:"newFullname" binding:"required"`
	NewEmail    string `json:"newEmail" binding:"required"`
}

// UserCriteria is used to list users with criterias
type UserCriteria struct {
	Skip            int
	Limit           int
	WithGroups      bool
	IDUser          string
	Username        string
	Fullname        string
	DateMinCreation string
	DateMaxCreation string
	SortBy		string
}

//ContactsJSON represents a contact for a user, in contacts attribute on a user
type ContactsJSON struct {
	Contacts               []Contact   `json:"contacts"`
	CountContactsPresences int         `json:"countContactsPresences"`
	ContactsPresences      *[]Presence `json:"contactsPresence"`
}

// UserJSON used by GET /user/me
type UserJSON struct {
	User User `json:"user"`
}

// UsernameUserJSON contains just a username
type UsernameUserJSON struct {
	Username string `json:"username" binding:"required"`
}

// CheckTopicsUserJSON used to check if user have default topics
type CheckTopicsUserJSON struct {
	Username         string `json:"username"  binding:"required"`
	FixPrivateTopics bool   `json:"fixPrivateTopics"  binding:"required"`
	FixDefaultGroup  bool   `json:"fixDefaultGroup"  binding:"required"`
}

// ConvertUserJSON is used to convert a user to a system user
type ConvertUserJSON struct {
	Username              string `json:"username" binding:"required"`
	CanWriteNotifications bool   `json:"canWriteNotifications" binding:"required"`
	CanListUsersAsAdmin   bool   `json:"canListUsersAsAdmin" binding:"required"`
}

// RenameUserJSON is used for rename a user
type RenameUserJSON struct {
	Username    string `json:"username"  binding:"required"`
	NewUsername string `json:"newUsername"  binding:"required"`
}

// VerifyJSON is used for returns password for a user with verify action
type VerifyJSON struct {
	Message  string `json:"message,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	URL      string `json:"url,omitempty"`
}

// UserList returns all users
func (c *Client) UserList(criteria *UserCriteria) (*UsersJSON, error) {
	if c == nil {
		return nil, ErrClientNotInitiliazed
	}

	if criteria == nil {
		criteria = &UserCriteria{
			Skip:  0,
			Limit: 100,
		}
	}

	v := url.Values{}
	v.Set("skip", strconv.Itoa(criteria.Skip))
	v.Set("limit", strconv.Itoa(criteria.Limit))

	v.Set("withGroups", strconv.FormatBool(criteria.WithGroups))
	v.Set("idUser", criteria.IDUser)
	v.Set("username", criteria.Username)
	v.Set("fullname", criteria.Fullname)
	v.Set("dateMinCreation", criteria.DateMinCreation)
	v.Set("dateMaxCreation", criteria.DateMaxCreation)

	path := fmt.Sprintf("/users?%s", v.Encode())

	body, err := c.reqWant(http.MethodGet, 200, path, nil)
	if err != nil {
		ErrorLogFunc("Error getting users list: %s", err)
		return nil, err
	}

	DebugLogFunc("Users List Reponse: %s", string(body))
	var users = UsersJSON{}
	if err := json.Unmarshal(body, &users); err != nil {
		ErrorLogFunc("Error getting user list: %s", err)
		return nil, err
	}

	return &users, nil
}

// UserMe returns current user
func (c *Client) UserMe() (*UserJSON, error) {
	b, err := c.reqWant("GET", http.StatusOK, "/user/me", nil)

	if err != nil {
		ErrorLogFunc("Error while GET /user/me: %s", err)
		return nil, err
	}

	userJSON := &UserJSON{}
	if err := json.Unmarshal(b, userJSON); err != nil {
		return nil, err
	}

	return userJSON, nil
}

// UserContacts returns contacts presences since n seconds
func (c *Client) UserContacts(sinceSeconds int) ([]byte, error) {
	return c.simpleGetAndGetBytes(fmt.Sprintf("/user/me/contacts/%d", sinceSeconds))
}

// UserAddContact adds a contact
func (c *Client) UserAddContact(toAdd string) ([]byte, error) {
	return c.simplePostAndGetBytes("/user/me/contacts/"+toAdd, 201, nil)
}

// UserRemoveContact removes a contact from a user
func (c *Client) UserRemoveContact(toRemove string) ([]byte, error) {
	return c.simpleDeleteAndGetBytes("/user/me/contacts/"+toRemove, 200, nil)
}

// UserAddFavoriteTopic adds a favorite topic on current user
func (c *Client) UserAddFavoriteTopic(toAdd string) ([]byte, error) {
	return c.simplePostAndGetBytes("/user/me/topics"+toAdd, 201, nil)
}

// UserRemoveFavoriteTopic remove a favorite topic from current user
func (c *Client) UserRemoveFavoriteTopic(toRemove string) ([]byte, error) {
	return c.simpleDeleteAndGetBytes("/user/me/topics"+toRemove, 200, nil)
}

// UserEnableNotificationsTopic enables notifications on one topic
func (c *Client) UserEnableNotificationsTopic(topic string) ([]byte, error) {
	return c.simplePostAndGetBytes("/user/me/enable/notifications/topics"+topic, 201, nil)
}

// UserEnableNotificationsAllTopics enables notification on all topics
func (c *Client) UserEnableNotificationsAllTopics() ([]byte, error) {
	return c.simplePostAndGetBytes("/user/me/disable/notifications/alltopics", 201, nil)
}

// UserDisableNotificationsTopic disables notifications on one topic
func (c *Client) UserDisableNotificationsTopic(topic string) ([]byte, error) {
	return c.simplePostAndGetBytes("/user/me/disable/notifications/topics"+topic, 201, nil)
}

// UserDisableNotificationsAllTopics disable notification on all topics
func (c *Client) UserDisableNotificationsAllTopics() ([]byte, error) {
	return c.simplePostAndGetBytes("/user/me/disable/notifications/alltopics", 201, nil)
}

// UserAddFavoriteTag adds a favorite tag to current user
func (c *Client) UserAddFavoriteTag(toAdd string) ([]byte, error) {
	return c.simplePostAndGetBytes("/user/me/tags/"+toAdd, 201, nil)
}

// UserRemoveFavoriteTag removes a favorite tag from current user
func (c *Client) UserRemoveFavoriteTag(toRemove string) ([]byte, error) {
	return c.simpleDeleteAndGetBytes("/user/me/tags/"+toRemove, 200, nil)
}

// UserAdd creates a new user
// if callback is "", "tatcli --url=:scheme://:host::port:path user verify --save :username :token" will
// be used
func (c *Client) UserAdd(u UserCreateJSON) ([]byte, error) {
	if u.Callback == "" {
		u.Callback = "tatcli --url=:scheme://:host::port:path user verify --save :username :token"
	}
	b, err := json.Marshal(u)
	if err != nil {
		ErrorLogFunc("UserAdd> Error while marshal user: %s", err)
		return nil, err
	}

	return c.reqWant("POST", http.StatusCreated, "/user", b)
}

// UserReset is used for reset password for a user
func (c *Client) UserReset(v UserResetJSON) ([]byte, error) {
	return c.simplePostAndGetBytes("/user/reset", 201, v)
}

// UserResetSystem is used for reset password for a system user
func (c *Client) UserResetSystem(v UsernameUserJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/user/resetsystem", 201, v)
}

// UserConvertToSystem converts a user to a system user
func (c *Client) UserConvertToSystem(s ConvertUserJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/user/convert", http.StatusCreated, s)
}

// UserUpdateSystem updates a system user
func (c *Client) UserUpdateSystem(u ConvertUserJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/user/updatesystem", http.StatusCreated, u)
}

// UserArchive archives a user
func (c *Client) UserArchive(username string) error {
	m := UsernameUserJSON{Username: username}
	jsonStr, err := json.Marshal(m)
	if err != nil {
		ErrorLogFunc("UserAdd> Error while marshal username: %s", err)
		return err
	}

	_, err = c.reqWant("PUT", http.StatusCreated, "/user/archive", jsonStr)
	return err
}

// UserRename renames a user
func (c *Client) UserRename(v RenameUserJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/user/rename", 201, v)
}

// UserUpdate is used for update current user
func (c *Client) UserUpdate(v UpdateUserJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/user/update", 201, v)
}

// UserSetAdmin set a user as an admin
func (c *Client) UserSetAdmin(u UsernameUserJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/user/setadmin", http.StatusCreated, u)
}

// UserVerify is used for verify user and returns password
func (c *Client) UserVerify(username, tokenVerify string) (*VerifyJSON, error) {
	path := fmt.Sprintf("/user/verify/%s/%s", username, tokenVerify)
	out, err := c.simpleGetAndGetBytes(path)
	if err != nil {
		ErrorLogFunc("Error while GET /user/verify: %s", err)
		return nil, err
	}

	verifyJSON := &VerifyJSON{}
	if err := json.Unmarshal(out, verifyJSON); err != nil {
		return nil, err
	}
	return verifyJSON, nil
}

// UserCheck checks if user have default topics
func (c *Client) UserCheck(check CheckTopicsUserJSON) ([]byte, error) {
	return c.simplePutAndGetBytes("/user/check", http.StatusCreated, check)
}
