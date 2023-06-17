package db

import (
	"fmt"
	"os"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbHost     = os.Getenv("DB_HOST")
	dbPort     = os.Getenv("DB_PORT")
	dbPassword = os.Getenv("DB_PASSWORD")
	dbName     = os.Getenv("DB_NAME")
)

func NewDB() (db *gorm.DB, err error) {
	dsn := getConnectionStr("")
	connectDB := func() error {
		curr, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			fmt.Printf("err:%+v , retry...\n", err)
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
//
// Also remember to set ENV DB_PASSWORD to connect db when running test
func NewTestDatabase() (db *gorm.DB, err error) {
	sysDBName := "postgres"
	dsn := getConnectionStr(sysDBName)
	defaultDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	dbName = "leaderboard_test"
	dsn = getConnectionStr(dbName)
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

func getConnectionStr(nameOfDB string) string {
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if nameOfDB != "" {
		dbName = nameOfDB
	}
	if dbName == "" {
		dbName = "leaderboard"
	}

	return fmt.Sprintf(
		"host=%s user=postgres password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Taipei",
		dbHost,
		dbPassword,
		dbName,
		dbPort,
	)
}
