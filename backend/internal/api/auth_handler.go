package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"

	"github.com/serraniel/wokko/backend/internal/auth"
)

// Request/response types

type setupRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type registerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	InviteToken string `json:"invite_token"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

type userResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role"`
}

// helpers

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum)
}

func pgUUIDFromUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func uuidFromPgUUID(u pgtype.UUID) (uuid.UUID, error) {
	if !u.Valid {
		return uuid.UUID{}, fmt.Errorf("null uuid")
	}
	return u.Bytes, nil
}

func (s *Server) issueTokens(c echo.Context, q *dbq.Queries, user dbq.User) error {
	userID, err := uuidFromPgUUID(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid user id")
	}

	accessToken, err := auth.GenerateAccessToken(s.cfg.JWTSecret, userID, user.Username, user.Role)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not generate access token")
	}

	rawRefresh, err := generateRefreshToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not generate refresh token")
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = q.CreateRefreshToken(c.Request().Context(), dbq.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not store refresh token")
	}

	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    rawRefresh,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  expiresAt,
	})

	return c.JSON(http.StatusOK, tokenResponse{AccessToken: accessToken})
}

// SetupAdmin handles POST /api/auth/setup
func (s *Server) SetupAdmin(c echo.Context) error {
	q := dbq.New(s.pool)

	val, err := q.GetSetupComplete(c.Request().Context())
	if err == nil && val == "true" {
		return echo.NewHTTPError(http.StatusBadRequest, "already set up")
	}

	var req setupRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if len(req.Username) < 3 || len(req.Username) > 64 {
		return echo.NewHTTPError(http.StatusBadRequest, "username must be 3-64 characters")
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not hash password")
	}

	user, err := q.CreateUser(c.Request().Context(), dbq.CreateUserParams{
		Username:     req.Username,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
		Role:         "admin",
		DisplayName:  pgtype.Text{String: req.DisplayName, Valid: req.DisplayName != ""},
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not create user")
	}

	if err := q.SetSetupComplete(c.Request().Context()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not mark setup complete")
	}

	return s.issueTokens(c, q, user)
}

// Register handles POST /api/auth/register
func (s *Server) Register(c echo.Context) error {
	q := dbq.New(s.pool)

	val, err := q.GetSetupComplete(c.Request().Context())
	if err != nil || val != "true" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "setup not complete")
	}

	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if len(req.Username) < 3 || len(req.Username) > 64 {
		return echo.NewHTTPError(http.StatusBadRequest, "username must be 3-64 characters")
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	var invite *dbq.InviteLink
	if !s.cfg.OpenRegistration {
		if req.InviteToken == "" {
			return echo.NewHTTPError(http.StatusForbidden, "invite token required")
		}
		inv, err := q.GetInviteLinkByToken(c.Request().Context(), req.InviteToken)
		if err != nil {
			return echo.NewHTTPError(http.StatusForbidden, "invalid invite token")
		}
		if inv.ExpiresAt.Valid && inv.ExpiresAt.Time.Before(time.Now()) {
			return echo.NewHTTPError(http.StatusForbidden, "invite token expired")
		}
		if inv.UseCount >= inv.MaxUses {
			return echo.NewHTTPError(http.StatusForbidden, "invite token exhausted")
		}
		invite = &inv
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not hash password")
	}

	user, err := q.CreateUser(c.Request().Context(), dbq.CreateUserParams{
		Username:     req.Username,
		Email:        pgtype.Text{String: req.Email, Valid: req.Email != ""},
		DisplayName:  pgtype.Text{String: req.DisplayName, Valid: req.DisplayName != ""},
		PasswordHash: pgtype.Text{String: hash, Valid: true},
		Role:         "user",
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not create user")
	}

	if invite != nil {
		_ = q.UseInviteLink(c.Request().Context(), dbq.UseInviteLinkParams{
			Token:  invite.Token,
			UsedBy: user.ID,
		})
	}

	return s.issueTokens(c, q, user)
}

// Login handles POST /api/auth/login
func (s *Server) Login(c echo.Context) error {
	q := dbq.New(s.pool)

	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	user, err := q.GetUserByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	if !user.PasswordHash.Valid || !auth.CheckPassword(user.PasswordHash.String, req.Password) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	return s.issueTokens(c, q, user)
}

// RefreshToken handles POST /api/auth/refresh
func (s *Server) RefreshToken(c echo.Context) error {
	q := dbq.New(s.pool)

	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing refresh token")
	}

	tokenHash := hashToken(cookie.Value)
	rt, err := q.GetRefreshTokenByHash(c.Request().Context(), tokenHash)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
	}

	user, err := q.GetUserByID(c.Request().Context(), rt.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}

	userID, err := uuidFromPgUUID(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid user id")
	}

	accessToken, err := auth.GenerateAccessToken(s.cfg.JWTSecret, userID, user.Username, user.Role)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not generate access token")
	}

	return c.JSON(http.StatusOK, tokenResponse{AccessToken: accessToken})
}

// ValidateSession handles GET /api/auth/validate
// It checks the refresh token cookie without rotating it.
func (s *Server) ValidateSession(c echo.Context) error {
	q := dbq.New(s.pool)

	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing refresh token")
	}

	tokenHash := hashToken(cookie.Value)
	_, err = q.GetRefreshTokenByHash(c.Request().Context(), tokenHash)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
	}

	return c.JSON(http.StatusOK, map[string]bool{"valid": true})
}

// Logout handles POST /api/auth/logout
func (s *Server) Logout(c echo.Context) error {
	q := dbq.New(s.pool)

	cookie, err := c.Cookie("refresh_token")
	if err == nil {
		tokenHash := hashToken(cookie.Value)
		_ = q.DeleteRefreshToken(c.Request().Context(), tokenHash)
	}

	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1,
	})

	return c.NoContent(http.StatusNoContent)
}

// GetCurrentUser handles GET /api/auth/me
func (s *Server) GetCurrentUser(c echo.Context) error {
	q := dbq.New(s.pool)

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing user context")
	}

	user, err := q.GetUserByID(c.Request().Context(), pgUUIDFromUUID(userID))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	resp := userResponse{
		ID:       userID.String(),
		Username: user.Username,
		Role:     user.Role,
	}
	if user.Email.Valid {
		resp.Email = user.Email.String
	}
	if user.DisplayName.Valid {
		resp.DisplayName = user.DisplayName.String
	}

	return c.JSON(http.StatusOK, resp)
}

// Passkey stubs

func (s *Server) PasskeyRegisterBegin(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}

func (s *Server) PasskeyRegisterFinish(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}

func (s *Server) PasskeyLoginBegin(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}

func (s *Server) PasskeyLoginFinish(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "not implemented")
}
