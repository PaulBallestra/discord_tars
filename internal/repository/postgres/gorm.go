package postgres

import (
	"fmt"
	"time"

	"discord-tars/internal/config"
	"discord-tars/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormDB wraps the GORM DB instance
type GormDB struct {
	*gorm.DB
}

// NewGormConnection establishes a connection to PostgreSQL using GORM
func NewGormConnection(cfg config.DatabaseConfig) (*GormDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		// Disable foreign key constraints when migrating to avoid errors
		// with existing schema from SQL initialization scripts
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate models
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &GormDB{DB: db}, nil
}

// Close closes the database connection
func (db *GormDB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// autoMigrate automatically migrates the database schema
func autoMigrate(db *gorm.DB) error {
	// Enable pgvector extension
	db.Exec("CREATE EXTENSION IF NOT EXISTS vector")

	// Auto-migrate models
	return db.AutoMigrate(
		&models.Guild{},
		&models.Channel{},
		&models.User{},
		&models.Message{},
		&models.MessageEmbedding{},
	)
}
