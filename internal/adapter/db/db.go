package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Init(dbURL string) (*gorm.DB, error) {

	if dbURL == "" {
		return nil, errors.New("invalid database url")
	}

	// Initialize DB session
	dbConn, err := gorm.Open(
		postgres.Open(dbURL),
		defaultGormConfig(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return dbConn, nil
}

// defaultGormConfig returns a default GORM configuration
// with default settings suited for this backend project
func defaultGormConfig() *gorm.Config {
	return &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel:      logger.Silent, // log level
				Colorful:      true,          // enable colored output
				SlowThreshold: time.Second * 5,
				// ^ threshold for what should be considered a slow query
			},
		),
		FullSaveAssociations: true,
	}
}
