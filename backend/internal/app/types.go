package app

// These types are the JSON contract of the web API. The frontend's TypeScript types are
// generated from this file (see `make types`), so the two can't drift apart.

type User struct {
	ID          int64  `json:"id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Locale      string `json:"locale"`
	HouseholdID int64  `json:"household_id"`
	Disabled    bool   `json:"disabled"`
}

type Person struct {
	ID          int64  `json:"id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
}

type Status struct {
	SetupComplete    bool   `json:"setup_complete"`
	OpenRegistration bool   `json:"open_registration"`
	AllowPublic      bool   `json:"allow_public"`
	MCPEnabled       bool   `json:"mcp_enabled"`
	Version          string `json:"version"`
}

type Translation struct {
	Locale       string `json:"locale"`
	Name         string `json:"name"`
	Plural       string `json:"plural"`
	Abbreviation string `json:"abbreviation"`
}

type UnitRef struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Plural       string `json:"plural"`
	Abbreviation string `json:"abbreviation"`
}

type FoodRef struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Plural string `json:"plural"`
}

type Food struct {
	FoodRef      `json:",inline" tstype:",extends"`
	DefaultUnit  *UnitRef      `json:"default_unit"`
	Translations []Translation `json:"translations"`
	Aliases      []string      `json:"aliases"`
	Uses         int64         `json:"uses"`
	CanEdit      bool          `json:"can_edit"`
}

type Unit struct {
	UnitRef      `json:",inline" tstype:",extends"`
	Translations []Translation `json:"translations"`
	Aliases      []string      `json:"aliases"`
	Uses         int64         `json:"uses"`
	CanEdit      bool          `json:"can_edit"`
}

type FoodInput struct {
	Translations  []Translation `json:"translations"`
	DefaultUnitID *int64        `json:"default_unit_id"`
	Aliases       []string      `json:"aliases"`
}

type UnitInput struct {
	Translations []Translation `json:"translations"`
	Aliases      []string      `json:"aliases"`
}

type MergeInput struct {
	SourceID int64 `json:"source_id"`
	TargetID int64 `json:"target_id"`
}

type Keyword struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Page[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

type Ingredient struct {
	ID     int64    `json:"id"`
	Food   FoodRef  `json:"food"`
	Unit   *UnitRef `json:"unit"`
	Amount *float64 `json:"amount"`
	Note   string   `json:"note"`
}

type Step struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Instruction string       `json:"instruction"`
	Minutes     int64        `json:"minutes"`
	Ingredients []Ingredient `json:"ingredients"`
}

type RecipeSummary struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	WorkMinutes  int64     `json:"work_minutes"`
	WaitMinutes  int64     `json:"wait_minutes"`
	Servings     int64     `json:"servings"`
	ServingsText string    `json:"servings_text"`
	Visibility   string    `json:"visibility"`
	PhotoURL     string    `json:"photo_url"`
	ThumbURL     string    `json:"thumb_url"`
	Keywords     []Keyword `json:"keywords"`
	Owner        *Person   `json:"owner"`
	Rating       *float64  `json:"rating"`
	LastCooked   string    `json:"last_cooked"`
	TimesCooked  int64     `json:"times_cooked"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
	DeletedAt    string    `json:"deleted_at"`
}

type Recipe struct {
	RecipeSummary `json:",inline" tstype:",extends"`
	SourceURL     string `json:"source_url"`
	Version       int64  `json:"version"`
	CanEdit       bool   `json:"can_edit"`
	CanManage     bool   `json:"can_manage"`
	Steps         []Step `json:"steps"`
}

type IngredientInput struct {
	FoodID   int64    `json:"food_id"`
	FoodName string   `json:"food_name"`
	UnitID   int64    `json:"unit_id"`
	UnitName string   `json:"unit_name"`
	Amount   *float64 `json:"amount"`
	Note     string   `json:"note"`
}

type StepInput struct {
	Name        string            `json:"name"`
	Instruction string            `json:"instruction"`
	Minutes     int64             `json:"minutes"`
	Ingredients []IngredientInput `json:"ingredients"`
}

// RecipeInput is one complete recipe. Saving it is a single transaction.
type RecipeInput struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	WorkMinutes  int64       `json:"work_minutes"`
	WaitMinutes  int64       `json:"wait_minutes"`
	Servings     int64       `json:"servings"`
	ServingsText string      `json:"servings_text"`
	SourceURL    string      `json:"source_url"`
	Visibility   string      `json:"visibility"`
	Keywords     []string    `json:"keywords"`
	Steps        []StepInput `json:"steps"`
	// BaseVersion is the version this edit started from, used to detect conflicting edits.
	BaseVersion int64 `json:"base_version"`
}

type RecipeVersion struct {
	ID        int64  `json:"id"`
	Version   int64  `json:"version"`
	CreatedAt string `json:"created_at"`
}

type ShareLink struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

type Sharing struct {
	Users []Person    `json:"users"`
	Links []ShareLink `json:"links"`
}

type CookLogEntry struct {
	ID        int64  `json:"id"`
	Servings  *int64 `json:"servings"`
	Rating    *int64 `json:"rating"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"created_at"`
}

type CookLogInput struct {
	Servings *int64 `json:"servings"`
	Rating   *int64 `json:"rating"`
	Comment  string `json:"comment"`
}

type Friend struct {
	ID       int64  `json:"id"`
	Person   Person `json:"person"`
	Accepted bool   `json:"accepted"`
	Incoming bool   `json:"incoming"`
}

type Household struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Members []Person `json:"members"`
	Invites []Invite `json:"invites"`
}

// Invite carries Token only in the response that created it, since only a hash is stored.
type Invite struct {
	ID        int64  `json:"id"`
	Token     string `json:"token,omitempty"`
	MaxUses   int64  `json:"max_uses"`
	UseCount  int64  `json:"use_count"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

type InviteInput struct {
	MaxUses       int64 `json:"max_uses"`
	ExpiresInDays int64 `json:"expires_in_days"`
}

type Settings struct {
	OpenRegistration bool `json:"open_registration"`
	AllowPublic      bool `json:"allow_public"`
	MCPEnabled       bool `json:"mcp_enabled"`
}

type BackupRun struct {
	ID         int64  `json:"id"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Status     string `json:"status"`
	File       string `json:"file"`
	SizeBytes  int64  `json:"size_bytes"`
	Error      string `json:"error"`
}

// Health is everything an admin needs to know the data is safe.
type Health struct {
	Version         string      `json:"version"`
	SchemaVersion   int         `json:"schema_version"`
	Recipes         int64       `json:"recipes"`
	Foods           int64       `json:"foods"`
	Units           int64       `json:"units"`
	Users           int64       `json:"users"`
	LastBackup      *BackupRun  `json:"last_backup"`
	LastSuccess     *BackupRun  `json:"last_success"`
	BackupOverdue   bool        `json:"backup_overdue"`
	BackupFailing   bool        `json:"backup_failing"`
	RecentBackups   []BackupRun `json:"recent_backups"`
	DataFreeBytes   int64       `json:"data_free_bytes"`
	BackupFreeBytes int64       `json:"backup_free_bytes"`
	LowDisk         bool        `json:"low_disk"`
	IntegrityOK     bool        `json:"integrity_ok"`
	IntegrityError  string      `json:"integrity_error"`
	CheckedAt       string      `json:"checked_at"`
	MissingPhotos   int64       `json:"missing_photos"`
}

// ImportRequest asks for a recipe draft from a web address or from pasted text.
type ImportRequest struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}

// ImportedRecipe is a draft for the editor. Nothing has been saved yet.
type ImportedRecipe struct {
	Recipe RecipeInput `json:"recipe"`
	// ImageData is the source's photo in base64. The editor uploads it like any other photo
	// when the person saves, so it passes through the normal checks.
	ImageData string `json:"image_data"`
	ImageType string `json:"image_type"`
	// Guessed means the result was read from the page's layout, not recipe data the site published.
	Guessed bool `json:"guessed"`
}

// RecipeBook is a named collection of recipes, visible to its owner's household.
type RecipeBook struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RecipeCount int64  `json:"recipe_count"`
	CanEdit     bool   `json:"can_edit"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RecipeBookDetail struct {
	RecipeBook `json:",inline" tstype:",extends"`
	Recipes    []RecipeSummary `json:"recipes"`
}

type RecipeBookInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CollectionLink shares everything its owner hasn't kept private.
type CollectionLink struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// MealType is a slot in the plan, such as breakfast or dinner.
type MealType struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position int64  `json:"position"`
	Time     string `json:"time"`
	Color    string `json:"color"`
}

type MealTypeInput struct {
	Name     string `json:"name"`
	Position int64  `json:"position"`
	Time     string `json:"time"`
	Color    string `json:"color"`
}

// MealPlan is one planned meal for the household. RecipeHidden means it points at a recipe
// this person isn't allowed to see, so only that fact is shown.
type MealPlan struct {
	ID           int64          `json:"id"`
	Title        string         `json:"title"`
	Note         string         `json:"note"`
	Servings     float64        `json:"servings"`
	FromDate     string         `json:"from_date"`
	ToDate       string         `json:"to_date"`
	MealType     MealType       `json:"meal_type"`
	Recipe       *RecipeSummary `json:"recipe"`
	RecipeHidden bool           `json:"recipe_hidden"`
	CreatedBy    *Person        `json:"created_by"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}

type MealPlanInput struct {
	Title         string  `json:"title"`
	Note          string  `json:"note"`
	Servings      float64 `json:"servings"`
	FromDate      string  `json:"from_date"`
	ToDate        string  `json:"to_date"`
	MealTypeID    int64   `json:"meal_type_id"`
	RecipeID      int64   `json:"recipe_id"`
	AddToShopping bool    `json:"add_to_shopping"`
}

type SupermarketCategory struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CategoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SupermarketOrderEntry struct {
	ID       int64               `json:"id"`
	Category SupermarketCategory `json:"category"`
	Position int64               `json:"position"`
}

type Supermarket struct {
	ID          int64                   `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Categories  []SupermarketOrderEntry `json:"categories"`
}

// ShoppingRecipe says which recipe and servings a group of shopping items came from.
type ShoppingRecipe struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Servings   float64 `json:"servings"`
	RecipeID   int64   `json:"recipe_id"`
	MealPlanID int64   `json:"meal_plan_id"`
}

type ShoppingItem struct {
	ID        int64                `json:"id"`
	Food      FoodRef              `json:"food"`
	Unit      *UnitRef             `json:"unit"`
	Amount    *float64             `json:"amount"`
	Note      string               `json:"note"`
	Checked   bool                 `json:"checked"`
	Position  int64                `json:"position"`
	Category  *SupermarketCategory `json:"category"`
	Recipe    *ShoppingRecipe      `json:"recipe"`
	CreatedBy *Person              `json:"created_by"`
	CreatedAt string               `json:"created_at"`
	UpdatedAt string               `json:"updated_at"`
}

type ShoppingItemInput struct {
	// Line is free text such as "2 kg potatoes", read like an imported ingredient.
	Line     string   `json:"line"`
	FoodID   int64    `json:"food_id"`
	FoodName string   `json:"food_name"`
	UnitID   int64    `json:"unit_id"`
	UnitName string   `json:"unit_name"`
	Amount   *float64 `json:"amount"`
	Note     string   `json:"note"`
	Checked  bool     `json:"checked"`
}

// ShoppingAddRecipe adds a recipe's ingredients, scaled to the servings. IngredientIDs narrows
// that to some of them.
type ShoppingAddRecipe struct {
	RecipeID      int64   `json:"recipe_id"`
	Servings      float64 `json:"servings"`
	MealPlanID    int64   `json:"meal_plan_id"`
	IngredientIDs []int64 `json:"ingredient_ids"`
}

type ShoppingCheckInput struct {
	IDs     []int64 `json:"ids"`
	Checked bool    `json:"checked"`
}

type Nutrient struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Unit     string `json:"unit"`
	Position int64  `json:"position"`
}

// FoodNutrition is per 100 g of a food. PieceGrams is what one piece weighs.
type FoodNutrition struct {
	PieceGrams *float64          `json:"piece_grams"`
	Values     map[int64]float64 `json:"values"`
}

type RecipeNutrient struct {
	Nutrient   Nutrient `json:"nutrient"`
	Total      float64  `json:"total"`
	PerServing float64  `json:"per_serving"`
}

// RecipeNutrition adds up what's known. Missing lists ingredients it couldn't count, so a
// partial result is never mistaken for a complete one.
type RecipeNutrition struct {
	Nutrients []RecipeNutrient `json:"nutrients"`
	Counted   int64            `json:"counted"`
	Total     int64            `json:"total"`
	Missing   []string         `json:"missing"`
}

type Passkey struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
}

// APIToken lets an AI assistant act for its owner through MCP.
type APIToken struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Scope      string `json:"scope"`
	IsApp      bool   `json:"is_app"`
	ExpiresAt  string `json:"expires_at"`
	LastUsedAt string `json:"last_used_at"`
	CreatedAt  string `json:"created_at"`
}

type APITokenInput struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`
	ExpiresInDays int64  `json:"expires_in_days"`
}

// NewAPIToken holds the secret, which is shown only this once.
type NewAPIToken struct {
	APIToken `json:",inline" tstype:",extends"`
	Secret   string `json:"secret"`
}

type AssistantActivity struct {
	ID        int64  `json:"id"`
	TokenName string `json:"token_name"`
	Tool      string `json:"tool"`
	Target    string `json:"target"`
	CreatedAt string `json:"created_at"`
}

// OAuthRequest describes an app asking for access, for the consent page.
type OAuthRequest struct {
	ClientName   string `json:"client_name"`
	RedirectHost string `json:"redirect_host"`
	Scope        string `json:"scope"`
}

type OAuthApproval struct {
	ClientID      string `json:"client_id"`
	RedirectURI   string `json:"redirect_uri"`
	CodeChallenge string `json:"code_challenge"`
	Scope         string `json:"scope"`
	State         string `json:"state"`
}

type OAuthRedirect struct {
	RedirectURL string `json:"redirect_url"`
}
