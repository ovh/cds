package vault

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/logical"
)

const (
	// The location of the key used to generate response-wrapping JWTs
	coreWrappingJWTKeyPath = "core/wrapping/jwtkey"
)

func (c *Core) ensureWrappingKey() error {
	entry, err := c.barrier.Get(coreWrappingJWTKeyPath)
	if err != nil {
		return err
	}

	var keyParams clusterKeyParams

	if entry == nil {
		key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		if err != nil {
			return errwrap.Wrapf("failed to generate wrapping key: {{err}}", err)
		}
		keyParams.D = key.D
		keyParams.X = key.X
		keyParams.Y = key.Y
		keyParams.Type = corePrivateKeyTypeP521
		val, err := jsonutil.EncodeJSON(keyParams)
		if err != nil {
			return errwrap.Wrapf("failed to encode wrapping key: {{err}}", err)
		}
		entry = &Entry{
			Key:   coreWrappingJWTKeyPath,
			Value: val,
		}
		if err = c.barrier.Put(entry); err != nil {
			return errwrap.Wrapf("failed to store wrapping key: {{err}}", err)
		}
	}

	// Redundant if we just created it, but in this case serves as a check anyways
	if err = jsonutil.DecodeJSON(entry.Value, &keyParams); err != nil {
		return errwrap.Wrapf("failed to decode wrapping key parameters: {{err}}", err)
	}

	c.wrappingJWTKey = &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P521(),
			X:     keyParams.X,
			Y:     keyParams.Y,
		},
		D: keyParams.D,
	}

	c.logger.Info("core: loaded wrapping token key")

	return nil
}

func (c *Core) wrapInCubbyhole(req *logical.Request, resp *logical.Response) (*logical.Response, error) {
	// Before wrapping, obey special rules for listing: if no entries are
	// found, 404. This prevents unwrapping only to find empty data.
	if req.Operation == logical.ListOperation {
		if resp == nil || len(resp.Data) == 0 {
			return nil, logical.ErrUnsupportedPath
		}
		keysRaw, ok := resp.Data["keys"]
		if !ok || keysRaw == nil {
			return nil, logical.ErrUnsupportedPath
		}
		keys, ok := keysRaw.([]string)
		if !ok {
			return nil, logical.ErrUnsupportedPath
		}
		if len(keys) == 0 {
			return nil, logical.ErrUnsupportedPath
		}
	}

	var err error

	// If we are wrapping, the first part (performed in this functions) happens
	// before auditing so that resp.WrapInfo.Token can contain the HMAC'd
	// wrapping token ID in the audit logs, so that it can be determined from
	// the audit logs whether the token was ever actually used.
	creationTime := time.Now()
	te := TokenEntry{
		Path:           req.Path,
		Policies:       []string{"response-wrapping"},
		CreationTime:   creationTime.Unix(),
		TTL:            resp.WrapInfo.TTL,
		NumUses:        1,
		ExplicitMaxTTL: resp.WrapInfo.TTL,
	}

	if err := c.tokenStore.create(&te); err != nil {
		c.logger.Error("core: failed to create wrapping token", "error", err)
		return nil, ErrInternalError
	}

	resp.WrapInfo.Token = te.ID
	resp.WrapInfo.CreationTime = creationTime

	// This will only be non-nil if this response contains a token, so in that
	// case put the accessor in the wrap info.
	if resp.Auth != nil {
		resp.WrapInfo.WrappedAccessor = resp.Auth.Accessor
	}

	switch resp.WrapInfo.Format {
	case "jwt":
		// Create the JWT
		claims := jws.Claims{}
		// Map the JWT ID to the token ID for ease ofuse
		claims.SetJWTID(te.ID)
		// Set the issue time to the creation time
		claims.SetIssuedAt(creationTime)
		// Set the expiration to the TTL
		claims.SetExpiration(creationTime.Add(resp.WrapInfo.TTL))
		if resp.Auth != nil {
			claims.Set("accessor", resp.Auth.Accessor)
		}
		claims.Set("type", "wrapping")
		claims.Set("addr", c.redirectAddr)
		jwt := jws.NewJWT(claims, crypto.SigningMethodES512)
		serWebToken, err := jwt.Serialize(c.wrappingJWTKey)
		if err != nil {
			c.logger.Error("core: failed to serialize JWT", "error", err)
			return nil, ErrInternalError
		}
		resp.WrapInfo.Token = string(serWebToken)
	}

	cubbyReq := &logical.Request{
		Operation:   logical.CreateOperation,
		Path:        "cubbyhole/response",
		ClientToken: te.ID,
	}

	// During a rewrap, store the original response, don't wrap it again.
	if req.Path == "sys/wrapping/rewrap" {
		cubbyReq.Data = map[string]interface{}{
			"response": resp.Data["response"],
		}
	} else {
		httpResponse := logical.LogicalResponseToHTTPResponse(resp)

		// Add the unique identifier of the original request to the response
		httpResponse.RequestID = req.ID

		// Because of the way that JSON encodes (likely just in Go) we actually get
		// mixed-up values for ints if we simply put this object in the response
		// and encode the whole thing; so instead we marshal it first, then store
		// the string response. This actually ends up making it easier on the
		// client side, too, as it becomes a straight read-string-pass-to-unmarshal
		// operation.

		marshaledResponse, err := json.Marshal(httpResponse)
		if err != nil {
			c.logger.Error("core: failed to marshal wrapped response", "error", err)
			return nil, ErrInternalError
		}

		cubbyReq.Data = map[string]interface{}{
			"response": string(marshaledResponse),
		}
	}

	cubbyResp, err := c.router.Route(cubbyReq)
	if err != nil {
		// Revoke since it's not yet being tracked for expiration
		c.tokenStore.Revoke(te.ID)
		c.logger.Error("core: failed to store wrapped response information", "error", err)
		return nil, ErrInternalError
	}
	if cubbyResp != nil && cubbyResp.IsError() {
		c.tokenStore.Revoke(te.ID)
		c.logger.Error("core: failed to store wrapped response information", "error", cubbyResp.Data["error"])
		return cubbyResp, nil
	}

	// Store info for lookup
	cubbyReq.Path = "cubbyhole/wrapinfo"
	cubbyReq.Data = map[string]interface{}{
		"creation_ttl":  resp.WrapInfo.TTL,
		"creation_time": creationTime,
	}
	cubbyResp, err = c.router.Route(cubbyReq)
	if err != nil {
		// Revoke since it's not yet being tracked for expiration
		c.tokenStore.Revoke(te.ID)
		c.logger.Error("core: failed to store wrapping information", "error", err)
		return nil, ErrInternalError
	}
	if cubbyResp != nil && cubbyResp.IsError() {
		c.tokenStore.Revoke(te.ID)
		c.logger.Error("core: failed to store wrapping information", "error", cubbyResp.Data["error"])
		return cubbyResp, nil
	}

	auth := &logical.Auth{
		ClientToken: te.ID,
		Policies:    []string{"response-wrapping"},
		LeaseOptions: logical.LeaseOptions{
			TTL:       te.TTL,
			Renewable: false,
		},
	}

	// Register the wrapped token with the expiration manager
	if err := c.expiration.RegisterAuth(te.Path, auth); err != nil {
		// Revoke since it's not yet being tracked for expiration
		c.tokenStore.Revoke(te.ID)
		c.logger.Error("core: failed to register cubbyhole wrapping token lease", "request_path", req.Path, "error", err)
		return nil, ErrInternalError
	}

	return nil, nil
}

func (c *Core) ValidateWrappingToken(req *logical.Request) (bool, error) {
	if req == nil {
		return false, fmt.Errorf("invalid request")
	}

	var err error

	var token string
	var thirdParty bool
	if req.Data != nil && req.Data["token"] != nil {
		thirdParty = true
		if tokenStr, ok := req.Data["token"].(string); !ok {
			return false, fmt.Errorf("could not decode token in request body")
		} else if tokenStr == "" {
			return false, fmt.Errorf("empty token in request body")
		} else {
			token = tokenStr
		}
	} else {
		token = req.ClientToken
	}

	// Check for it being a JWT. If it is, and it is valid, we extract the
	// internal client token from it and use that during lookup.
	if strings.Count(token, ".") == 2 {
		wt, err := jws.ParseJWT([]byte(token))
		// If there's an error we simply fall back to attempting to use it as a regular token
		if err == nil && wt != nil {
			validator := &jwt.Validator{}
			validator.SetClaim("type", "wrapping")
			if err = wt.Validate(&c.wrappingJWTKey.PublicKey, crypto.SigningMethodES512, []*jwt.Validator{validator}...); err != nil {
				return false, errwrap.Wrapf("wrapping token signature could not be validated: {{err}}", err)
			}
			token, _ = wt.Claims().JWTID()
			// We override the given request client token so that the rest of
			// Vault sees the real value. This also ensures audit logs are
			// consistent with the actual token that was issued.
			if !thirdParty {
				req.ClientToken = token
			} else {
				req.Data["token"] = token
			}
		}
	}

	if token == "" {
		return false, fmt.Errorf("token is empty")
	}

	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	if c.sealed {
		return false, ErrSealed
	}
	if c.standby {
		return false, ErrStandby
	}

	te, err := c.tokenStore.Lookup(token)
	if err != nil {
		return false, err
	}
	if te == nil {
		return false, nil
	}

	if len(te.Policies) != 1 {
		return false, nil
	}

	if te.Policies[0] != responseWrappingPolicyName {
		return false, nil
	}

	return true, nil
}
