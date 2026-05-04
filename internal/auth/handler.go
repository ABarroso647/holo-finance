package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"holo/internal/components"
	db "holo/internal/db/generated"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

const (
	sessionName  = "holo-session"
	waSessionKey = "wa-session"
	authKey      = "authenticated"
	redirectKey  = "redirect-after-login"
)

type Handler struct {
	wauthn  *webauthn.WebAuthn
	queries *db.Queries
	store   sessions.Store
}

// WEBAUTHN_RPID defaults to "localhost", WEBAUTHN_RPORIGIN defaults to "http://localhost:8080".
func New(queries *db.Queries, store sessions.Store) (*Handler, error) {
	rpid := os.Getenv("WEBAUTHN_RPID")
	if rpid == "" {
		rpid = "localhost"
	}
	rporigin := os.Getenv("WEBAUTHN_RPORIGIN")
	if rporigin == "" {
		rporigin = "http://localhost:8080"
	}

	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Holo",
		RPID:          rpid,
		RPOrigins:     []string{rporigin},
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn init: %w", err)
	}

	return &Handler{wauthn: w, queries: queries, store: store}, nil
}

// holoUser implements webauthn.User for the single app user.
type holoUser struct {
	creds []webauthn.Credential
}

func (u *holoUser) WebAuthnID() []byte                        { return []byte("holo-user") }
func (u *holoUser) WebAuthnName() string                      { return "holo" }
func (u *holoUser) WebAuthnDisplayName() string               { return "Holo" }
func (u *holoUser) WebAuthnIcon() string                      { return "" }
func (u *holoUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func (h *Handler) loadUser(r *http.Request) (*holoUser, error) {
	rows, err := h.queries.ListWebAuthnCredentials(r.Context())
	if err != nil {
		return nil, err
	}
	user := &holoUser{}
	for _, row := range rows {
		user.creds = append(user.creds, webauthn.Credential{
			ID:        row.CredentialID,
			PublicKey: row.PublicKey,
			Flags: webauthn.CredentialFlags{
				BackupEligible: row.BackupEligible == 1,
				BackupState:    row.BackupState == 1,
			},
			Authenticator: webauthn.Authenticator{
				SignCount: uint32(row.SignCount),
			},
		})
	}
	return user, nil
}

func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	count, _ := h.queries.CountWebAuthnCredentials(r.Context())
	if count > 0 {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	components.RegisterPage().Render(r.Context(), w)
}

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	count, _ := h.queries.CountWebAuthnCredentials(r.Context())
	if count == 0 {
		http.Redirect(w, r, "/auth/register", http.StatusFound)
		return
	}
	components.LoginPage().Render(r.Context(), w)
}

func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user := &holoUser{}
	creation, sessionData, err := h.wauthn.BeginRegistration(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("begin registration: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.saveWASession(w, r, sessionData); err != nil {
		http.Error(w, "save session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creation)
}

func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	sessionData, err := h.loadWASession(r)
	if err != nil {
		http.Error(w, "session expired — try again", http.StatusBadRequest)
		return
	}

	user := &holoUser{}
	credential, err := h.wauthn.FinishRegistration(user, *sessionData, r)
	if err != nil {
		log.Printf("auth: finish registration: %v", err)
		http.Error(w, fmt.Sprintf("finish registration: %v", err), http.StatusBadRequest)
		return
	}

	be := int64(0)
	if credential.Flags.BackupEligible {
		be = 1
	}
	bs := int64(0)
	if credential.Flags.BackupState {
		bs = 1
	}
	if err := h.queries.CreateWebAuthnCredential(r.Context(), db.CreateWebAuthnCredentialParams{
		ID:             uuid.New().String(),
		CredentialID:   credential.ID,
		PublicKey:      credential.PublicKey,
		SignCount:      int64(credential.Authenticator.SignCount),
		BackupEligible: be,
		BackupState:    bs,
	}); err != nil {
		http.Error(w, fmt.Sprintf("store credential: %v", err), http.StatusInternalServerError)
		return
	}

	h.setAuthenticated(w, r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	user, err := h.loadUser(r)
	if err != nil || len(user.creds) == 0 {
		http.Error(w, "no credentials registered", http.StatusBadRequest)
		return
	}

	assertion, sessionData, err := h.wauthn.BeginLogin(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("begin login: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.saveWASession(w, r, sessionData); err != nil {
		http.Error(w, "save session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assertion)
}

func (h *Handler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	sessionData, err := h.loadWASession(r)
	if err != nil {
		http.Error(w, "session expired — try again", http.StatusBadRequest)
		return
	}

	user, err := h.loadUser(r)
	if err != nil {
		http.Error(w, "load credentials", http.StatusInternalServerError)
		return
	}

	credential, err := h.wauthn.FinishLogin(user, *sessionData, r)
	if err != nil {
		log.Printf("auth: finish login: %v", err)
		http.Error(w, fmt.Sprintf("authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	be := int64(0)
	if credential.Flags.BackupEligible {
		be = 1
	}
	bs := int64(0)
	if credential.Flags.BackupState {
		bs = 1
	}
	h.queries.UpdateWebAuthnCredential(r.Context(), db.UpdateWebAuthnCredentialParams{ //nolint:errcheck
		SignCount:      int64(credential.Authenticator.SignCount),
		BackupEligible: be,
		BackupState:    bs,
		CredentialID:   credential.ID,
	})

	h.setAuthenticated(w, r)

	sess, _ := h.store.Get(r, sessionName)
	redirect := "/"
	if v, ok := sess.Values[redirectKey].(string); ok && v != "" {
		redirect = v
		delete(sess.Values, redirectKey)
		sess.Save(r, w) //nolint:errcheck
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "redirect": redirect})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.store.Get(r, sessionName)
	sess.Values[authKey] = false
	sess.Options.MaxAge = -1
	sess.Save(r, w) //nolint:errcheck
	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func RequireAuth(store sessions.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := store.Get(r, sessionName)
			if auth, ok := sess.Values[authKey].(bool); !ok || !auth {
				sess.Values[redirectKey] = r.URL.RequestURI()
				sess.Save(r, w) //nolint:errcheck
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (h *Handler) setAuthenticated(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.store.Get(r, sessionName)
	sess.Values[authKey] = true
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	sess.Save(r, w) //nolint:errcheck
}

func (h *Handler) saveWASession(w http.ResponseWriter, r *http.Request, data *webauthn.SessionData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	sess, _ := h.store.Get(r, sessionName)
	sess.Values[waSessionKey] = base64.StdEncoding.EncodeToString(raw)
	sess.Options = &sessions.Options{Path: "/", MaxAge: 300, HttpOnly: true, SameSite: http.SameSiteStrictMode}
	return sess.Save(r, w)
}

func (h *Handler) loadWASession(r *http.Request) (*webauthn.SessionData, error) {
	sess, err := h.store.Get(r, sessionName)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	encoded, ok := sess.Values[waSessionKey].(string)
	if !ok || encoded == "" {
		return nil, fmt.Errorf("no webauthn session")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &data, nil
}
