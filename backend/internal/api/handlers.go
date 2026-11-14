package api

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/importer"
	"github.com/serraniel/wokko/backend/internal/media"
)

func ok(w http.ResponseWriter) error {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	return nil
}

func reply[T any](w http.ResponseWriter, v T, err error) error {
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, v)
	return nil
}

func replyCreated[T any](w http.ResponseWriter, v T, err error) error {
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, v)
	return nil
}

func done(w http.ResponseWriter, err error) error {
	if err != nil {
		return err
	}
	return ok(w)
}

// --- accounts ---

func (s *Server) status(w http.ResponseWriter, r *http.Request) error {
	st, err := s.svc.Status(r.Context())
	return reply(w, st, err)
}

type credentials struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Invite      string `json:"invite"`
	Locale      string `json:"locale"`
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) error {
	var in credentials
	if err := readJSON(r, &in); err != nil {
		return err
	}
	u, token, err := s.svc.Setup(r.Context(), in.Handle, in.DisplayName, in.Password, in.Locale)
	if err != nil {
		return err
	}
	s.setSession(w, token)
	writeJSON(w, http.StatusCreated, u)
	return nil
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	var in credentials
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if !s.loginLimit.Allow(clientKey(r, in.Handle)) {
		return errTooMany
	}
	u, token, err := s.svc.Login(r.Context(), in.Handle, in.Password)
	if err != nil {
		return err
	}
	s.setSession(w, token)
	writeJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	var in credentials
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if !s.loginLimit.Allow(clientKey(r, "register")) {
		return errTooMany
	}
	u, token, err := s.svc.Register(r.Context(), in.Handle, in.DisplayName, in.Password, in.Invite, in.Locale)
	if err != nil {
		return err
	}
	s.setSession(w, token)
	writeJSON(w, http.StatusCreated, u)
	return nil
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	if t := sessionToken(r); t != "" {
		if err := s.svc.Logout(r.Context(), t); err != nil {
			return err
		}
	}
	s.setSession(w, "")
	return ok(w)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if !s.loginLimit.Allow(clientKey(r, "reset")) {
		return errTooMany
	}
	return done(w, s.svc.ResetPassword(r.Context(), in.Token, in.Password))
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return reply(w, s.svc.Me(v), nil)
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	in := struct {
		DisplayName         *string `json:"display_name"`
		Locale              *string `json:"locale"`
		AssistantNewEntries *string `json:"assistant_new_entries"`
	}{}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	p := app.ProfileInput{DisplayName: v.User.DisplayName, Locale: v.User.Locale, AssistantNewEntries: v.User.AssistantNewEntries}
	if in.DisplayName != nil {
		p.DisplayName = *in.DisplayName
	}
	if in.Locale != nil {
		p.Locale = *in.Locale
	}
	if in.AssistantNewEntries != nil {
		p.AssistantNewEntries = *in.AssistantNewEntries
	}
	u, err := s.svc.UpdateProfile(r.Context(), v, p)
	return reply(w, u, err)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if !s.loginLimit.Allow(clientKey(r, "pw:"+v.User.Handle)) {
		return errTooMany
	}
	token, err := s.svc.ChangePassword(r.Context(), v, in.Current, in.Next)
	if err != nil {
		return err
	}
	s.setSession(w, token)
	return ok(w)
}

func (s *Server) export(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	format, err := app.ExportFormat(r.URL.Query().Get("format"))
	if err != nil {
		return err
	}
	data, err := s.svc.ExportScope(r.Context(), v, r.URL.Query().Get("scope"))
	if err != nil {
		return err
	}
	return s.sendExport(w, format, data)
}

func (s *Server) exportRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	format, err := app.ExportFormat(r.URL.Query().Get("format"))
	if err != nil {
		return err
	}
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	data, err := s.svc.ExportRecipe(r.Context(), v, id)
	if err != nil {
		return err
	}
	return s.sendExport(w, format, data)
}

func (s *Server) exportShared(w http.ResponseWriter, r *http.Request) error {
	format, err := app.ExportFormat(r.URL.Query().Get("format"))
	if err != nil {
		return err
	}
	data, err := s.svc.ExportSharedRecipe(r.Context(), s.anonLocale(r), r.PathValue("token"))
	if err != nil {
		return err
	}
	return s.sendExport(w, format, data)
}

func (s *Server) exportPublic(w http.ResponseWriter, r *http.Request) error {
	format, err := app.ExportFormat(r.URL.Query().Get("format"))
	if err != nil {
		return err
	}
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	data, err := s.svc.ExportPublicRecipe(r.Context(), s.anonLocale(r), id)
	if err != nil {
		return err
	}
	return s.sendExport(w, format, data)
}

// sendExport streams the archive. Every check has passed by now, so the headers can go out.
func (s *Server) sendExport(w http.ResponseWriter, format string, data app.ExportData) error {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+app.FileName(data, format)+`"`)
	w.Header().Set("Cache-Control", "no-store")
	return s.svc.WriteExport(w, format, data)
}

// --- recipes ---

func (s *Server) listRecipes(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	q := r.URL.Query()
	p, err := s.svc.ListRecipes(r.Context(), v, app.RecipeQuery{
		Q: q.Get("q"), Sort: q.Get("sort"), KeywordID: queryInt(r, "keyword"), Owner: queryInt(r, "owner"),
		Limit: queryInt(r, "limit"), Offset: queryInt(r, "offset"),
	})
	return reply(w, p, err)
}

func (s *Server) createRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.RecipeInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	rec, err := s.svc.CreateRecipe(r.Context(), v, in)
	return replyCreated(w, rec, err)
}

func (s *Server) getRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	rec, err := s.svc.GetRecipe(r.Context(), v, id)
	return reply(w, rec, err)
}

func (s *Server) updateRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in app.RecipeInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	rec, err := s.svc.UpdateRecipe(r.Context(), v, id, in)
	return reply(w, rec, err)
}

func (s *Server) withID(w http.ResponseWriter, r *http.Request, fn func(id int64) error) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return done(w, fn(id))
}

func (s *Server) deleteRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.DeleteRecipe(r.Context(), v, id) })
}

func (s *Server) restoreRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.RestoreRecipe(r.Context(), v, id) })
}

func (s *Server) purgeRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.PurgeRecipe(r.Context(), v, id) })
}

func (s *Server) listTrash(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	t, err := s.svc.ListTrash(r.Context(), v)
	return reply(w, t, err)
}

func (s *Server) importRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	if !s.importLimit.Allow(strconv.FormatInt(v.User.ID, 10)) {
		return errTooMany
	}
	var in app.ImportRequest
	if err := readJSON(r, &in); err != nil {
		return err
	}
	out, err := s.svc.ImportRecipe(r.Context(), in)
	return reply(w, out, err)
}

// importFile streams the upload to a temporary file beside the data, because archives are read
// at random and can be far bigger than is sensible to hold in memory.
func (s *Server) importFile(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	if !s.importLimit.Allow(strconv.FormatInt(v.User.ID, 10)) {
		return errTooMany
	}
	bad := &app.Error{Status: http.StatusBadRequest, Code: "import_file_unknown", Message: "No recipes found in this file"}
	r.Body = http.MaxBytesReader(w, r.Body, importer.MaxFileBytes+1<<20)
	parts, err := r.MultipartReader()
	if err != nil {
		return bad
	}
	var part *multipart.Part
	for {
		if part, err = parts.NextPart(); err != nil {
			return bad
		}
		if part.FormName() == "file" {
			break
		}
	}
	tmp, err := os.CreateTemp(s.svc.Cfg.DataDir, "import-*.tmp")
	if err != nil {
		return fmt.Errorf("create import file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	size, err := io.Copy(tmp, part)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return &app.Error{Status: http.StatusRequestEntityTooLarge, Code: "import_file_too_big", Message: "This file is too big to import"}
		}
		return bad
	}
	out, err := s.svc.ImportFile(r.Context(), v, tmp, size, part.FileName(), r.URL.Query().Get("dry_run") != "")
	return reply(w, out, err)
}

func (s *Server) uploadPhoto(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+1<<20)
	file, _, err := r.FormFile("photo")
	if err != nil {
		return &app.Error{Status: http.StatusBadRequest, Code: "invalid_image", Message: "Please choose a photo no larger than 20 MB"}
	}
	defer file.Close()
	rec, err := s.svc.SetPhoto(r.Context(), v, id, file)
	return reply(w, rec, err)
}

func (s *Server) deletePhoto(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.DeletePhoto(r.Context(), v, id) })
}

func (s *Server) listVersions(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	out, err := s.svc.ListVersions(r.Context(), v, id)
	return reply(w, out, err)
}

func (s *Server) restoreVersion(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	vid, err := pathID(r, "vid")
	if err != nil {
		return err
	}
	rec, err := s.svc.RestoreVersion(r.Context(), v, id, vid)
	return reply(w, rec, err)
}

func (s *Server) getSharing(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	out, err := s.svc.GetSharing(r.Context(), v, id)
	return reply(w, out, err)
}

func (s *Server) shareUser(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		Handle string `json:"handle"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if !s.handleLimit.Allow(strconv.FormatInt(v.User.ID, 10)) {
		return errTooMany
	}
	return done(w, s.svc.ShareWithUser(r.Context(), v, id, in.Handle))
}

func (s *Server) unshareUser(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	uid, err := pathID(r, "uid")
	if err != nil {
		return err
	}
	return done(w, s.svc.UnshareWithUser(r.Context(), v, id, uid))
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		ExpiresInDays int64 `json:"expires_in_days"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	l, err := s.svc.CreateShareLink(r.Context(), v, id, in.ExpiresInDays)
	return replyCreated(w, l, err)
}

func (s *Server) revokeLink(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	lid, err := pathID(r, "lid")
	if err != nil {
		return err
	}
	return done(w, s.svc.RevokeShareLink(r.Context(), v, id, lid))
}

func (s *Server) listCookLog(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	out, err := s.svc.ListCookLog(r.Context(), v, id)
	return reply(w, out, err)
}

func (s *Server) addCookLog(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in app.CookLogInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	e, err := s.svc.AddCookLog(r.Context(), v, id, in)
	return replyCreated(w, e, err)
}

// --- catalog ---

func (s *Server) listFoods(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p, err := s.svc.ListFoods(r.Context(), v, r.URL.Query().Get("q"), queryInt(r, "limit"), queryInt(r, "offset"))
	return reply(w, p, err)
}

func (s *Server) getFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	f, err := s.svc.GetFood(r.Context(), v, id)
	return reply(w, f, err)
}

func (s *Server) createFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.FoodInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	f, err := s.svc.CreateFood(r.Context(), v, in)
	return replyCreated(w, f, err)
}

func (s *Server) updateFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in app.FoodInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	f, err := s.svc.UpdateFood(r.Context(), v, id, in)
	return reply(w, f, err)
}

func (s *Server) deleteFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.DeleteFood(r.Context(), v, id) })
}

func (s *Server) mergeFoods(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.MergeInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	return done(w, s.svc.MergeFoods(r.Context(), v, in))
}

func (s *Server) listUnits(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p, err := s.svc.ListUnits(r.Context(), v, r.URL.Query().Get("q"), queryInt(r, "limit"), queryInt(r, "offset"))
	return reply(w, p, err)
}

func (s *Server) getUnit(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	u, err := s.svc.GetUnit(r.Context(), v, id)
	return reply(w, u, err)
}

func (s *Server) createUnit(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.UnitInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	u, err := s.svc.CreateUnit(r.Context(), v, in)
	return replyCreated(w, u, err)
}

func (s *Server) updateUnit(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in app.UnitInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	u, err := s.svc.UpdateUnit(r.Context(), v, id, in)
	return reply(w, u, err)
}

func (s *Server) deleteUnit(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.DeleteUnit(r.Context(), v, id) })
}

func (s *Server) mergeUnits(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.MergeInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	return done(w, s.svc.MergeUnits(r.Context(), v, in))
}

func (s *Server) listKeywords(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p, err := s.svc.ListKeywords(r.Context(), v, r.URL.Query().Get("q"), queryInt(r, "limit"), queryInt(r, "offset"))
	return reply(w, p, err)
}

// --- friends and household ---

func (s *Server) listFriends(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListFriends(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) requestFriend(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Handle string `json:"handle"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if !s.handleLimit.Allow(strconv.FormatInt(v.User.ID, 10)) {
		return errTooMany
	}
	return done(w, s.svc.RequestFriend(r.Context(), v, in.Handle))
}

func (s *Server) acceptFriend(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.AcceptFriend(r.Context(), v, id) })
}

func (s *Server) removeFriend(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.RemoveFriend(r.Context(), v, id) })
}

func (s *Server) getHousehold(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	h, err := s.svc.GetHousehold(r.Context(), v)
	return reply(w, h, err)
}

func (s *Server) renameHousehold(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	return done(w, s.svc.RenameHousehold(r.Context(), v, in.Name))
}

func (s *Server) createHouseholdInvite(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.InviteInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	inv, err := s.svc.CreateHouseholdInvite(r.Context(), v, in)
	return replyCreated(w, inv, err)
}

func (s *Server) deleteInvite(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.DeleteInvite(r.Context(), v, id) })
}

func (s *Server) joinHousehold(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Token string `json:"token"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	if !s.loginLimit.Allow(clientKey(r, "join")) {
		return errTooMany
	}
	return done(w, s.svc.JoinHousehold(r.Context(), v, in.Token))
}

func (s *Server) leaveHousehold(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return done(w, s.svc.LeaveHousehold(r.Context(), v))
}

// --- admin ---

func (s *Server) adminListUsers(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListUsers(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) adminUpdateUser(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		Disabled *bool   `json:"disabled"`
		Role     *string `json:"role"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	u, err := s.svc.AdminUpdateUser(r.Context(), v, id, in.Disabled, in.Role)
	return reply(w, u, err)
}

func (s *Server) adminDeleteUser(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	q := r.URL.Query()
	return done(w, s.svc.DeleteUser(r.Context(), v, id, queryInt(r, "reassign_to"), q.Get("delete_recipes") == "true"))
}

func (s *Server) adminResetLink(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	token, err := s.svc.CreatePasswordReset(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
	return nil
}

func (s *Server) adminListInvites(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListRegistrationInvites(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) adminCreateInvite(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.InviteInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	inv, err := s.svc.CreateRegistrationInvite(r.Context(), v, in)
	return replyCreated(w, inv, err)
}

func (s *Server) adminGetSettings(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	if v.User.Role != "admin" {
		return app.ErrForbidden
	}
	st, err := s.svc.GetSettings(r.Context())
	return reply(w, st, err)
}

func (s *Server) adminUpdateSettings(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.Settings
	if err := readJSON(r, &in); err != nil {
		return err
	}
	st, err := s.svc.UpdateSettings(r.Context(), v, in)
	return reply(w, st, err)
}

func (s *Server) adminHealth(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	h, err := s.svc.GetHealth(r.Context(), v)
	return reply(w, h, err)
}

func (s *Server) adminBackup(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	if v.User.Role != "admin" {
		return app.ErrForbidden
	}
	run, err := s.svc.RunBackup(r.Context())
	if err != nil {
		return &app.Error{Status: http.StatusInternalServerError, Code: "backup_failed", Message: err.Error()}
	}
	writeJSON(w, http.StatusCreated, run)
	return nil
}

// --- public ---

func (s *Server) listPublic(w http.ResponseWriter, r *http.Request) error {
	p, err := s.svc.ListPublicRecipes(r.Context(), s.anonLocale(r), r.URL.Query().Get("q"), queryInt(r, "limit"), queryInt(r, "offset"))
	return reply(w, p, err)
}

func (s *Server) getPublic(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	rec, err := s.svc.GetPublicRecipe(r.Context(), s.anonLocale(r), id)
	return reply(w, rec, err)
}

func (s *Server) getShared(w http.ResponseWriter, r *http.Request) error {
	rec, err := s.svc.GetSharedRecipe(r.Context(), s.anonLocale(r), r.PathValue("token"))
	return reply(w, rec, err)
}

// serveMedia only sends a photo to someone who may see its recipe.
func (s *Server) serveMedia(w http.ResponseWriter, r *http.Request) error {
	file := r.PathValue("file")
	path, valid := s.svc.Media.Path(file)
	if !valid || !strings.HasSuffix(file, ".jpg") {
		return app.ErrNotFound
	}
	v := app.Viewer{}
	if u, err := s.svc.Authenticate(r.Context(), sessionToken(r)); err == nil {
		v.User = u
	} else if !errors.Is(err, app.ErrUnauthorized) {
		return fmt.Errorf("authenticate photo request: %w", err)
	}
	allowed, err := s.svc.CanSeePhoto(r.Context(), v, file, r.URL.Query().Get("share"))
	if err != nil {
		return err
	}
	if !allowed {
		return app.ErrNotFound
	}
	if _, err := os.Stat(path); err != nil {
		return app.ErrNotFound
	}
	// File names are unique per upload, so they can be cached, but only privately.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, path)
	return nil
}

func (s *Server) recipeInsights(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	res, err := s.svc.RecipeInsights(r.Context(), v, id)
	return reply(w, res, err)
}

func (s *Server) sharedInsights(w http.ResponseWriter, r *http.Request) error {
	res, err := s.svc.SharedRecipeInsights(r.Context(), s.anonLocale(r), r.PathValue("token"))
	return reply(w, res, err)
}

func (s *Server) publicInsights(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	res, err := s.svc.PublicRecipeInsights(r.Context(), s.anonLocale(r), id)
	return reply(w, res, err)
}

func (s *Server) catalogStatus(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	st, err := s.svc.CatalogStatus(r.Context(), v)
	return reply(w, st, err)
}

func (s *Server) resolveCatalogSuggestion(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		Take bool `json:"take"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	return reply(w, map[string]bool{"ok": true}, s.svc.ResolveCatalogSuggestion(r.Context(), v, id, in.Take))
}
