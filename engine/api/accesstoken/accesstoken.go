package accesstoken

import (
	"crypto/rsa"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	localIssuer string
	signingKey  *rsa.PrivateKey
	verifyKey   *rsa.PublicKey
)

// Init the package y passing the signing key
func Init(issuer string, k []byte) error {
	localIssuer = issuer
	var err error
	signingKey, err = jwt.ParseRSAPrivateKeyFromPEM(k)
	if err != nil {
		return sdk.WithStack(err)
	}
	verifyKey = &signingKey.PublicKey
	return nil
}

// New returns a new access token for a user
func New(u sdk.User, groups []sdk.Group, origin, desc string, expiration *time.Time) (sdk.AccessToken, string, error) {
	var token sdk.AccessToken
	token.ID = sdk.UUID()
	token.Created = time.Now()
	token.ExpireAt = expiration
	token.Description = desc
	token.Origin = origin
	token.Status = sdk.AccessTokenStatusEnabled
	token.Groups = groups

	var tmpUser = u
	tmpUser.Auth = sdk.Auth{}
	tmpUser.Favorites = nil
	tmpUser.Groups = nil
	tmpUser.Permissions = sdk.UserPermissions{}
	token.User = tmpUser
	token.UserID = u.ID

	jwttoken, err := Regen(&token)
	if err != nil {
		return token, jwttoken, sdk.WithStack(err)
	}

	return token, jwttoken, nil
}

// Regen regenerate the signed token value
func Regen(token *sdk.AccessToken) (string, error) {
	claims := sdk.AccessTokenJWTClaims{
		ID:     token.ID,
		Groups: sdk.GroupsToIDs(token.Groups),
		StandardClaims: jwt.StandardClaims{
			Issuer:   localIssuer,
			Subject:  token.User.Username,
			Id:       token.ID,
			IssuedAt: time.Now().Unix(),
		},
	}

	if token.ExpireAt != nil {
		claims.ExpiresAt = token.ExpireAt.Unix()
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	ss, err := jwtToken.SignedString(signingKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}

	return ss, nil
}

// IsValid checks a jwt token against all access_token
func IsValid(db gorp.SqlExecutor, jwtToken string) (bool, error) {
	token, err := verifyToken(jwtToken)
	if err != nil {
		return false, sdk.WrapError(err, "invalid token")
	}

	claims := token.Claims.(*sdk.AccessTokenJWTClaims)
	id := claims.StandardClaims.Id

	// Load the access token from the id read in the claim
	accessToken, err := FindByID(db, id)
	if err != nil {
		log.Error("accesstoken.IsValid> unable find access token %s: %v", id, err)
		return false, sdk.ErrUnauthorized
	}

	// Check groups from the claims againts the groups in the database
	ids := sdk.GroupsToIDs(accessToken.Groups)
	for _, groupID := range claims.Groups {
		if !sdk.IsInInt64Array(groupID, ids) {
			log.Error("accesstoken.IsValid> token %s is invalid (group mismatch): %v", id, err)
			return false, nil
		}
	}

	return token != nil, err
}

func verifyToken(jwtToken string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(jwtToken, &sdk.AccessTokenJWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "Unexpected signing method: %v", token.Header["alg"])
			}
			return verifyKey, nil
		})

	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if claims, ok := token.Claims.(*sdk.AccessTokenJWTClaims); ok && token.Valid {
		log.Debug("Token isValid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		return nil, sdk.ErrUnauthorized
	}

	return token, nil
}
