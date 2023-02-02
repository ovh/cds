package sdk

import "time"

type UserLink struct {
	ID                 string    `json:"id" db:"id"`
	AuthentifiedUserID string    `json:"authentified_user_id" db:"authentified_user_id"`
	Type               string    `json:"type" db:"type" cli:"type"`
	Username           string    `json:"username" db:"username" cli:"username"`
	Created            time.Time `json:"created" db:"created"`
}
