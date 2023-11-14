package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var DBerr error

func DBConnect() *gorm.DB {
	dsn := "host=timescaledb user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"
	DB, DBerr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if DBerr != nil {
		panic("Failed to connect database")
	}
	return DB
}

func GetDB() *gorm.DB {
	return DB
}

func Init() {
	DBConnect()
}
