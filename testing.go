package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func memoryDb(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	require.Nil(t, err)
	err = db.AutoMigrate(&Vocab{})
	require.Nil(t, err)
	return db
}

func inDaysJSON(n int) string {
	return inDays(n).Format(time.RFC3339)
}
