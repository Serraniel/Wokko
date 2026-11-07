package api_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/backup"
	"github.com/serraniel/wokko/backend/internal/store"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// SQLite doesn't enforce foreign keys unless every connection asks for it.
func TestForeignKeysAreEnforcedOnEveryConnection(t *testing.T) {
	e := newEnv(t)
	r, w, err := e.svc.Store.ForeignKeysOn(context.Background())
	if err != nil || !r || !w {
		t.Fatalf("foreign keys: reader=%v writer=%v err=%v", r, w, err)
	}
}

func TestCatalogEntriesInUseCannotBeRemovedByTheDatabase(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	rec := c.createRecipe(simpleRecipe("Soup"))
	food := rec.Steps[0].Ingredients[0].Food.ID
	err := e.svc.Store.Tx(context.Background(), func(q *sqlcgen.Queries) error { return q.DeleteFood(context.Background(), food) })
	if err == nil {
		t.Fatal("the schema must refuse to delete a food that a recipe uses")
	}
	var got app.Recipe
	c.ok("GET", path("/recipes/%d", rec.ID), nil, 200).decode(t, &got)
	if got.Steps[0].Ingredients[0].Food.ID != food {
		t.Fatal("recipe lost its ingredient")
	}
}

func TestBackupAndRestoreRoundTrip(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	rec := c.createRecipe(simpleRecipe("Precious soup"))
	if res := c.upload(rec.ID, testPNG(t)); res.status != 200 {
		t.Fatal(string(res.body))
	}

	// Everything goes through the real endpoint, while the app is running.
	var run app.BackupRun
	c.ok("POST", v1+"/admin/backups", nil, 201).decode(t, &run)
	if run.Status != "ok" || run.File == "" || run.SizeBytes == 0 {
		t.Fatalf("backup: %+v", run)
	}
	archive := filepath.Join(e.cfg.BackupDir, run.File)
	if err := backup.Verify(archive); err != nil {
		t.Fatal(err)
	}

	// Disaster: restore into a new data directory and read the recipe back.
	dir := filepath.Join(t.TempDir(), "restored")
	if _, err := backup.Restore(archive, dir, time.Now()); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "wokko.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	row, err := st.Queries.GetRecipe(context.Background(), rec.ID)
	if err != nil || row.Name != "Precious soup" {
		t.Fatalf("restored recipe: %+v %v", row, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "uploads", row.Photo.String)); err != nil {
		t.Fatalf("restored photo: %v", err)
	}
}

func TestRestoreKeepsCurrentDataAside(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	var run app.BackupRun
	c.ok("POST", v1+"/admin/backups", nil, 201).decode(t, &run)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wokko.db"), []byte("existing data"), 0o640); err != nil {
		t.Fatal(err)
	}
	kept, err := backup.Restore(filepath.Join(e.cfg.BackupDir, run.File), dir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(kept, "wokko.db"))
	if err != nil || string(b) != "existing data" {
		t.Fatalf("previous data must be kept: %q %v", b, err)
	}
}

func TestDamagedBackupIsDetectedAndRestoreChangesNothing(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	var run app.BackupRun
	c.ok("POST", v1+"/admin/backups", nil, 201).decode(t, &run)
	archive := filepath.Join(e.cfg.BackupDir, run.File)
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	damaged := filepath.Join(t.TempDir(), "damaged.tar.gz")
	if err := os.WriteFile(damaged, data[:len(data)/2], 0o640); err != nil {
		t.Fatal(err)
	}
	if err := backup.Verify(damaged); err == nil {
		t.Fatal("a cut-off backup must fail verification")
	}
	dir := t.TempDir()
	current := filepath.Join(dir, "wokko.db")
	if err := os.WriteFile(current, []byte("current"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.Restore(damaged, dir, time.Now()); err == nil {
		t.Fatal("restoring a damaged backup must fail")
	}
	if b, err := os.ReadFile(current); err != nil || string(b) != "current" {
		t.Fatalf("failed restore touched current data: %q %v", b, err)
	}
}

func TestPruneKeepsNewestEvenAfterALongGap(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().AddDate(0, 0, -100)
	names := []string{"2020-01-01", "2020-01-02", "2020-01-03", "2020-01-04", "2020-01-05"}
	for _, n := range names {
		p := filepath.Join(dir, "wokko-backup-"+n+".tar.gz")
		if err := os.WriteFile(p, []byte("x"), 0o640); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if err := backup.Prune(dir, 30, time.Now()); err != nil {
		t.Fatal(err)
	}
	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != backup.KeepAtLeast || !strings.Contains(left[0].Name(), "2020-01-03") {
		t.Fatalf("left: %v", left)
	}
}

// The test case from real life: Tandoor's backups silently stopped for 12 days.
func TestStoppedBackupsAreReportedToAdmins(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	ctx := context.Background()

	var h app.Health
	c.ok("GET", v1+"/admin/health", nil, 200).decode(t, &h)
	if !h.BackupOverdue {
		t.Fatal("no backup at all must count as overdue")
	}
	if due, err := e.svc.BackupDue(ctx); err != nil || !due {
		t.Fatalf("a first backup should be due: %v %v", due, err)
	}

	c.ok("POST", v1+"/admin/backups", nil, 201)
	c.ok("GET", v1+"/admin/health", nil, 200).decode(t, &h)
	if h.BackupOverdue || h.BackupFailing || h.LastSuccess == nil || len(h.RecentBackups) != 1 {
		t.Fatalf("fresh backup: %+v", h)
	}
	if due, err := e.svc.BackupDue(ctx); err != nil || due {
		t.Fatalf("no backup should be due right after one: %v %v", due, err)
	}

	real := e.svc.Now
	e.svc.Now = func() time.Time { return real().Add(72 * time.Hour) }
	c.ok("GET", v1+"/admin/health", nil, 200).decode(t, &h)
	if !h.BackupOverdue {
		t.Fatal("a backup three days old must be reported as overdue")
	}
	if due, err := e.svc.BackupDue(ctx); err != nil || !due {
		t.Fatalf("a backup should be due: %v %v", due, err)
	}
	e.svc.Now = real
}

func TestFailingBackupIsRecordedAndReported(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	// A file where the backup folder should be, like a NAS mount gone wrong.
	blocked := filepath.Join(t.TempDir(), "not-a-folder")
	if err := os.WriteFile(blocked, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	e.svc.Cfg.BackupDir = blocked
	c.wantCode("POST", v1+"/admin/backups", nil, 500, "backup_failed")
	var h app.Health
	c.ok("GET", v1+"/admin/health", nil, 200).decode(t, &h)
	if !h.BackupFailing || h.LastBackup == nil || h.LastBackup.Error == "" {
		t.Fatalf("failure must be visible: %+v", h)
	}
	c.ok("GET", v1+"/me", nil, 200) // the app keeps working
}

func TestIntegrityAndMissingPhotosAreReported(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	rec := c.createRecipe(simpleRecipe("Soup"))
	var withPhoto app.Recipe
	c.upload(rec.ID, testPNG(t)).decode(t, &withPhoto)
	e.svc.RunChecks(context.Background())
	var h app.Health
	c.ok("GET", v1+"/admin/health", nil, 200).decode(t, &h)
	if !h.IntegrityOK || h.MissingPhotos != 0 || h.CheckedAt == "" {
		t.Fatalf("clean: %+v", h)
	}
	if err := os.Remove(filepath.Join(e.cfg.UploadsDir(), filepath.Base(withPhoto.PhotoURL))); err != nil {
		t.Fatal(err)
	}
	e.svc.RunChecks(context.Background())
	c.ok("GET", v1+"/admin/health", nil, 200).decode(t, &h)
	if h.MissingPhotos != 1 {
		t.Fatalf("missing photo not reported: %+v", h)
	}
}

func TestDatabaseNewerThanTheAppIsRefused(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "wokko.db")
	st, err := store.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.Migrate(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := st.ForceSchemaVersionForTest(ctx, 9999); err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(ctx, nil); !errors.Is(err, store.ErrDatabaseTooNew) {
		t.Fatalf("got %v", err)
	}
}

func TestUpgradeStopsIfTheSafetyBackupFails(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "wokko.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.Migrate(ctx, nil); err != nil {
		t.Fatal(err)
	}
	before, err := st.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ForceSchemaVersionForTest(ctx, 1); err != nil {
		t.Fatal(err)
	}
	called := false
	err = st.Migrate(ctx, func() error { called = true; return errors.New("disk full") })
	if !called || err == nil || !strings.Contains(err.Error(), "database left unchanged") {
		t.Fatalf("called=%v err=%v", called, err)
	}
	after, err := st.SchemaVersion(ctx)
	if err != nil || after != 1 || before < 2 {
		t.Fatalf("schema moved: before=%d after=%d err=%v", before, after, err)
	}
}

func TestSeedsRunOnceSoUserChangesSurviveRestarts(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "wokko.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.Migrate(ctx, nil); err != nil {
		t.Fatal(err)
	}
	err = st.Tx(ctx, func(q *sqlcgen.Queries) error {
		return q.UpsertFoodName(ctx, sqlcgen.UpsertFoodNameParams{FoodID: 1, Locale: "en", Name: "my onion", SearchName: "my onion"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(ctx, nil); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Queries.ListFoodNames(ctx, []int64{1})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Locale == "en" && r.Name != "my onion" {
			t.Fatalf("seed overwrote a user change: %q", r.Name)
		}
	}
}

func TestSecondProcessCannotOpenSameData(t *testing.T) {
	dir := t.TempDir()
	first, err := store.Lock(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := store.Lock(dir); err == nil {
		t.Fatal("second lock must fail")
	}
	if _, err := backup.Restore("unused", dir, time.Now()); err == nil || !strings.Contains(err.Error(), "stop Wokko") {
		t.Fatalf("restore must refuse while running: %v", err)
	}
}

func TestLocalDiskIsAccepted(t *testing.T) {
	if err := store.CheckLocalFS(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

// Set WOKKO_TEST_NETWORK_DIR to a mounted network share to check the refusal for real.
func TestNetworkShareIsRefused(t *testing.T) {
	dir := os.Getenv("WOKKO_TEST_NETWORK_DIR")
	if dir == "" {
		t.Skip("WOKKO_TEST_NETWORK_DIR not set")
	}
	if err := store.CheckLocalFS(dir); err == nil {
		t.Fatal("a network share must be refused")
	}
}

func TestExpiredTrashIsPurgedButRecentTrashIsKept(t *testing.T) {
	e := newEnv(t)
	c := e.admin()
	r := c.createRecipe(simpleRecipe("Soup"))
	c.ok("DELETE", path("/recipes/%d", r.ID), nil, 200)
	ctx := context.Background()
	if err := e.svc.PurgeExpiredTrash(ctx); err != nil {
		t.Fatal(err)
	}
	var trash []app.RecipeSummary
	c.ok("GET", v1+"/trash", nil, 200).decode(t, &trash)
	if len(trash) != 1 {
		t.Fatal("recent trash must stay")
	}
	real := e.svc.Now
	e.svc.Now = func() time.Time { return real().AddDate(0, 0, 31) }
	defer func() { e.svc.Now = real }()
	if err := e.svc.PurgeExpiredTrash(ctx); err != nil {
		t.Fatal(err)
	}
	c.ok("GET", v1+"/trash", nil, 200).decode(t, &trash)
	if len(trash) != 0 {
		t.Fatal("expired trash must be removed")
	}
}
