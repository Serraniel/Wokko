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
