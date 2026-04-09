package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	dbq "github.com/serraniel/wokko/backend/internal/db/queries"
)

type UserAdminResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Role        string  `json:"role"`
	IsActive    bool    `json:"is_active"`
}

func userToAdminResponse(u dbq.User) UserAdminResponse {
	id, _ := uuid.FromBytes(u.ID.Bytes[:])
	r := UserAdminResponse{
		ID:       id.String(),
		Username: u.Username,
		Role:     u.Role,
		IsActive: u.IsActive,
	}
	if u.Email.Valid {
		r.Email = &u.Email.String
	}
	if u.DisplayName.Valid {
		r.DisplayName = &u.DisplayName.String
	}
	return r
}

// GET /api/admin/users
func (s *Server) ListUsers(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	limit := intQueryParam(c, "limit", 20, 200)
	offset := intQueryParam(c, "offset", 0, 0)

	users, err := q.ListUsers(ctx, dbq.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	out := make([]UserAdminResponse, 0, len(users))
	for _, u := range users {
		out = append(out, userToAdminResponse(u))
	}
	return c.JSON(http.StatusOK, out)
}

// DELETE /api/admin/users/:id
func (s *Server) DeleteUserAdmin(c echo.Context) error {
	callerID, _ := c.Get("user_id").(uuid.UUID)
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	targetUUID, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	targetID, _ := uuid.FromBytes(targetUUID.Bytes[:])
	if targetID == callerID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot delete self"})
	}

	if err := q.DeleteUser(ctx, targetUUID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// POST /api/admin/invites
func (s *Server) CreateInviteLinkAdmin(c echo.Context) error {
	callerID, _ := c.Get("user_id").(uuid.UUID)
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	var req struct {
		MaxUses       *int32 `json:"max_uses"`
		ExpiresInDays *int   `json:"expires_in_days"`
	}
	_ = c.Bind(&req)

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}
	token := hex.EncodeToString(tokenBytes)

	maxUses := int32(1)
	if req.MaxUses != nil {
		maxUses = *req.MaxUses
	}

	params := dbq.CreateInviteLinkParams{
		Token:     token,
		CreatedBy: uuidToPgtype(callerID),
		MaxUses:   maxUses,
	}
	if req.ExpiresInDays != nil {
		t := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		params.ExpiresAt = pgtype.Timestamptz{Time: t, Valid: true}
	}

	link, err := q.CreateInviteLink(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	id, _ := uuid.FromBytes(link.ID.Bytes[:])
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":       id.String(),
		"token":    link.Token,
		"max_uses": link.MaxUses,
		"url":      s.cfg.PublicURL + "/register?invite=" + link.Token,
	})
}

// GET /api/admin/invites
func (s *Server) ListInviteLinksAdmin(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	links, err := q.ListInviteLinks(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	out := make([]map[string]interface{}, 0, len(links))
	for _, l := range links {
		id, _ := uuid.FromBytes(l.ID.Bytes[:])
		m := map[string]interface{}{
			"id":        id.String(),
			"token":     l.Token,
			"max_uses":  l.MaxUses,
			"use_count": l.UseCount,
		}
		if l.ExpiresAt.Valid {
			m["expires_at"] = l.ExpiresAt.Time
		}
		out = append(out, m)
	}
	return c.JSON(http.StatusOK, out)
}

// DELETE /api/admin/invites/:id
func (s *Server) DeleteInviteLinkAdmin(c echo.Context) error {
	ctx := c.Request().Context()
	q := dbq.New(s.pool)

	id, err := parsePgtypeUUID(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := q.DeleteInviteLink(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// GET /api/admin/settings
func (s *Server) GetSettings(c echo.Context) error {
	ctx := c.Request().Context()
	rows, err := s.pool.Query(ctx, "SELECT key, value FROM system_settings")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	settings := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		settings[key] = value
	}
	return c.JSON(http.StatusOK, settings)
}

// PUT /api/admin/settings
func (s *Server) UpdateSettings(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		OpenRegistration   *bool `json:"open_registration"`
		AllowPublicRecipes *bool `json:"allow_public_recipes"`
		MCPEnabled         *bool `json:"mcp_enabled"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	updates := map[string]string{}
	if req.OpenRegistration != nil {
		if *req.OpenRegistration {
			updates["open_registration"] = "true"
		} else {
			updates["open_registration"] = "false"
		}
	}
	if req.AllowPublicRecipes != nil {
		if *req.AllowPublicRecipes {
			updates["allow_public_recipes"] = "true"
		} else {
			updates["allow_public_recipes"] = "false"
		}
	}
	if req.MCPEnabled != nil {
		if *req.MCPEnabled {
			updates["mcp_enabled"] = "true"
		} else {
			updates["mcp_enabled"] = "false"
		}
	}

	for key, value := range updates {
		if _, err := s.pool.Exec(ctx, "UPDATE system_settings SET value=$2, updated_at=NOW() WHERE key=$1", key, value); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
