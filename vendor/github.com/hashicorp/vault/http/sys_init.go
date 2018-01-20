package http

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/vault"
)

func handleSysInit(core *vault.Core) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleSysInitGet(core, w, r)
		case "PUT", "POST":
			handleSysInitPut(core, w, r)
		default:
			respondError(w, http.StatusMethodNotAllowed, nil)
		}
	})
}

func handleSysInitGet(core *vault.Core, w http.ResponseWriter, r *http.Request) {
	init, err := core.Initialized()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondOk(w, &InitStatusResponse{
		Initialized: init,
	})
}

func handleSysInitPut(core *vault.Core, w http.ResponseWriter, r *http.Request) {
	// Parse the request
	var req InitRequest
	if err := parseRequest(r, w, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	// Initialize
	barrierConfig := &vault.SealConfig{
		SecretShares:    req.SecretShares,
		SecretThreshold: req.SecretThreshold,
		StoredShares:    req.StoredShares,
		PGPKeys:         req.PGPKeys,
	}

	recoveryConfig := &vault.SealConfig{
		SecretShares:    req.RecoveryShares,
		SecretThreshold: req.RecoveryThreshold,
		PGPKeys:         req.RecoveryPGPKeys,
	}

	if core.SealAccess().StoredKeysSupported() {
		if barrierConfig.SecretShares != 1 {
			respondError(w, http.StatusBadRequest, fmt.Errorf("secret shares must be 1"))
			return
		}
		if barrierConfig.SecretThreshold != barrierConfig.SecretShares {
			respondError(w, http.StatusBadRequest, fmt.Errorf("secret threshold must be same as secret shares"))
			return
		}
		if barrierConfig.StoredShares != barrierConfig.SecretShares {
			respondError(w, http.StatusBadRequest, fmt.Errorf("stored shares must be same as secret shares"))
			return
		}
		if barrierConfig.PGPKeys != nil && len(barrierConfig.PGPKeys) > 0 {
			respondError(w, http.StatusBadRequest, fmt.Errorf("PGP keys not supported when storing shares"))
			return
		}
	} else {
		if barrierConfig.StoredShares > 0 {
			respondError(w, http.StatusBadRequest, fmt.Errorf("stored keys are not supported"))
			return
		}
	}

	if len(barrierConfig.PGPKeys) > 0 && len(barrierConfig.PGPKeys) != barrierConfig.SecretShares-barrierConfig.StoredShares {
		respondError(w, http.StatusBadRequest, fmt.Errorf("incorrect number of PGP keys"))
		return
	}

	if core.SealAccess().RecoveryKeySupported() {
		if len(recoveryConfig.PGPKeys) > 0 && len(recoveryConfig.PGPKeys) != recoveryConfig.SecretShares-recoveryConfig.StoredShares {
			respondError(w, http.StatusBadRequest, fmt.Errorf("incorrect number of PGP keys for recovery"))
			return
		}
	}

	initParams := &vault.InitParams{
		BarrierConfig:   barrierConfig,
		RecoveryConfig:  recoveryConfig,
		RootTokenPGPKey: req.RootTokenPGPKey,
	}

	result, initErr := core.Initialize(initParams)
	if initErr != nil {
		if !errwrap.ContainsType(initErr, new(vault.NonFatalError)) {
			respondError(w, http.StatusBadRequest, initErr)
			return
		} else {
			// Add a warnings field? The error will be logged in the vault log
			// already.
		}
	}

	// Encode the keys
	keys := make([]string, 0, len(result.SecretShares))
	keysB64 := make([]string, 0, len(result.SecretShares))
	for _, k := range result.SecretShares {
		keys = append(keys, hex.EncodeToString(k))
		keysB64 = append(keysB64, base64.StdEncoding.EncodeToString(k))
	}

	resp := &InitResponse{
		Keys:      keys,
		KeysB64:   keysB64,
		RootToken: result.RootToken,
	}

	if len(result.RecoveryShares) > 0 {
		resp.RecoveryKeys = make([]string, 0, len(result.RecoveryShares))
		resp.RecoveryKeysB64 = make([]string, 0, len(result.RecoveryShares))
		for _, k := range result.RecoveryShares {
			resp.RecoveryKeys = append(resp.RecoveryKeys, hex.EncodeToString(k))
			resp.RecoveryKeysB64 = append(resp.RecoveryKeysB64, base64.StdEncoding.EncodeToString(k))
		}
	}

	core.UnsealWithStoredKeys()

	respondOk(w, resp)
}

type InitRequest struct {
	SecretShares      int      `json:"secret_shares"`
	SecretThreshold   int      `json:"secret_threshold"`
	StoredShares      int      `json:"stored_shares"`
	PGPKeys           []string `json:"pgp_keys"`
	RecoveryShares    int      `json:"recovery_shares"`
	RecoveryThreshold int      `json:"recovery_threshold"`
	RecoveryPGPKeys   []string `json:"recovery_pgp_keys"`
	RootTokenPGPKey   string   `json:"root_token_pgp_key"`
}

type InitResponse struct {
	Keys            []string `json:"keys"`
	KeysB64         []string `json:"keys_base64"`
	RecoveryKeys    []string `json:"recovery_keys,omitempty"`
	RecoveryKeysB64 []string `json:"recovery_keys_base64,omitempty"`
	RootToken       string   `json:"root_token"`
}

type InitStatusResponse struct {
	Initialized bool `json:"initialized"`
}
