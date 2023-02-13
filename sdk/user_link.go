package sdk

import "time"

type UserLink struct {
	ID                 string    `json:"id" db:"id"`
	AuthentifiedUserID string    `json:"authentified_user_id" db:"authentified_user_id"`
	Type               string    `json:"type" db:"type"`
	ExternalID         string    `json:"external_id" db:"external_id"`
	Username           string    `json:"username" db:"username"`
	Created            time.Time `json:"created" db:"created"`
}
