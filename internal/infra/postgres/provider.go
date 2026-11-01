package postgres

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mohsin123321/notification-system/internal/platform/config"
)

var migrationModels []any

func RegisterModels(models ...any) {
	migrationModels = append(migrationModels, models...)
}

// Postgres is the database adapter implementing lifecycle.Component.
type Postgres struct {
	db  *gorm.DB
	cfg config.PostgresConfig
}

func New(cfg *config.Config) *Postgres {
	return &Postgres{cfg: cfg.Postgres}
}

func (p *Postgres) Name() string { return "postgres" }

func (p *Postgres) Start() error {
	db, err := gorm.Open(postgres.Open(p.cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("postgres raw db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	p.db = db
	log.Printf("postgres connected to %s:%d/%s", p.cfg.Host, p.cfg.Port, p.cfg.Name)

	if p.cfg.AutoMigrate && len(migrationModels) > 0 {
		if err := db.AutoMigrate(migrationModels...); err != nil {
			return fmt.Errorf("automigrate: %w", err)
		}
		log.Printf("postgres automigrate complete (%d models)", len(migrationModels))
	}
	return nil
}

func (p *Postgres) Stop() error {
	if p.db == nil {
		return nil
	}
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (p *Postgres) DB() *gorm.DB {
	return p.db
}
