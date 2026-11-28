package persistence

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// getTestDB connects to the MySQL server the integration fixtures run
// against (normally the shared dev database), taking TEST_MYSQL_DSN or
// falling back to DB_DSN. An unset DSN or unreachable server returns an
// error so the caller can skip instead of failing.
func getTestDB() (*gorm.DB, error) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		return nil, fmt.Errorf("no test database configured: set TEST_MYSQL_DSN or DB_DSN")
	}
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
