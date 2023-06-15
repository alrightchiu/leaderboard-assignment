package db

import (
	"fmt"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB() (db *gorm.DB, err error) {
	// TODO: move to config?
	// host: postgresql-db or localhost
	dsn := "host=localhost user=postgres password=somesecret dbname=leaderboard port=5432 sslmode=disable TimeZone=Asia/Taipei"

	connectDB := func() error {
		curr, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return err
		}

		db = curr

		sql, err := db.DB()
		if err != nil {
			return err
		}
		sql.SetMaxIdleConns(10)

		return nil
	}

	if err := backoff.Retry(connectDB, backoff.NewExponentialBackOff()); err != nil {
		return nil, err
	}

	return db, nil
}

// NewTestDatabase setup test database
// Use docker-compose to bring up PostgreSQL server
func NewTestDatabase() (db *gorm.DB, err error) {
	dsn := `host=localhost user=postgres password=somesecret dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Taipei`
	defaultDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	dbName := "leaderboard_test_db"
	dsn = fmt.Sprintf("host=localhost user=postgres password=somesecret dbname=%s port=5432 sslmode=disable TimeZone=Asia/Taipei", dbName)
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf(`FATAL: database "%s" does not exist`, dbName)) {
			defaultDB = defaultDB.Exec(fmt.Sprintf("create database %s", dbName))
			if err := defaultDB.Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	db = db.Debug()

	return db, nil
}
