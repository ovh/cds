package user

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/ovh/tat"
	"github.com/ovh/tat/api/cache"
	"github.com/ovh/tat/api/group"
	"github.com/ovh/tat/api/presence"
	"github.com/ovh/tat/api/store"
	"github.com/ovh/tat/api/topic"
	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/redis.v4"
)

func buildUserCriteria(criteria *tat.UserCriteria) (bson.M, error) {
	var query = []bson.M{}
	query = append(query, bson.M{"isArchived": false})

	if criteria.IDUser != "" {
		queryIDUsers := bson.M{}
		queryIDUsers["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.IDUser, ",") {
			queryIDUsers["$or"] = append(queryIDUsers["$or"].([]bson.M), bson.M{"_id": val})
		}
		query = append(query, queryIDUsers)
	}
	if criteria.Username != "" {
		queryUsernames := bson.M{}
		queryUsernames["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.Username, ",") {
			queryUsernames["$or"] = append(queryUsernames["$or"].([]bson.M), bson.M{"username": val})
		}
		query = append(query, queryUsernames)
	}
	if criteria.Fullname != "" {
		queryFullnames := bson.M{}
		queryFullnames["$or"] = []bson.M{}
		for _, val := range strings.Split(criteria.Fullname, ",") {
			queryFullnames["$or"] = append(queryFullnames["$or"].([]bson.M), bson.M{"fullname": val})
		}
		query = append(query, queryFullnames)
	}

	var bsonDate = bson.M{}

	if criteria.DateMinCreation != "" {
		i, err := strconv.ParseInt(criteria.DateMinCreation, 10, 64)
		if err != nil {
			return bson.M{}, fmt.Errorf("Error while parsing dateMinCreation %s", err)
		}
		tm := time.Unix(i, 0)
		bsonDate["$gte"] = tm.Unix()
	}
	if criteria.DateMaxCreation != "" {
		i, err := strconv.ParseInt(criteria.DateMaxCreation, 10, 64)
		if err != nil {
			return bson.M{}, fmt.Errorf("Error while parsing dateMaxCreation %s", err)
		}
		tm := time.Unix(i, 0)
		bsonDate["$lte"] = tm.Unix()
	}
	if len(bsonDate) > 0 {
		query = append(query, bson.M{"dateCreation": bsonDate})
	}

	if len(query) > 0 {
		return bson.M{"$and": query}, nil
	} else if len(query) == 1 {
		return query[0], nil
	}
	return bson.M{}, nil
}

func getUserListField(isAdmin bool) bson.M {
	if isAdmin {
		return bson.M{"username": 1,
			"fullname":              1,
			"email":                 1,
			"isAdmin":               1,
			"dateCreation":          1,
			"canWriteNotifications": 1,
			"canListUsersAsAdmin":   1,
		}
	}
	return bson.M{"username": 1,
		"fullname": 1,
	}
}

// ListUsers returns users list selected by criteria
func ListUsers(criteria *tat.UserCriteria, isAdmin bool) (int, []tat.User, error) {
	var users []tat.User

	cursor, errl := listUsersCursor(criteria, isAdmin)
	if errl != nil {
		return -1, users, errl
	}
	count, err := cursor.Count()
	if err != nil {
		return -1, users, fmt.Errorf("Error while count Users %s", err)
	}

	sortBy := criteria.SortBy
	if sortBy == "" {
		sortBy = "-dateCreation"
	}
	err = cursor.Select(getUserListField(isAdmin)).
		Sort(sortBy).
		Skip(criteria.Skip).
		Limit(criteria.Limit).
		All(&users)

	if err != nil {
		return -1, users, fmt.Errorf("Error while Find All Users %s", err)
	}

	// Admin could ask groups for all users. Not perf, but really rare
	if criteria.WithGroups && isAdmin {
		var usersWithGroups []tat.User
		for _, u := range users {
			gs, errGetGroupsOnlyName := group.GetUserGroupsOnlyName(u.Username)
			u.Groups = gs
			log.Infof("User %s, Groups%s", u.Username, u.Groups)
			if errGetGroupsOnlyName != nil {
				log.Errorf("Error while getting group for user %s, Error:%s", u.Username, errGetGroupsOnlyName)
			}
			usersWithGroups = append(usersWithGroups, u)
		}
		return count, usersWithGroups, nil
	}
	return count, users, err
}

func listUsersCursor(criteria *tat.UserCriteria, isAdmin bool) (*mgo.Query, error) {
	c, err := buildUserCriteria(criteria)
	if err != nil {
		return nil, err
	}
	return store.Tat().CUsers.Find(c), nil
}

// Insert a new user, return tokenVerify to user, in order to
// validate account after check email
func Insert(user *tat.User) (string, error) {
	user.ID = bson.NewObjectId().Hex()

	user.DateCreation = time.Now().Unix()
	user.Auth.DateAskReset = time.Now().Unix()
	user.Auth.EmailVerified = false
	user.IsSystem = false
	user.IsArchived = false
	user.CanWriteNotifications = false
	user.CanListUsersAsAdmin = false
	nbUsers, err := CountUsers()
	if err != nil {
		log.Errorf("Error while count all users%s", err)
		return "", err
	}
	if nbUsers > 0 {
		user.IsAdmin = false
	} else {
		log.Infof("user %s is the first user, he is now admin", user.Username)
		user.IsAdmin = true
	}
	tokenVerify := ""
	tokenVerify, user.Auth.HashedTokenVerify, err = generateUserPassword()
	if err != nil {
		log.Errorf("Error while generate Token Verify for new user %s", err)
		return tokenVerify, err
	}

	if err = store.Tat().CUsers.Insert(user); err != nil {
		log.Errorf("Error while inserting new user %s", err)
	}
	return tokenVerify, err
}

// AskReset generate a new saltTokenVerify / hashedTokenVerify
// return tokenVerify (to be sent to user by mail)
func AskReset(user *tat.User) (string, error) {

	err := FindByUsernameAndEmail(user, user.Username, user.Email)
	if err != nil {
		return "", err
	}

	tokenVerify, hashedTokenVerify, err := generateUserPassword()
	if err != nil {
		log.Errorf("Error while generate Token for reset password %s", err)
		return tokenVerify, err
	}

	err = store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"auth.hashedTokenVerify": hashedTokenVerify,
			"auth.dateAskReset":      time.Now().Unix(),
		}})

	if err != nil {
		log.Errorf("Error while ask reset user %s", err)
	}
	return tokenVerify, err
}

// Verify checks username and tokenVerify, if ok, return true, password if it's a new user
// Password is not stored in Database (only hashedPassword)
// return isNewUser, password, err
func Verify(user *tat.User, username, tokenVerify string) (bool, string, error) {
	emailVerified, err := findByUsernameAndTokenVerify(user, username, tokenVerify)
	if err != nil {
		return false, "", err
	}
	password, err := regenerateAndStoreAuth(user)
	CheckDefaultGroup(user, true)
	CheckTopics(user, true)
	return !emailVerified, password, err
}

func regenerateAndStoreAuth(user *tat.User) (string, error) {
	password, hashedPassword, err := generateUserPassword()
	if err != nil {
		log.Errorf("Error while genereate password for user %s", err)
		return password, err
	}
	err = store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"auth.hashedTokenVerify": "", // reset tokenVerify
			"auth.hashedPassword":    hashedPassword,
			"auth.dateVerify":        time.Now().Unix(),
			"auth.dateRenewPassword": time.Now().Unix(),
			"auth.emailVerify":       true,
		}})

	if err != nil {
		log.Errorf("Error while updating user %s", err)
	}

	return password, err
}

var fieldsExceptAuth = bson.M{
	"username":               1,
	"fullname":               1,
	"email":                  1,
	"isAdmin":                1,
	"isSystem":               1,
	"isArchived":             1,
	"canWriteNotifications":  1,
	"canListUsersAsAdmin":    1,
	"dateCreation":           1,
	"favoritesTopics":        1,
	"offNotificationsTopics": 1,
	"favoritesTags":          1,
	"contacts":               1,
}

// FindByUsernameAndPassword search username, use user's salt to generates hashedPassword
// and check username + hashedPassword in DB
func FindByUsernameAndPassword(user *tat.User, username, password string) (bool, error) {
	var tmpUser = tat.User{}
	err := store.Tat().CUsers.
		Find(bson.M{"username": username}).
		Select(bson.M{"auth.hashedPassword": 1, "auth.saltPassword": 1}).
		One(&tmpUser)

	if err == mgo.ErrNotFound {
		return false, fmt.Errorf("FindByUsernameAndPassword> Error fetching for username %s, err:%s", username, err.Error())
	} else if err != nil {
		return false, fmt.Errorf("FindByUsernameAndPassword> Error while fetching hash with username %s, err:%s", username, err.Error())
	}

	if !isCheckValid(password, tmpUser.Auth.HashedPassword) {
		return false, fmt.Errorf("FindByUsernameAndPassword> Error while checking user %s with given password", username)
	}

	// ok, user is checked, get all fields now
	return FindByUsername(user, username)
}

// TrustUsername create user is not already registered
func TrustUsername(user *tat.User, username string) error {

	var userCheck = tat.User{}
	found, errCheck := FindByUsername(&userCheck, username)

	if errCheck != nil {
		return fmt.Errorf("Error with DB Backend: %s", errCheck)
	} else if errCheck == nil && !found {

		user.Username = username
		setEmailAndFullnameFromTrustedUsername(user)

		tokenVerify, err := Insert(user)
		if err != nil {
			return fmt.Errorf("TrustUsername, Error while Insert user %s : %s", username, err.Error())
		}

		// force default group and topics, even if it should be done in Verify
		CheckDefaultGroup(user, true)
		CheckTopics(user, true)

		if _, _, err = Verify(user, username, tokenVerify); err != nil {
			return fmt.Errorf("TrustUsername, Error while verify : %s", err.Error())
		}

		log.Infof("User %s created by TrustUsername", username)
	}

	// ok, user is checked, get all fields now
	//return FindByUsername(user, username)
	found, err := FindByUsername(user, username)
	if !found || err != nil {
		return fmt.Errorf("TrustUsername, Error while find username:%s err:%s", username, err.Error())
	}

	return nil
}

func setEmailAndFullnameFromTrustedUsername(user *tat.User) {
	conf := viper.GetString("trusted_usernames_emails_fullnames")
	tuples := strings.Split(conf, ",")

	user.Fullname = user.Username
	user.Email = user.Username + "@" + viper.GetString("default_domain")

	if len(conf) < 2 {
		return
	}

	for _, tuple := range tuples {
		t := strings.Split(tuple, ":")
		if len(t) != 3 {
			log.Errorf("Misconfiguration of trusted_usernames_emails tuple:%s", tuple)
			continue
		}
		usernameTuple := t[0]
		emailTuple := t[1]
		fullnameTuple := t[2]
		if usernameTuple == user.Username && emailTuple != "" && fullnameTuple != "" {
			user.Email = emailTuple
			user.Fullname = strings.Replace(fullnameTuple, "_", " ", -1)
			return
		}
	}
}

// FindByUsernameAndPassword search username, use user's salt to generates tokenVerify
// and check username + hashedTokenVerify in DB
func findByUsernameAndTokenVerify(user *tat.User, username, tokenVerify string) (bool, error) {
	var tmpUser = tat.User{}
	err := store.Tat().CUsers.
		Find(bson.M{"username": username}).
		Select(bson.M{"auth.emailVerify": 1, "auth.hashedTokenVerify": 1, "auth.saltTokenVerify": 1, "auth.dateAskReset": 1}).
		One(&tmpUser)
	if err != nil {
		return false, fmt.Errorf("findByUsernameAndTokenVerify > Error while fetching hashed Token Verify with username %s", username)
	}

	// dateAskReset more than 30 min, expire token
	if time.Since(time.Unix(tmpUser.Auth.DateAskReset, 0)).Minutes() > 30 {
		return false, fmt.Errorf("Token Validation expired. Please ask a reset of your password with username %s", username)
	}
	if !isCheckValid(tokenVerify, tmpUser.Auth.HashedTokenVerify) {
		return false, fmt.Errorf("Error while checking user %s with given token", username)
	}

	// ok, user is checked, get all fields now
	found, err := FindByUsername(user, username)
	if !found || err != nil {
		return false, err
	}

	return tmpUser.Auth.EmailVerified, nil
}

//FindByUsernameAndEmail retrieve information from user with username
func FindByUsernameAndEmail(user *tat.User, username, email string) error {
	err := store.Tat().CUsers.
		Find(bson.M{"username": username, "email": email}).
		Select(fieldsExceptAuth).
		One(&user)
	if err != nil {
		log.Errorf("Error while fetching user with username %s", username)
	}
	return err
}

//FindByUsername retrieve information from user with username
func FindByUsername(user *tat.User, username string) (bool, error) {

	//Load from cache
	bytes, err := cache.Client().Get(cache.Key("tat", "users", username)).Bytes()
	if err != nil && err != redis.Nil {
		log.Warnf("Unable to get user from cache")
		goto loadFromDB
	}
	json.Unmarshal(bytes, user)
	//If the user has beeen successfully loaded
	if user.Username != "" {
		return true, nil
	}

loadFromDB:
	err = store.Tat().CUsers.
		Find(bson.M{"username": username}).
		Select(fieldsExceptAuth).
		One(user)

	if err == mgo.ErrNotFound {
		log.Infof("FindByUsername username %s not found", username)
		return false, nil
	} else if err != nil {
		log.Errorf("Error while fetching user with username %s err:%s", username, err)
		return false, err
	}

	//Push to cache
	bytes, err = json.Marshal(user)
	if err != nil {
		return false, err
	}
	cache.Client().Set(cache.Key("tat", "users", username), string(bytes), 12*time.Hour)
	return true, nil
}

//FindByFullname retrieve information from user with fullname
func FindByFullname(user *tat.User, fullname string) (bool, error) {
	err := store.Tat().CUsers.
		Find(bson.M{"fullname": fullname}).
		Select(fieldsExceptAuth).
		One(&user)

	if err == mgo.ErrNotFound {
		return false, nil
	} else if err != nil {
		log.Errorf("Error while fetching user with fullname %s", fullname)
		return false, err
	}
	return true, nil
}

//FindByEmail retrieve information from user with email
func FindByEmail(user *tat.User, email string) (bool, error) {
	err := store.Tat().CUsers.
		Find(bson.M{"email": email}).
		Select(fieldsExceptAuth).
		One(&user)
	if err == mgo.ErrNotFound {
		return false, nil
	} else if err != nil {
		log.Errorf("Error while fetching user with email %s", email)
		return false, err
	}
	return true, nil
}

func getFavoriteTopic(user *tat.User, topic string) (string, error) {
	for _, cur := range user.FavoritesTopics {
		if cur == topic {
			return cur, nil
		}
	}
	l := ""
	return l, fmt.Errorf("topic %s not found in favorites topics of user", topic)
}

func containsFavoriteTopic(user *tat.User, topic string) bool {
	_, err := getFavoriteTopic(user, topic)
	if err == nil {
		return true
	}
	return false
}

// AddFavoriteTopic add a favorite topic to user
func AddFavoriteTopic(user *tat.User, topic string) error {
	if containsFavoriteTopic(user, topic) {
		return fmt.Errorf("AddFavoriteTopic not possible, %s is already a favorite topic", topic)
	}

	err := store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$push": bson.M{"favoritesTopics": topic}})
	if err != nil {
		log.Errorf("Error while add favorite topic to user %s: %s", user.Username, err)
		return err
	}
	cache.CleanUsernames(user.Username)
	return nil
}

// RemoveFavoriteTopic removes a favorite topic from user
func RemoveFavoriteTopic(user *tat.User, topic string) error {
	topicName, err := tat.CheckAndFixNameTopic(topic)
	if err != nil {
		return err
	}

	t, err := getFavoriteTopic(user, topicName)
	if err != nil {
		return fmt.Errorf("Remove favorite topic is not possible, %s is not a favorite of this user", topicName)
	}

	err = store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$pull": bson.M{"favoritesTopics": t}})

	if err != nil {
		log.Errorf("Error while remove favorite topic from user %s: %s", user.Username, err)
		return err
	}
	cache.CleanUsernames(user.Username)
	return nil
}

func containsOffNotificationsTopic(user *tat.User, topic string) bool {
	_, err := getOffNotificationsTopic(user, topic)
	if err == nil {
		return true
	}
	return false
}

func getOffNotificationsTopic(user *tat.User, topic string) (string, error) {
	for _, cur := range user.OffNotificationsTopics {
		if cur == topic {
			return cur, nil
		}
	}
	l := ""
	return l, fmt.Errorf("topic %s not found in off notifications topics of user", topic)
}

// EnableNotificationsTopic remove topic from user list offNotificationsTopics
func EnableNotificationsTopic(user *tat.User, topic string) error {
	topicName, err := tat.CheckAndFixNameTopic(topic)
	if err != nil {
		return err
	}

	t, err := getOffNotificationsTopic(user, topicName)
	if err != nil {
		return fmt.Errorf("Enable notifications on topic %s is not possible, notifications are already enabled", topicName)
	}

	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$pull": bson.M{"offNotificationsTopics": t}})
}

// DisableNotificationsTopic add topic to user list offNotificationsTopics
func DisableNotificationsTopic(user *tat.User, topic string) error {
	if containsOffNotificationsTopic(user, topic) {
		return fmt.Errorf("DisableNotificationsTopic not possible, notifications are already off on topic %s", topic)
	}

	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$push": bson.M{"offNotificationsTopics": topic}})
}

// EnableNotificationsAllTopics removes all topics from user list offNotificationsTopics
func EnableNotificationsAllTopics(user *tat.User) error {
	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"offNotificationsTopics": []bson.M{}}})
}

// DisableNotificationsAllTopics add all topics to user list offNotificationsTopics, except /Private/*
func DisableNotificationsAllTopics(user *tat.User) error {
	criteria := &tat.TopicCriteria{
		Skip:  0,
		Limit: 9000000,
	}
	_, topics, err := topic.ListTopics(criteria, user, false, false, false)
	if err != nil {
		return err
	}

	topicsToSet := []string{}
	for _, topic := range topics {
		if !strings.HasPrefix(topic.Topic, "/Private") {
			topicsToSet = append(topicsToSet, topic.Topic)
		}
	}

	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"offNotificationsTopics": topicsToSet}})
}

func getFavoriteTag(user *tat.User, tag string) (string, error) {
	for _, cur := range user.FavoritesTags {
		if cur == tag {
			return cur, nil
		}
	}
	l := ""
	return l, fmt.Errorf("topic %s not found in favorites tags of user", tag)
}

func containsFavoriteTag(user *tat.User, tag string) bool {
	_, err := getFavoriteTag(user, tag)
	if err == nil {
		return true
	}
	return false
}

// AddFavoriteTag Add a favorite tag to user
func AddFavoriteTag(user *tat.User, tag string) error {
	if containsFavoriteTag(user, tag) {
		return fmt.Errorf("AddFavoriteTag not possible, %s is already a favorite tag", tag)
	}
	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$push": bson.M{"favoritesTags": tag}})
}

// RemoveFavoriteTag remove a favorite tag from user
func RemoveFavoriteTag(user *tat.User, tag string) error {
	t, err := getFavoriteTag(user, tag)
	if err != nil {
		return fmt.Errorf("Remove favorite tag is not possible, %s is not a favorite of this user", tag)
	}

	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$pull": bson.M{"favoritesTags": t}})
}

func getContact(user *tat.User, contactUsername string) (tat.Contact, error) {
	for _, cur := range user.Contacts {
		if cur.Username == contactUsername {
			return cur, nil
		}
	}
	l := tat.Contact{}
	return l, fmt.Errorf("contact %s not found", contactUsername)
}

func containsContact(user *tat.User, contactUsername string) bool {
	_, err := getContact(user, contactUsername)
	if err == nil {
		return true
	}
	return false
}

// AddContact add a contact to user
func AddContact(user *tat.User, contactUsername string, contactFullname string) error {
	if containsContact(user, contactUsername) {
		return fmt.Errorf("AddContact not possible, %s is already a contact of this user", contactUsername)
	}
	var newContact = &tat.Contact{Username: contactUsername, Fullname: contactFullname}

	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$push": bson.M{"contacts": newContact}})
}

// RemoveContact removes a contact from user
func RemoveContact(user *tat.User, contactUsername string) error {
	l, err := getContact(user, contactUsername)
	if err != nil {
		return fmt.Errorf("Remove Contact is not possible, %s is not a contact of this user", contactUsername)
	}

	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$pull": bson.M{"contacts": l}})
}

// ConvertToSystem set attribute IsSysetm to true and suffix mail with a random string. If
// canWriteNotifications is true, this system user can write into /Private/username/Notifications topics
// canListUsersAsAdmin is true, this system user can view all user's fields as an admin (email, etc...)
// returns password, err
func ConvertToSystem(user *tat.User, userAdmin string, canWriteNotifications, canListUsersAsAdmin bool) (string, error) {
	email := fmt.Sprintf("%s$system$by$%s$%d", user.Email, userAdmin, time.Now().Unix())
	err := store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"email":                 email,
			"isSystem":              true,
			"canWriteNotifications": canWriteNotifications,
			"canListUsersAsAdmin":   canListUsersAsAdmin,
			"auth.emailVerified":    true,
		}})

	if err != nil {
		return "", err
	}

	return regenerateAndStoreAuth(user)
}

// UpdateSystemUser updates flags CanWriteNotifications and CanListUsersAsAdmin
func UpdateSystemUser(user *tat.User, canWriteNotifications, canListUsersAsAdmin bool) error {
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"canWriteNotifications": canWriteNotifications,
			"canListUsersAsAdmin":   canListUsersAsAdmin,
		}})
}

// ResetSystemUserPassword reset a password for a system user
// returns newPassword
func ResetSystemUserPassword(user *tat.User) (string, error) {
	if !user.IsSystem {
		return "", fmt.Errorf("Reset password not possible, %s is not a system user", user.Username)
	}
	return regenerateAndStoreAuth(user)
}

// ConvertToAdmin set attribute IsAdmin to true
func ConvertToAdmin(user *tat.User, userAdmin string) error {
	log.Warnf("%s grant %s to admin", userAdmin, user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"isAdmin": true}})
}

// Archive changes username of one user and set attribute email, username to random string
func Archive(user *tat.User, userAdmin string) error {
	newFullname := fmt.Sprintf("%s$archive$by$%s$%d", user.Fullname, userAdmin, time.Now().Unix())
	newUsername := fmt.Sprintf("%s$archive$by$%s$%d", user.Username, userAdmin, time.Now().Unix())
	email := fmt.Sprintf("%s$archive$by$%s$%d", user.Email, userAdmin, time.Now().Unix())
	err := store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"email": email, "fullname": newFullname, "isArchived": true}})

	if err != nil {
		return err
	}
	return Rename(user, newUsername)
}

// Rename changes username of one user
func Rename(user *tat.User, newUsername string) error {
	var userCheck = tat.User{}
	found, errCheck := FindByUsername(&userCheck, newUsername)

	if errCheck != nil {
		return fmt.Errorf("Rename> Error with DB Backend:%s", errCheck)
	} else if found {
		return fmt.Errorf("Rename> Username %s already exists", newUsername)
	}

	err := store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"username": newUsername}})

	if err != nil {
		return err
	}

	topic.ChangeUsernameOnTopics(user.Username, newUsername)
	group.ChangeUsernameOnGroups(user.Username, newUsername)
	presence.ChangeAuthorUsernameOnPresences(user.Username, newUsername)
	cache.CleanUsernames(user.Username)
	return nil
}

// Update changes fullname and email of user
func Update(user *tat.User, newFullname, newEmail string) error {

	userCheck := tat.User{}
	found, err := FindByEmail(&userCheck, newEmail)
	if err != nil {
		return err
	}
	if user.Email != newEmail && found {
		return fmt.Errorf("Email %s already exists", newEmail)
	}

	found2, err2 := FindByFullname(&userCheck, newFullname)
	if err2 != nil {
		return err2
	}
	if user.Fullname != newFullname && found2 {
		return fmt.Errorf("Fullname %s already exists", newFullname)
	}

	cache.CleanUsernames(user.Username)
	return store.Tat().CUsers.Update(
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"fullname": newFullname, "email": newEmail}})
}

// CountUsers returns the total number of users in db
func CountUsers() (int, error) {
	return store.Tat().CUsers.Count()
}

// CreatePrivateTopic creates a Private Topic. Name of topic will be :
// /Private/username and if subTopic != "", it will be :
// /Private/username/subTopic
// CanUpdateMsg, CanDeleteMsg set to true
func CreatePrivateTopic(user *tat.User, subTopic string) error {
	topicName := "/Private/" + user.Username
	description := "Private Topic"

	if subTopic != "" {
		topicName = fmt.Sprintf("%s/%s", topicName, subTopic)
		description = fmt.Sprintf("%s - %s of %s", description, subTopic, user.Username)
	} else {
		description = fmt.Sprintf("%s - %s", description, user.Username)
	}
	t := &tat.Topic{
		Topic:        topicName,
		Description:  description,
		CanUpdateMsg: true,
		CanDeleteMsg: true,
	}
	e := topic.Insert(t, user)
	if e != nil {
		log.Errorf("Error while creating Private topic %s: %s", topicName, e.Error())
	}
	return e
}

// AddDefaultGroup add default group to user
func AddDefaultGroup(user *tat.User) error {
	groupname := viper.GetString("default_group")

	// no default group
	if groupname == "" {
		return nil
	}

	tatGroup, errfinding := group.FindByName(groupname)
	if errfinding != nil {
		e := fmt.Errorf("Error while fetching default group : %s", errfinding.Error())
		return e
	}
	err := group.AddUser(tatGroup, "Tat", user.Username)
	if err != nil {
		e := fmt.Errorf("Error while adding user to default group : %s", err.Error())
		return e
	}
	return nil
}

// CheckDefaultGroup check default group and creates it if fixDefaultGroup is true
func CheckDefaultGroup(user *tat.User, fixDefaultGroup bool) string {
	defaultGroupInfo := ""

	userGroups, err := group.GetUserGroupsOnlyName(user.Username)
	if err != nil {
		return "Error while fetching user groups"
	}

	find := false
	for _, g := range userGroups {
		if g == viper.GetString("default_group") {
			find = true
			defaultGroupInfo = fmt.Sprintf("user in %s OK", viper.GetString("default_group"))
			break
		}
	}
	if !find {
		if fixDefaultGroup {
			if err = AddDefaultGroup(user); err != nil {
				return err.Error()
			}
			defaultGroupInfo = fmt.Sprintf("user added in default group %s", viper.GetString("default_group"))
		} else {
			defaultGroupInfo = fmt.Sprintf("user in default group %s KO", viper.GetString("default_group"))
		}
	}
	return defaultGroupInfo
}

// CheckTopics check default topics for user and creates them if fixTopics is true
func CheckTopics(user *tat.User, fixTopics bool) string {
	topicsInfo := ""
	topicNames := [...]string{"", "Notifications"}
	for _, shortName := range topicNames {
		topicName := fmt.Sprintf("/Private/%s", user.Username)
		if shortName != "" {
			topicName = fmt.Sprintf("%s/%s", topicName, shortName)
		}

		if _, errfinding := topic.FindByTopic(topicName, false, false, false, nil); errfinding != nil {
			topicsInfo = fmt.Sprintf("%s %s KO : not exist; ", topicsInfo, topicName)
			if fixTopics {
				if err := CreatePrivateTopic(user, shortName); err != nil {
					topicsInfo = fmt.Sprintf("%s Error while creating %s; ", topicsInfo, topicName)
				} else {
					topicsInfo = fmt.Sprintf("%s %s created; ", topicsInfo, topicName)
				}
			}
		} else {
			topicsInfo = fmt.Sprintf("%s %s OK; ", topicsInfo, topicName)
		}
	}
	return topicsInfo
}
