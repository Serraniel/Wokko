// Package app is Wokko's business logic. The web API and the kitshn layer both call it, so
// permissions and data safety rules are the same no matter which client is used.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/config"
	"github.com/serraniel/wokko/backend/internal/importer"
	"github.com/serraniel/wokko/backend/internal/media"
	"github.com/serraniel/wokko/backend/internal/store"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
	"github.com/serraniel/wokko/backend/internal/text"
)

// Error is an error safe to show to clients. Code is a stable key the UI translates, so
// messages can appear in any language.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func fail(status int, code, msg string) *Error { return &Error{status, code, msg} }

var (
	ErrNotFound     = fail(http.StatusNotFound, "not_found", "Not found")
	ErrForbidden    = fail(http.StatusForbidden, "forbidden", "You don't have permission to do this")
	ErrUnauthorized = fail(http.StatusUnauthorized, "unauthorized", "Please log in")
	ErrBadLogin     = fail(http.StatusUnauthorized, "bad_login", "Wrong handle or password")
	ErrConflict     = fail(http.StatusConflict, "edit_conflict", "Someone else changed this recipe")
)

func invalid(code, msg string) *Error { return fail(http.StatusBadRequest, code, msg) }

// Service holds everything the business logic needs.
type Service struct {
	Store    *store.Store
	Media    media.Store
	Cfg      config.Config
	Log      *slog.Logger
	passkeys passkeyFlows
	// Fetcher downloads pages for recipe imports.
	Fetcher *importer.Fetcher
	Now     func() time.Time

	backupMu sync.Mutex
	healthMu sync.RWMutex
	health   checkResult
}

func New(s *store.Store, cfg config.Config, log *slog.Logger) *Service {
	return &Service{Store: s, Media: media.Store{Dir: cfg.UploadsDir()}, Cfg: cfg, Log: log, Now: time.Now, Fetcher: importer.NewFetcher(cfg.Version, false)}
}

const timeFormat = "2006-01-02T15:04:05Z"

func (s *Service) stamp(t time.Time) string { return t.UTC().Format(timeFormat) }

// Viewer is who is making a request. User is nil for anonymous visitors.
type Viewer struct {
	User   *sqlcgen.User
	Locale string
}

func (v Viewer) id() int64 {
	if v.User == nil {
		return 0
	}
	return v.User.ID
}

func (v Viewer) isAdmin() bool { return v.User != nil && v.User.Role == "admin" }

func (v Viewer) locale() string {
	if v.Locale != "" {
		return v.Locale
	}
	return "en"
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

func toUser(u sqlcgen.User) User {
	return User{ID: u.ID, Handle: u.Handle, DisplayName: u.DisplayName, Role: u.Role, Locale: u.Locale, HouseholdID: u.HouseholdID, Disabled: u.DisabledAt.Valid, AssistantNewEntries: u.AssistantNewEntries}
}

func toPerson(u sqlcgen.User) Person {
	return Person{ID: u.ID, Handle: u.Handle, DisplayName: u.DisplayName}
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
func nullID(id int64) sql.NullInt64   { return sql.NullInt64{Int64: id, Valid: id > 0} }

var handleRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

func validateCredentials(handle, password string) error {
	if !handleRe.MatchString(handle) {
		return invalid("bad_handle", "Handles have 3 to 32 letters, numbers, dots, dashes or underscores")
	}
	return validatePassword(password)
}

func validatePassword(pw string) error {
	if len(pw) < 8 {
		return invalid("bad_password", "Passwords need at least 8 characters")
	}
	if len(pw) > 256 {
		return invalid("bad_password", "Passwords can have at most 256 characters")
	}
	return nil
}

// --- status, settings ---

func (s *Service) settingBool(ctx context.Context, key string) (bool, error) {
	v, err := s.Store.Queries.GetSetting(ctx, key)
	if err != nil {
		return false, fmt.Errorf("read setting %s: %w", key, err)
	}
	return v == "true", nil
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	n, err := s.Store.Queries.CountUsers(ctx)
	if err != nil {
		return Status{}, err
	}
	st, err := s.GetSettings(ctx)
	if err != nil {
		return Status{}, err
	}
	return Status{SetupComplete: n > 0, OpenRegistration: st.OpenRegistration, AllowPublic: st.AllowPublic, MCPEnabled: st.MCPEnabled, Version: s.Cfg.Version}, nil
}

func (s *Service) GetSettings(ctx context.Context) (Settings, error) {
	o, err := s.settingBool(ctx, "open_registration")
	if err != nil {
		return Settings{}, err
	}
	p, err := s.settingBool(ctx, "allow_public_recipes")
	if err != nil {
		return Settings{}, err
	}
	m, err := s.settingBool(ctx, "mcp_enabled")
	if err != nil {
		return Settings{}, err
	}
	return Settings{OpenRegistration: o, AllowPublic: p, MCPEnabled: m}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, v Viewer, in Settings) (Settings, error) {
	if !v.isAdmin() {
		return Settings{}, ErrForbidden
	}
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: "open_registration", Value: fmt.Sprint(in.OpenRegistration)}); err != nil {
			return err
		}
		if err := q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: "allow_public_recipes", Value: fmt.Sprint(in.AllowPublic)}); err != nil {
			return err
		}
		return q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: "mcp_enabled", Value: fmt.Sprint(in.MCPEnabled)})
	})
	if err != nil {
		return Settings{}, err
	}
	return in, nil
}

// --- accounts ---

const sessionLifetime = 60 * 24 * time.Hour

func (s *Service) createUser(ctx context.Context, q *sqlcgen.Queries, handle, display, password, role, locale string, household int64) (sqlcgen.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return sqlcgen.User{}, err
	}
	if !text.ValidLocale(locale) {
		locale = "en"
	}
	if household == 0 {
		h, err := q.CreateHousehold(ctx, handle)
		if err != nil {
			return sqlcgen.User{}, err
		}
		household = h.ID
	}
	if _, err := q.GetUserByHandle(ctx, handle); err == nil {
		return sqlcgen.User{}, fail(http.StatusConflict, "handle_taken", "This handle is taken")
	} else if !isNoRows(err) {
		return sqlcgen.User{}, err
	}
	return q.CreateUser(ctx, sqlcgen.CreateUserParams{Handle: handle, DisplayName: strings.TrimSpace(display), PasswordHash: hash, Role: role, Locale: locale, HouseholdID: household})
}

// Setup creates the first admin. It only works while no user exists.
func (s *Service) Setup(ctx context.Context, handle, display, password, locale string) (User, string, error) {
	if err := validateCredentials(handle, password); err != nil {
		return User{}, "", err
	}
	var u sqlcgen.User
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.CountUsers(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			return fail(http.StatusConflict, "already_setup", "Setup is already complete")
		}
		u, err = s.createUser(ctx, q, handle, display, password, "admin", locale, 0)
		return err
	})
	if err != nil {
		return User{}, "", err
	}
	token, err := s.newSession(ctx, u.ID)
	return toUser(u), token, err
}

// Register creates an account with a valid invite, or without one if registration is open.
func (s *Service) Register(ctx context.Context, handle, display, password, invite, locale string) (User, string, error) {
	if err := validateCredentials(handle, password); err != nil {
		return User{}, "", err
	}
	st, err := s.GetSettings(ctx)
	if err != nil {
		return User{}, "", err
	}
	if !st.OpenRegistration && invite == "" {
		return User{}, "", fail(http.StatusForbidden, "invite_required", "Registration is invite-only. Please enter your invite code")
	}
	var u sqlcgen.User
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		var household int64
		if invite != "" {
			inv, err := s.consumeInvite(ctx, q, invite)
			if err != nil {
				return err
			}
			household = inv.HouseholdID.Int64
		}
		var err error
		u, err = s.createUser(ctx, q, handle, display, password, "user", locale, household)
		return err
	})
	if err != nil {
		return User{}, "", err
	}
	token, err := s.newSession(ctx, u.ID)
	return toUser(u), token, err
}

func (s *Service) consumeInvite(ctx context.Context, q *sqlcgen.Queries, token string) (sqlcgen.Invite, error) {
	bad := fail(http.StatusForbidden, "invalid_invite", "This invite is invalid, expired or already used")
	inv, err := q.GetInviteByHash(ctx, auth.HashToken(strings.TrimSpace(token)))
	if isNoRows(err) {
		return inv, bad
	}
	if err != nil {
		return inv, err
	}
	if inv.ExpiresAt.Valid && inv.ExpiresAt.String <= s.stamp(s.Now()) {
		return inv, bad
	}
	n, err := q.UseInvite(ctx, inv.ID)
	if err != nil {
		return inv, err
	}
	if n == 0 {
		return inv, bad
	}
	return inv, nil
}

func (s *Service) newSession(ctx context.Context, userID int64) (string, error) {
	token, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		return q.CreateSession(ctx, sqlcgen.CreateSessionParams{UserID: userID, TokenHash: auth.HashToken(token), ExpiresAt: s.stamp(s.Now().Add(sessionLifetime))})
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

var dummyHash, _ = auth.HashPassword("timing protection for unknown handles")

// Login checks credentials and returns a new session token.
func (s *Service) Login(ctx context.Context, handle, password string) (User, string, error) {
	u, err := s.Store.Queries.GetUserByHandle(ctx, strings.TrimSpace(handle))
	if isNoRows(err) {
		_, _ = auth.CheckPassword(dummyHash, password) // same response time as a real check
		return User{}, "", ErrBadLogin
	}
	if err != nil {
		return User{}, "", err
	}
	ok, err := auth.CheckPassword(u.PasswordHash, password)
	if err != nil {
		return User{}, "", err
	}
	if !ok || u.DisabledAt.Valid {
		return User{}, "", ErrBadLogin
	}
	token, err := s.newSession(ctx, u.ID)
	return toUser(u), token, err
}

// Authenticate resolves a session token to its user, extending sessions that are in use.
func (s *Service) Authenticate(ctx context.Context, token string) (*sqlcgen.User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}
	now := s.Now()
	row, err := s.Store.Queries.GetSessionUser(ctx, sqlcgen.GetSessionUserParams{TokenHash: auth.HashToken(token), ExpiresAt: s.stamp(now)})
	if isNoRows(err) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if row.Session.ExpiresAt < s.stamp(now.Add(sessionLifetime/2)) {
		extend := func(q *sqlcgen.Queries) error {
			return q.ExtendSession(ctx, sqlcgen.ExtendSessionParams{ExpiresAt: s.stamp(now.Add(sessionLifetime)), ID: row.Session.ID})
		}
		if err := s.Store.Tx(ctx, extend); err != nil {
			s.Log.Warn("could not extend session", "error", err) // the session itself is still valid
		}
	}
	return &row.User, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error { return q.DeleteSessionByHash(ctx, auth.HashToken(token)) })
}

// Me returns the viewer's account.
func (s *Service) Me(v Viewer) User { return toUser(*v.User) }

func (s *Service) UpdateProfile(ctx context.Context, v Viewer, in ProfileInput) (User, error) {
	if !text.ValidLocale(in.Locale) {
		return User{}, invalid("bad_locale", "Unknown language")
	}
	switch in.AssistantNewEntries {
	case NewEntriesNever, NewEntriesAsk, NewEntriesAllow:
	default:
		return User{}, invalid("bad_new_entries", "Choose never, ask or allow")
	}
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		return q.UpdateProfile(ctx, sqlcgen.UpdateProfileParams{DisplayName: strings.TrimSpace(in.DisplayName), Locale: in.Locale, AssistantNewEntries: in.AssistantNewEntries, ID: v.id()})
	})
	if err != nil {
		return User{}, err
	}
	u, err := s.Store.Queries.GetUser(ctx, v.id())
	return toUser(u), err
}

// ChangePassword verifies the old password and ends all other sessions.
func (s *Service) ChangePassword(ctx context.Context, v Viewer, current, next string) (string, error) {
	if err := validatePassword(next); err != nil {
		return "", err
	}
	ok, err := auth.CheckPassword(v.User.PasswordHash, current)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrBadLogin
	}
	if err := s.setPassword(ctx, v.id(), next); err != nil {
		return "", err
	}
	return s.newSession(ctx, v.id())
}

func (s *Service) setPassword(ctx context.Context, userID int64, pw string) error {
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.UpdatePassword(ctx, sqlcgen.UpdatePasswordParams{PasswordHash: hash, ID: userID}); err != nil {
			return err
		}
		if err := q.DeletePasswordResets(ctx, userID); err != nil {
			return err
		}
		return q.DeleteUserSessions(ctx, userID)
	})
}

// CreatePasswordReset lets an admin issue a recovery link, so accounts need no email.
func (s *Service) CreatePasswordReset(ctx context.Context, v Viewer, userID int64) (string, error) {
	if !v.isAdmin() {
		return "", ErrForbidden
	}
	if _, err := s.Store.Queries.GetUser(ctx, userID); isNoRows(err) {
		return "", ErrNotFound
	} else if err != nil {
		return "", err
	}
	token, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.DeletePasswordResets(ctx, userID); err != nil {
			return err
		}
		return q.CreatePasswordReset(ctx, sqlcgen.CreatePasswordResetParams{TokenHash: auth.HashToken(token), UserID: userID, ExpiresAt: s.stamp(s.Now().Add(72 * time.Hour))})
	})
	return token, err
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	r, err := s.Store.Queries.GetPasswordReset(ctx, sqlcgen.GetPasswordResetParams{TokenHash: auth.HashToken(token), ExpiresAt: s.stamp(s.Now())})
	if isNoRows(err) {
		return fail(http.StatusForbidden, "invalid_reset", "This link is invalid or has expired")
	}
	if err != nil {
		return err
	}
	return s.setPassword(ctx, r.UserID, password)
}

// --- admin: users ---

func (s *Service) ListUsers(ctx context.Context, v Viewer) ([]User, error) {
	if !v.isAdmin() {
		return nil, ErrForbidden
	}
	rows, err := s.Store.Queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, len(rows))
	for i, u := range rows {
		out[i] = toUser(u)
	}
	return out, nil
}

func (s *Service) requireOtherAdmin(ctx context.Context, q *sqlcgen.Queries, u sqlcgen.User) error {
	if u.Role != "admin" || u.DisabledAt.Valid {
		return nil
	}
	n, err := q.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if n <= 1 {
		return fail(http.StatusConflict, "last_admin", "This is the only admin")
	}
	return nil
}

// AdminUpdateUser changes a user's disabled state or role. Disabling ends their sessions.
func (s *Service) AdminUpdateUser(ctx context.Context, v Viewer, id int64, disabled *bool, role *string) (User, error) {
	if !v.isAdmin() {
		return User{}, ErrForbidden
	}
	var out sqlcgen.User
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		u, err := q.GetUser(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if disabled != nil {
			if *disabled {
				if err := s.requireOtherAdmin(ctx, q, u); err != nil {
					return err
				}
				if err := q.SetUserDisabled(ctx, sqlcgen.SetUserDisabledParams{DisabledAt: nullStr(s.stamp(s.Now())), ID: id}); err != nil {
					return err
				}
				if err := q.DeleteUserSessions(ctx, id); err != nil {
					return err
				}
			} else if err := q.SetUserDisabled(ctx, sqlcgen.SetUserDisabledParams{ID: id}); err != nil {
				return err
			}
		}
		if role != nil {
			if *role != "admin" && *role != "user" {
				return invalid("bad_role", "Unknown role")
			}
			if *role == "user" {
				if err := s.requireOtherAdmin(ctx, q, u); err != nil {
					return err
				}
			}
			if err := q.SetUserRole(ctx, sqlcgen.SetUserRoleParams{Role: *role, ID: id}); err != nil {
				return err
			}
		}
		out, err = q.GetUser(ctx, id)
		return err
	})
	return toUser(out), err
}

// DeleteUser never removes recipes implicitly. If the user owns any, the admin must choose:
// transfer them to another user (reassignTo) or delete them (deleteRecipes).
func (s *Service) DeleteUser(ctx context.Context, v Viewer, id, reassignTo int64, deleteRecipes bool) error {
	if !v.isAdmin() {
		return ErrForbidden
	}
	if id == v.id() {
		return fail(http.StatusConflict, "cannot_delete_self", "You can't delete your own account")
	}
	var photos []string
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		u, err := q.GetUser(ctx, id)
		if isNoRows(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := s.requireOtherAdmin(ctx, q, u); err != nil {
			return err
		}
		n, err := q.CountRecipesByOwner(ctx, id)
		if err != nil {
			return err
		}
		if n > 0 {
			switch {
			case reassignTo > 0 && reassignTo != id:
				if _, err := q.GetUser(ctx, reassignTo); err != nil {
					return invalid("bad_target", "The user to receive the recipes doesn't exist")
				}
				if err := q.ReassignRecipes(ctx, sqlcgen.ReassignRecipesParams{Target: reassignTo, Source: id}); err != nil {
					return err
				}
			case deleteRecipes:
				rows, err := q.ListRecipesByOwner(ctx, id)
				if err != nil {
					return err
				}
				for _, r := range rows {
					if r.Photo.Valid {
						photos = append(photos, r.Photo.String)
					}
				}
				if err := q.DeleteRecipesByOwner(ctx, id); err != nil {
					return err
				}
			default:
				return fail(http.StatusConflict, "user_has_recipes", fmt.Sprintf("This user owns %d recipes. Choose what happens to them", n))
			}
		}
		if err := q.DeleteUser(ctx, id); err != nil {
			return err
		}
		members, err := q.CountHouseholdMembers(ctx, u.HouseholdID)
		if err != nil {
			return err
		}
		if members == 0 {
			return q.DeleteHousehold(ctx, u.HouseholdID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.removePhotos(photos)
	return nil
}

func (s *Service) removePhotos(photos []string) {
	for _, p := range photos {
		if err := s.Media.Delete(p); err != nil {
			s.Log.Warn("could not remove photo file", "photo", p, "error", err)
		}
	}
}

// --- invites ---

func (s *Service) createInvite(ctx context.Context, v Viewer, household int64, in InviteInput) (Invite, error) {
	token, err := auth.NewToken()
	if err != nil {
		return Invite{}, err
	}
	if in.MaxUses < 1 {
		in.MaxUses = 1
	}
	params := sqlcgen.CreateInviteParams{TokenHash: auth.HashToken(token), CreatedBy: v.id(), HouseholdID: nullID(household), MaxUses: in.MaxUses}
	if in.ExpiresInDays > 0 {
		params.ExpiresAt = nullStr(s.stamp(s.Now().AddDate(0, 0, int(in.ExpiresInDays))))
	}
	var row sqlcgen.Invite
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) (err error) {
		row, err = q.CreateInvite(ctx, params)
		return err
	})
	if err != nil {
		return Invite{}, err
	}
	out := toInvite(row)
	out.Token = token
	return out, nil
}

func toInvite(r sqlcgen.Invite) Invite {
	return Invite{ID: r.ID, MaxUses: r.MaxUses, UseCount: r.UseCount, ExpiresAt: r.ExpiresAt.String, CreatedAt: r.CreatedAt}
}

func (s *Service) CreateRegistrationInvite(ctx context.Context, v Viewer, in InviteInput) (Invite, error) {
	if !v.isAdmin() {
		return Invite{}, ErrForbidden
	}
	return s.createInvite(ctx, v, 0, in)
}

func (s *Service) ListRegistrationInvites(ctx context.Context, v Viewer) ([]Invite, error) {
	if !v.isAdmin() {
		return nil, ErrForbidden
	}
	rows, err := s.Store.Queries.ListRegistrationInvites(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Invite, len(rows))
	for i, r := range rows {
		out[i] = toInvite(r)
	}
	return out, nil
}

// DeleteInvite works for admins on any invite and for members on their household's invites.
func (s *Service) DeleteInvite(ctx context.Context, v Viewer, id int64) error {
	inv, err := s.Store.Queries.GetInvite(ctx, id)
	if isNoRows(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !v.isAdmin() && (!inv.HouseholdID.Valid || inv.HouseholdID.Int64 != v.User.HouseholdID) {
		return ErrForbidden
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error { return q.DeleteInvite(ctx, id) })
}

// --- households ---

func (s *Service) GetHousehold(ctx context.Context, v Viewer) (Household, error) {
	h, err := s.Store.Queries.GetHousehold(ctx, v.User.HouseholdID)
	if err != nil {
		return Household{}, err
	}
	members, err := s.Store.Queries.ListHouseholdMembers(ctx, h.ID)
	if err != nil {
		return Household{}, err
	}
	invites, err := s.Store.Queries.ListHouseholdInvites(ctx, nullID(h.ID))
	if err != nil {
		return Household{}, err
	}
	out := Household{ID: h.ID, Name: h.Name, Members: make([]Person, len(members)), Invites: make([]Invite, len(invites))}
	for i, m := range members {
		out.Members[i] = toPerson(m)
	}
	for i, inv := range invites {
		out.Invites[i] = toInvite(inv)
	}
	return out, nil
}

func (s *Service) RenameHousehold(ctx context.Context, v Viewer, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return invalid("bad_name", "Please enter a name")
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		return q.RenameHousehold(ctx, sqlcgen.RenameHouseholdParams{Name: name, ID: v.User.HouseholdID})
	})
}

func (s *Service) CreateHouseholdInvite(ctx context.Context, v Viewer, in InviteInput) (Invite, error) {
	return s.createInvite(ctx, v, v.User.HouseholdID, in)
}

// JoinHousehold moves the user into the invite's household. Their recipes stay theirs.
func (s *Service) JoinHousehold(ctx context.Context, v Viewer, token string) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		inv, err := s.consumeInvite(ctx, q, token)
		if err != nil {
			return err
		}
		if !inv.HouseholdID.Valid {
			return fail(http.StatusForbidden, "invalid_invite", "This invite is invalid, expired or already used")
		}
		return s.moveToHousehold(ctx, q, v.User, inv.HouseholdID.Int64)
	})
}

func (s *Service) LeaveHousehold(ctx context.Context, v Viewer) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.CountHouseholdMembers(ctx, v.User.HouseholdID)
		if err != nil {
			return err
		}
		if n <= 1 {
			return nil
		}
		h, err := q.CreateHousehold(ctx, v.User.Handle)
		if err != nil {
			return err
		}
		return s.moveToHousehold(ctx, q, v.User, h.ID)
	})
}

func (s *Service) moveToHousehold(ctx context.Context, q *sqlcgen.Queries, u *sqlcgen.User, target int64) error {
	if u.HouseholdID == target {
		return nil
	}
	if err := q.SetUserHousehold(ctx, sqlcgen.SetUserHouseholdParams{HouseholdID: target, ID: u.ID}); err != nil {
		return err
	}
	n, err := q.CountHouseholdMembers(ctx, u.HouseholdID)
	if err != nil {
		return err
	}
	if n == 0 {
		return q.DeleteHousehold(ctx, u.HouseholdID)
	}
	return nil
}

// --- friends: the only way to reach users, besides knowing their exact handle ---

// lookupHandle finds a user by their complete, exact handle. There is deliberately no
// partial match anywhere, so people can't be discovered by typing part of a name.
func (s *Service) lookupHandle(ctx context.Context, v Viewer, handle string) (sqlcgen.User, error) {
	missing := fail(http.StatusNotFound, "handle_not_found", "No user has exactly this handle")
	u, err := s.Store.Queries.GetUserByHandle(ctx, strings.TrimPrefix(strings.TrimSpace(handle), "@"))
	if isNoRows(err) || (err == nil && (u.DisabledAt.Valid || u.ID == v.id())) {
		return u, missing
	}
	return u, err
}

func (s *Service) ListFriends(ctx context.Context, v Viewer) ([]Friend, error) {
	rows, err := s.Store.Queries.ListFriendships(ctx, v.id())
	if err != nil {
		return nil, err
	}
	out := make([]Friend, 0, len(rows))
	for _, r := range rows {
		out = append(out, Friend{
			ID:       r.ID,
			Person:   Person{ID: r.OtherID, Handle: r.OtherHandle, DisplayName: r.OtherDisplayName},
			Accepted: r.AcceptedAt.Valid,
			Incoming: r.AddresseeID == v.id(),
		})
	}
	return out, nil
}

// RequestFriend sends a request to an exact handle. If that user already asked, it accepts.
func (s *Service) RequestFriend(ctx context.Context, v Viewer, handle string) error {
	u, err := s.lookupHandle(ctx, v, handle)
	if err != nil {
		return err
	}
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		f, err := q.GetFriendshipBetween(ctx, sqlcgen.GetFriendshipBetweenParams{A: v.id(), B: u.ID})
		if err == nil {
			if !f.AcceptedAt.Valid && f.AddresseeID == v.id() {
				_, err = q.AcceptFriendship(ctx, sqlcgen.AcceptFriendshipParams{ID: f.ID, AddresseeID: v.id()})
			}
			return err
		}
		if !isNoRows(err) {
			return err
		}
		return q.CreateFriendship(ctx, sqlcgen.CreateFriendshipParams{RequesterID: v.id(), AddresseeID: u.ID})
	})
}

func (s *Service) AcceptFriend(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.AcceptFriendship(ctx, sqlcgen.AcceptFriendshipParams{ID: id, AddresseeID: v.id()})
		if err == nil && n == 0 {
			return ErrNotFound
		}
		return err
	})
}

func (s *Service) RemoveFriend(ctx context.Context, v Viewer, id int64) error {
	return s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.DeleteFriendship(ctx, sqlcgen.DeleteFriendshipParams{ID: id, UserID: v.id()})
		if err == nil && n == 0 {
			return ErrNotFound
		}
		return err
	})
}

// VisibleUsers is the only user list any client gets: yourself, your household, your friends.
func (s *Service) VisibleUsers(ctx context.Context, v Viewer) ([]sqlcgen.User, error) {
	return s.Store.Queries.ListVisibleUsers(ctx, sqlcgen.ListVisibleUsersParams{UserID: v.id(), HouseholdID: v.User.HouseholdID})
}
