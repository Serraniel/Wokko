// Package kitshn lets the kitshn mobile app (a Tandoor client) use Wokko. It translates
// between Tandoor's API shapes and package app, so kitshn gets the same permissions and data
// safety as the web app.
//
// Response types declare exactly the fields of kitshn's own data classes (checked against
// kitshn v2.1.5), which is what SupportedKitshn refers to.
package kitshn

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

const (
	// SessionCookie and csrfCookie use the names kitshn's Django login flow expects.
	SessionCookie = "sessionid"
	csrfCookie    = "csrftoken"

	// SupportedKitshn is the kitshn release this layer was written against.
	SupportedKitshn = "2.1.5"
	// tandoorVersion is reported to kitshn, which expects a Tandoor server.
	tandoorVersion = "2.7.2"
)

// NewSessionCookie is the session cookie used by both the web app and kitshn. An empty
// token clears it.
func NewSessionCookie(token string, secure bool) *http.Cookie {
	c := &http.Cookie{Name: SessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 60 * 24 * 3600}
	if token == "" {
		c.MaxAge = -1
	}
	return c
}

type Layer struct {
	svc    *app.Service
	log    *slog.Logger
	limit  *auth.Limiter
	secure bool
}

func New(svc *app.Service, log *slog.Logger, limit *auth.Limiter, secure bool) *Layer {
	return &Layer{svc: svc, log: log, limit: limit, secure: secure}
}

// Routes registers everything kitshn calls. Go's {$} matches the exact path, trailing slash
// included, as Tandoor's API uses.
func (l *Layer) Routes(m *http.ServeMux) {
	m.HandleFunc("GET /_allauth/browser/v1/auth/session", l.session)
	m.HandleFunc("POST /_allauth/browser/v1/auth/login", l.login)
	m.HandleFunc("GET /api/{$}", l.authed(l.root))
	m.HandleFunc("GET /api/server-settings/current/{$}", l.authed(l.serverSettings))
	m.HandleFunc("GET /api/user/{$}", l.authed(l.users))
	m.HandleFunc("GET /api/user-preference/{$}", l.authed(l.preference))
	m.HandleFunc("GET /api/space/{$}", l.authed(l.spaces))
	m.HandleFunc("GET /api/space/current/{$}", l.authed(l.currentSpace))
	m.HandleFunc("GET /api/switch-active-space/{id}/{$}", l.authed(l.currentSpace))
	m.HandleFunc("GET /api/recipe/{$}", l.authed(l.listRecipes))
	m.HandleFunc("GET /api/recipe/{id}/{$}", l.authed(l.getRecipe))
	m.HandleFunc("GET /api/step/{id}/{$}", l.authed(l.getStep))
	m.HandleFunc("GET /api/food/{$}", l.authed(l.listFoods))
	m.HandleFunc("GET /api/food/{id}/{$}", l.authed(l.getFood))
	m.HandleFunc("GET /api/unit/{$}", l.authed(l.listUnits))
	m.HandleFunc("GET /api/unit/{id}/{$}", l.authed(l.getUnit))
	m.HandleFunc("GET /api/keyword/{$}", l.authed(l.listKeywords))
	m.HandleFunc("GET /api/keyword/{id}/{$}", l.authed(l.getKeyword))
	m.HandleFunc("GET /api/cook-log/{$}", l.authed(l.listCookLog))
	m.HandleFunc("POST /api/cook-log/{$}", l.authed(l.addCookLog))
	// Tandoor features Wokko doesn't have answer with empty lists, so kitshn's screens stay usable.
	for _, area := range []string{"shopping-list", "ai-provider"} {
		m.HandleFunc("GET /api/"+area+"/{$}", l.authed(l.emptyList))
	}
	l.writeRoutes(m)
	// Anything else fails as JSON, never with the web app's HTML, so kitshn reports it cleanly.
	m.HandleFunc("/api/", l.authed(func(http.ResponseWriter, *http.Request, app.Viewer) error { return app.ErrNotFound }))
}

type handler func(w http.ResponseWriter, r *http.Request, v app.Viewer) error

// authed accepts the session as a cookie or a Bearer token. Like Tandoor, it answers 403 to
// unauthenticated requests, because kitshn's connection test expects exactly that.
func (l *Layer) authed(fn handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := l.svc.Authenticate(r.Context(), requestToken(r))
		if errors.Is(err, app.ErrUnauthorized) {
			writeJSON(w, http.StatusForbidden, detail("Authentication credentials were not provided."))
			return
		}
		if err == nil {
			err = fn(w, r, app.Viewer{User: u, Locale: u.Locale})
		}
		if err != nil {
			l.writeError(w, r, err)
		}
	}
}

func requestToken(r *http.Request) string {
	if t, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return t
	}
	if c, err := r.Cookie(SessionCookie); err == nil {
		return c.Value
	}
	return ""
}

func detail(msg string) map[string]string { return map[string]string{"detail": msg} }

func (l *Layer) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *app.Error
	if errors.As(err, &ae) {
		writeJSON(w, ae.Status, detail(ae.Message))
		return
	}
	l.log.Error("kitshn request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	writeJSON(w, http.StatusInternalServerError, detail("Server error"))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the status is already sent; a broken connection can't be reported
}

// --- login ---

// session starts kitshn's login: it needs a csrftoken cookie, even on the 401 answer.
func (l *Layer) session(w http.ResponseWriter, r *http.Request) {
	token, err := auth.NewToken()
	if err != nil {
		l.writeError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: csrfCookie, Value: token, Path: "/", SameSite: http.SameSiteLaxMode, Secure: l.secure, MaxAge: 365 * 24 * 3600})
	writeJSON(w, http.StatusUnauthorized, map[string]any{"status": 401, "data": map[string]any{"flows": []any{}}, "meta": map[string]any{"is_authenticated": false}})
}

func (l *Layer) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "errors": []any{}})
		return
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	if !l.limit.Allow(host + "|" + strings.ToLower(in.Username)) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"status": 429, "errors": []any{}})
		return
	}
	u, token, err := l.svc.Login(r.Context(), in.Username, in.Password)
	if err != nil {
		var ae *app.Error
		if errors.As(err, &ae) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"status": 400, "errors": []any{map[string]string{"message": ae.Message}}})
			return
		}
		l.writeError(w, r, err)
		return
	}
	http.SetCookie(w, NewSessionCookie(token, l.secure))
	writeJSON(w, http.StatusOK, map[string]any{"status": 200, "data": map[string]any{"user": map[string]any{"id": u.ID, "username": u.Handle, "display": u.DisplayName}}, "meta": map[string]any{"is_authenticated": true}})
}

// --- basic info ---

func (l *Layer) root(w http.ResponseWriter, r *http.Request, _ app.Viewer) error {
	writeJSON(w, http.StatusOK, map[string]string{"recipe": "/api/recipe/"})
	return nil
}

func (l *Layer) serverSettings(w http.ResponseWriter, r *http.Request, _ app.Viewer) error {
	writeJSON(w, http.StatusOK, map[string]string{"version": tandoorVersion})
	return nil
}

type user struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
}

func toUser(id int64, handle, display string) user {
	if display == "" {
		display = handle
	}
	return user{ID: id, Username: handle, FirstName: display, DisplayName: display}
}

// users returns only you, your household and your friends: never a list of everyone.
func (l *Layer) users(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	rows, err := l.svc.VisibleUsers(r.Context(), v)
	if err != nil {
		return err
	}
	out := make([]user, len(rows))
	for i, u := range rows {
		out[i] = toUser(u.ID, u.Handle, u.DisplayName)
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (l *Layer) preference(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	writeJSON(w, http.StatusOK, []map[string]any{{"user": toUser(v.User.ID, v.User.Handle, v.User.DisplayName), "default_unit": "g", "default_meal_type": nil}})
	return nil
}

type space struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Message   string `json:"message"`
	AIEnabled bool   `json:"ai_enabled"`
}

func (l *Layer) space(r *http.Request, v app.Viewer) (space, error) {
	h, err := l.svc.GetHousehold(r.Context(), v)
	if err != nil {
		return space{}, err
	}
	return space{ID: h.ID, Name: h.Name, CreatedAt: v.User.CreatedAt}, nil
}

// A Wokko household is what Tandoor calls a space. Each user has exactly one.
func (l *Layer) currentSpace(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	s, err := l.space(r, v)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, s)
	return nil
}

func (l *Layer) spaces(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	s, err := l.space(r, v)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, paged[space]{Count: 1, Results: []space{s}})
	return nil
}

func (l *Layer) emptyList(w http.ResponseWriter, r *http.Request, _ app.Viewer) error {
	writeJSON(w, http.StatusOK, paged[any]{Results: []any{}})
	return nil
}

// --- paging ---

type paged[T any] struct {
	Count    int64   `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []T     `json:"results"`
}

type pageParams struct{ page, size int64 }

func readPage(r *http.Request) pageParams {
	q := r.URL.Query()
	page, _ := strconv.ParseInt(q.Get("page"), 10, 64)
	size, _ := strconv.ParseInt(q.Get("page_size"), 10, 64)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 25
	}
	if size > 200 {
		size = 200
	}
	return pageParams{page, size}
}

func (p pageParams) offset() int64 { return (p.page - 1) * p.size }

// toPaged adds next and previous links, keeping the request's other parameters. They are
// relative to the current host, because reverse proxies can change what the app sees.
func toPaged[T any](r *http.Request, p pageParams, total int64, items []T) paged[T] {
	link := func(page int64) *string {
		q := r.URL.Query()
		q.Set("page", strconv.FormatInt(page, 10))
		u := url.URL{Scheme: scheme(r), Host: r.Host, Path: r.URL.Path, RawQuery: q.Encode()}
		s := u.String()
		return &s
	}
	out := paged[T]{Count: total, Results: items}
	if p.page*p.size < total {
		out.Next = link(p.page + 1)
	}
	if p.page > 1 {
		out.Previous = link(p.page - 1)
	}
	return out
}

func scheme(r *http.Request) string {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}

// --- recipes ---

type keywordOverview struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

type recipeOverview struct {
	ID           int64             `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Image        *string           `json:"image"`
	Keywords     []keywordOverview `json:"keywords"`
	WorkingTime  int64             `json:"working_time"`
	WaitingTime  int64             `json:"waiting_time"`
	CreatedBy    *user             `json:"created_by"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	Internal     bool              `json:"internal"`
	Servings     int64             `json:"servings"`
	ServingsText string            `json:"servings_text"`
	Rating       *float64          `json:"rating"`
	LastCooked   *string           `json:"last_cooked"`
	New          bool              `json:"new"`
	Recent       *string           `json:"recent"`
}

type keyword struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Numchild    int    `json:"numchild"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	FullName    string `json:"full_name"`
}

func toKeyword(k app.Keyword) keyword {
	return keyword{ID: k.ID, Name: k.Name, Label: k.Name, CreatedAt: k.CreatedAt, UpdatedAt: k.UpdatedAt, FullName: k.Name}
}

type unit struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	PluralName *string `json:"plural_name"`
}

type food struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	PluralName          *string   `json:"plural_name"`
	FullName            string    `json:"full_name"`
	IgnoreShopping      bool      `json:"ignore_shopping"`
	SupermarketCategory *category `json:"supermarket_category"`
}

type ingredient struct {
	ID       int64   `json:"id"`
	Food     food    `json:"food"`
	Unit     *unit   `json:"unit"`
	Amount   float64 `json:"amount"`
	Note     string  `json:"note"`
	Order    int     `json:"order"`
	IsHeader bool    `json:"is_header"`
	NoAmount bool    `json:"no_amount"`
}

type step struct {
	ID                 int64        `json:"id"`
	Name               string       `json:"name"`
	Instruction        string       `json:"instruction"`
	Ingredients        []ingredient `json:"ingredients"`
	InstructionsMD     string       `json:"instructions_markdown"`
	Time               int64        `json:"time"`
	Order              int          `json:"order"`
	ShowAsHeader       bool         `json:"show_as_header"`
	ShowIngredientsTbl bool         `json:"show_ingredients_table"`
}

type recipe struct {
	ID                     int64                   `json:"id"`
	Name                   string                  `json:"name"`
	Description            string                  `json:"description"`
	Image                  *string                 `json:"image"`
	Keywords               []keyword               `json:"keywords"`
	Steps                  []step                  `json:"steps"`
	WorkingTime            int64                   `json:"working_time"`
	WaitingTime            int64                   `json:"waiting_time"`
	CreatedBy              *user                   `json:"created_by"`
	CreatedAt              string                  `json:"created_at"`
	UpdatedAt              string                  `json:"updated_at"`
	SourceURL              *string                 `json:"source_url"`
	Internal               bool                    `json:"internal"`
	ShowIngredientOverview bool                    `json:"show_ingredient_overview"`
	Properties             []property              `json:"properties"`
	FoodProperties         map[string]foodProperty `json:"food_properties"`
	Servings               int64                   `json:"servings"`
	ServingsText           string                  `json:"servings_text"`
	Private                bool                    `json:"private"`
	Rating                 *float64                `json:"rating"`
	LastCooked             *string                 `json:"last_cooked"`
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func owner(p *app.Person) *user {
	if p == nil {
		return nil
	}
	u := toUser(p.ID, p.Handle, p.DisplayName)
	return &u
}

func toOverview(r app.RecipeSummary) recipeOverview {
	out := recipeOverview{
		ID: r.ID, Name: r.Name, Description: r.Description, Image: strPtr(r.PhotoURL), Keywords: make([]keywordOverview, len(r.Keywords)),
		WorkingTime: r.WorkMinutes, WaitingTime: r.WaitMinutes, CreatedBy: owner(r.Owner), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		Internal: true, Servings: r.Servings, ServingsText: r.ServingsText, Rating: r.Rating, LastCooked: strPtr(r.LastCooked),
	}
	for i, k := range r.Keywords {
		out.Keywords[i] = keywordOverview{k.ID, k.Name}
	}
	return out
}

func toUnit(u app.UnitRef) unit { return unit{ID: u.ID, Name: u.Name, PluralName: strPtr(u.Plural)} }

func toFood(f app.FoodRef) food {
	return food{ID: f.ID, Name: f.Name, PluralName: strPtr(f.Plural), FullName: f.Name}
}

func toStep(s app.Step, order int) step {
	out := step{ID: s.ID, Name: s.Name, Instruction: s.Instruction, InstructionsMD: s.Instruction, Time: s.Minutes, Order: order, ShowIngredientsTbl: true, Ingredients: make([]ingredient, len(s.Ingredients))}
	for i, ing := range s.Ingredients {
		item := ingredient{ID: ing.ID, Food: toFood(ing.Food), Note: ing.Note, Order: i, NoAmount: ing.Amount == nil}
		if ing.Amount != nil {
			item.Amount = *ing.Amount
		}
		if ing.Unit != nil {
			u := toUnit(*ing.Unit)
			item.Unit = &u
		}
		out.Ingredients[i] = item
	}
	return out
}

func toRecipe(r app.Recipe) recipe {
	out := recipe{
		ID: r.ID, Name: r.Name, Description: r.Description, Image: strPtr(r.PhotoURL), Keywords: make([]keyword, len(r.Keywords)), Steps: make([]step, len(r.Steps)),
		WorkingTime: r.WorkMinutes, WaitingTime: r.WaitMinutes, CreatedBy: owner(r.Owner), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, SourceURL: strPtr(r.SourceURL),
		Internal: true, ShowIngredientOverview: true, Properties: []property{}, FoodProperties: map[string]foodProperty{}, Servings: r.Servings, ServingsText: r.ServingsText,
		Private: r.Visibility == "private", Rating: r.Rating, LastCooked: strPtr(r.LastCooked),
	}
	for i, k := range r.Keywords {
		out.Keywords[i] = toKeyword(k)
	}
	for i, s := range r.Steps {
		out.Steps[i] = toStep(s, i)
	}
	return out
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, app.ErrNotFound
	}
	return id, nil
}

// listRecipes supports search, keyword filter, random and the common sort orders. Other
// Tandoor filters aren't supported yet and are ignored.
func (l *Layer) listRecipes(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p := readPage(r)
	q := r.URL.Query()
	f := app.RecipeQuery{Q: q.Get("query"), Limit: p.size, Offset: p.offset()}
	f.AnyKeyword = idList(q, "keywords", "keywords_or", "keywords_and")
	f.AnyFood = idList(q, "foods", "foods_or", "foods_and")
	f.AnyBook = idList(q, "books", "books_or", "books_and")
	if id, err := strconv.ParseInt(q.Get("createdby"), 10, 64); err == nil {
		f.Owner = id
	}
	number := func(name string) int64 {
		n, _ := strconv.ParseInt(q.Get(name), 10, 64) // a missing or odd value means no limit
		return n
	}
	f.RatingMin, f.RatingMax = number("rating_gte"), number("rating_lte")
	f.CookedMin, f.CookedMax = number("timescooked_gte"), number("timescooked_lte")
	if n := number("rating"); n > 0 {
		f.RatingMin, f.RatingMax = n, n
	}
	if n := number("timescooked"); n > 0 {
		f.CookedMin, f.CookedMax = n, n
	}
	if q.Get("random") == "true" {
		f.Sort = "random"
	} else {
		f.Sort = map[string]string{
			"name": "name", "-name": "-name", "created_at": "-created", "-created_at": "created",
			"rating": "rating", "-rating": "-rating", "favorite": "times", "-favorite": "-times",
			"lastcooked": "cooked", "-lastcooked": "-cooked",
		}[q.Get("sort_order")]
	}
	res, err := l.svc.ListRecipes(r.Context(), v, f)
	if err != nil {
		return err
	}
	items := make([]recipeOverview, len(res.Items))
	for i, it := range res.Items {
		items[i] = toOverview(it)
	}
	writeJSON(w, http.StatusOK, toPaged(r, p, res.Total, items))
	return nil
}

func (l *Layer) getRecipe(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	rec, err := l.svc.GetRecipe(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l.fullRecipe(r, v, rec))
	return nil
}

func (l *Layer) getStep(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	st, order, err := l.svc.GetStep(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toStep(st, order))
	return nil
}

// --- catalog ---

func (l *Layer) listFoods(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p := readPage(r)
	res, err := l.svc.ListFoods(r.Context(), v, r.URL.Query().Get("query"), p.size, p.offset())
	if err != nil {
		return err
	}
	items := make([]food, len(res.Items))
	for i, f := range res.Items {
		items[i] = l.foodFor(r, v, f.FoodRef)
	}
	writeJSON(w, http.StatusOK, toPaged(r, p, res.Total, items))
	return nil
}

func (l *Layer) getFood(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	f, err := l.svc.GetFood(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, l.foodFor(r, v, f.FoodRef))
	return nil
}

func (l *Layer) listUnits(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p := readPage(r)
	res, err := l.svc.ListUnits(r.Context(), v, r.URL.Query().Get("query"), p.size, p.offset())
	if err != nil {
		return err
	}
	items := make([]unit, len(res.Items))
	for i, u := range res.Items {
		items[i] = toUnit(u.UnitRef)
	}
	writeJSON(w, http.StatusOK, toPaged(r, p, res.Total, items))
	return nil
}

func (l *Layer) getUnit(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	u, err := l.svc.GetUnit(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toUnit(u.UnitRef))
	return nil
}

func (l *Layer) listKeywords(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p := readPage(r)
	res, err := l.svc.ListKeywords(r.Context(), v, r.URL.Query().Get("query"), p.size, p.offset())
	if err != nil {
		return err
	}
	items := make([]keyword, len(res.Items))
	for i, k := range res.Items {
		items[i] = toKeyword(k)
	}
	writeJSON(w, http.StatusOK, toPaged(r, p, res.Total, items))
	return nil
}

func (l *Layer) getKeyword(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	k, err := l.svc.GetKeyword(r.Context(), v, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toKeyword(k))
	return nil
}

// --- cook log ---

type cookLog struct {
	ID        int64   `json:"id"`
	Recipe    int64   `json:"recipe"`
	Servings  *int64  `json:"servings"`
	Rating    *int64  `json:"rating"`
	Comment   *string `json:"comment"`
	CreatedAt string  `json:"created_at"`
	CreatedBy user    `json:"created_by"`
	UpdatedAt string  `json:"updated_at"`
}

func toCookLog(e app.CookLogEntry, recipe int64, u *sqlcgen.User) cookLog {
	return cookLog{ID: e.ID, Recipe: recipe, Servings: e.Servings, Rating: e.Rating, Comment: strPtr(e.Comment), CreatedAt: e.CreatedAt, UpdatedAt: e.CreatedAt, CreatedBy: toUser(u.ID, u.Handle, u.DisplayName)}
}

func (l *Layer) listCookLog(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	p := readPage(r)
	id, err := strconv.ParseInt(r.URL.Query().Get("recipe"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusOK, paged[cookLog]{Results: []cookLog{}})
		return nil
	}
	rows, err := l.svc.ListCookLog(r.Context(), v, id)
	if err != nil {
		return err
	}
	total := int64(len(rows))
	start := min(p.offset(), total)
	end := min(start+p.size, total)
	items := make([]cookLog, 0, end-start)
	for _, e := range rows[start:end] {
		items = append(items, toCookLog(e, id, v.User))
	}
	writeJSON(w, http.StatusOK, toPaged(r, p, total, items))
	return nil
}

func (l *Layer) addCookLog(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		Recipe   int64  `json:"recipe"`
		Servings *int64 `json:"servings"`
		Rating   *int64 `json:"rating"`
		Comment  string `json:"comment"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		return &app.Error{Status: http.StatusBadRequest, Code: "bad_request", Message: "The request could not be read"}
	}
	e, err := l.svc.AddCookLog(r.Context(), v, in.Recipe, app.CookLogInput{Servings: in.Servings, Rating: in.Rating, Comment: in.Comment})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toCookLog(e, in.Recipe, v.User))
	return nil
}
