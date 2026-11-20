package test

import (
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/core"
	"github.com/artalkjs/artalk/v2/internal/dao"
	db_logger "github.com/artalkjs/artalk/v2/internal/db/logger"
	"github.com/artalkjs/artalk/v2/internal/pkged"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type TestApp struct {
	*core.App
	dbDir string
}

func (t *TestApp) Cleanup() error {
	if err := t.ResetBootstrapState(); err != nil {
		return err
	}

	// Every app owns its own directory, so removing it cannot race with a test
	// that is already opening the next database.
	if t.dbDir != "" {
		return os.RemoveAll(t.dbDir)
	}
	return nil
}

func NewTestApp() (*TestApp, error) {
	// change WorkDir run on project root
	{
		_, filename, _, _ := runtime.Caller(0)
		rootDir := path.Join(path.Dir(filename), "..")
		if err := os.Chdir(rootDir); err != nil {
			panic(err)
		}
	}

	// load assets fs
	dirFS := os.DirFS("./")
	pkged.SetFS(dirFS)

	// Each app gets its own database directory. A single shared temp file meant
	// every test recreated the previous test's path right after Cleanup removed
	// it, so a failure to close or to drop the -wal/-shm sidecars leaked into the
	// next database.
	dbDir, err := os.MkdirTemp("", "atk_test_db")
	if err != nil {
		return nil, err
	}
	cleanupDir := func() { _ = os.RemoveAll(dbDir) }

	// open a sqlite db
	dbInstance, err := gorm.Open(sqlite.Open(filepath.Join(dbDir, "atk_test.db")), &gorm.Config{
		Logger: db_logger.New(),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "atk_", // Test table prefix, fixture filenames should match this
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		cleanupDir()
		return nil, err
	}

	// cerate dao instance (will migrate database table)
	dao := dao.NewDao(dbInstance)

	// get pure sql.DB instance
	sqlDB, err := dbInstance.DB()
	if err != nil {
		cleanupDir()
		return nil, err
	}

	// fixtures config
	fixtures, err := testfixtures.New(
		testfixtures.Database(sqlDB),              // Database connection
		testfixtures.Dialect("sqlite"),            // Available: "postgresql", "timescaledb", "mysql", "mariadb", "sqlite" and "sqlserver"
		testfixtures.Directory("./test/fixtures"), // The directory containing the YAML files
	)
	if err != nil {
		cleanupDir()
		return nil, err
	}

	// load fixtures data
	if err := fixtures.Load(); err != nil {
		cleanupDir()
		return nil, err
	}

	// create config instance
	conf, err := config.NewFromFile("./test/testdata/model_test_conf.yml")
	if err != nil {
		cleanupDir()
		return nil, err
	}

	// create app instance
	app := core.NewApp(conf)

	// set dao
	app.SetDao(dao)

	// bootstrap
	if err := app.Bootstrap(); err != nil {
		cleanupDir()
		return nil, err
	}

	// create test app instance
	t := &TestApp{
		App:   app,
		dbDir: dbDir,
	}

	return t, nil
}
