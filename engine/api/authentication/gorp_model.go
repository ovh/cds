package authentication

import (
	"time"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type AuthConsumerOld struct {
	ID                           string                          `json:"id" cli:"id,key" db:"id"`
	Name                         string                          `json:"name" cli:"name" db:"name"`
	Type                         sdk.AuthConsumerType            `json:"type" cli:"type" db:"type"`
	Description                  string                          `json:"description" cli:"description" db:"description"`
	ParentID                     *string                         `json:"parent_id,omitempty" db:"parent_id"`
	Created                      time.Time                       `json:"created" cli:"created" db:"created"`
	DeprecatedIssuedAt           time.Time                       `json:"issued_at" cli:"issued_at" db:"issued_at"`
	Disabled                     bool                            `json:"disabled" cli:"disabled" db:"disabled"`
	Warnings                     sdk.AuthConsumerWarnings        `json:"warnings,omitempty" db:"warnings"`
	LastAuthentication           *time.Time                      `json:"last_authentication,omitempty" db:"last_authentication"`
	ValidityPeriods              sdk.AuthConsumerValidityPeriods `json:"validity_periods,omitempty" db:"validity_periods"`
	AuthentifiedUserID           string                          `json:"user_id,omitempty" db:"user_id"`
	Data                         sdk.AuthConsumerData            `json:"-" db:"data"` // NEVER returns auth consumer data in json, TODO this fields should be visible only in auth package
	GroupIDs                     sdk.Int64Slice                  `json:"group_ids,omitempty" cli:"group_ids" db:"group_ids"`
	InvalidGroupIDs              sdk.Int64Slice                  `json:"invalid_group_ids,omitempty" db:"invalid_group_ids"`
	ScopeDetails                 sdk.AuthConsumerScopeDetails    `json:"scope_details,omitempty" cli:"scope_details" db:"scope_details"`
	ServiceName                  *string                         `json:"service_name,omitempty" db:"service_name"`
	ServiceType                  *string                         `json:"service_type,omitempty" db:"service_type"`
	ServiceRegion                *string                         `json:"service_region,omitempty" db:"service_region"`
	ServiceIgnoreJobWithNoRegion *bool                           `json:"service_ignore_job_with_no_region,omitempty" db:"service_ignore_job_with_no_region"`

	gorpmapper.SignedEntity
}

func (c AuthConsumerOld) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{c.ID, c.AuthentifiedUserID, c.Type, c.Data, c.Created, c.GroupIDs, c.ScopeDetails, c.Disabled, c.ServiceName, c.ServiceType, c.ServiceRegion, c.ServiceIgnoreJobWithNoRegion} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.AuthentifiedUserID}}{{printf .Type}}{{printf .Data}}{{printDate .Created}}{{printf .GroupIDs}}{{printf .ScopeDetails}}{{printf .Disabled}}",
		"{{.ID}}{{.AuthentifiedUserID}}{{print .Type}}{{print .Data}}{{printDate .Created}}{{print .GroupIDs}}{{print .ScopeDetails}}{{print .Disabled}}",
	}
}

type authConsumer struct {
	sdk.AuthConsumer
	gorpmapper.SignedEntity
}

func (c authConsumer) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{c.ID, c.Type, c.Created, c.Disabled, c.Name} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{printf .Type}}{{printDate .Created}}{{printf .Disabled}}{{.Name}}",
		"{{.ID}}{{print .Type}}{{printDate .Created}}{{print .Disabled}}",
	}
}

type authConsumerUserData struct {
	sdk.AuthUserConsumerData
	gorpmapper.SignedEntity
}

func (c authConsumerUserData) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{c.ID, c.AuthConsumerID, c.AuthentifiedUserID, c.Data, c.GroupIDs, c.ScopeDetails} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.AuthConsumerID}}{{.AuthentifiedUserID}}{{printf .Data}}{{printf .GroupIDs}}{{printf .ScopeDetails}}",
		"{{.ID}}{{.AuthConsumerID}}{{.AuthentifiedUserID}}{{print .Data}}{{print .GroupIDs}}{{print .ScopeDetails}}",
	}
}

type authConsumerHatcheryData struct {
	sdk.AuthConsumerHatcheryData
	gorpmapper.SignedEntity
}

func (c authConsumerHatcheryData) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{c.ID, c.AuthConsumerID, c.HatcheryID} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.AuthConsumerID}}{{.HatcheryID}}",
	}
}

type authSession struct {
	sdk.AuthSession
	gorpmapper.SignedEntity
}

func (s authSession) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{s.ID, s.ConsumerID, s.ExpireAt, s.Created, s.MFA} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.ConsumerID}}{{printDate .ExpireAt}}{{printDate .Created}}{{printf .MFA}}",
		"{{.ID}}{{.ConsumerID}}{{printDate .ExpireAt}}{{printDate .Created}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(authConsumer{}, "auth_consumer", false, "id"),
		gorpmapping.New(authSession{}, "auth_session", false, "id"),
		gorpmapping.New(authConsumerUserData{}, "auth_consumer_user", false, "id"),
		gorpmapping.New(AuthConsumerOld{}, "auth_consumer_old", false, "id"),
		gorpmapping.New(authConsumerHatcheryData{}, "auth_consumer_hatchery", false, "id"),
	)
}
