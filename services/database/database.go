// package database

// import (
// 	"gorm.io/driver/postgres"
// 	"gorm.io/gorm"
// )

// var DB *gorm.DB
// var DBerr error

// func DBConnect() *gorm.DB {
// 	dsn := "host=timescaledb user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"
// 	DB, DBerr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
// 	if DBerr != nil {
// 		panic("Failed to connect database")
// 	}
// 	return DB
// }

// func GetDB() *gorm.DB {
// 	return DB
// }

//	func Init() {
//		DBConnect()
//	}
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var DBerr error

func DBConnect() *gorm.DB {
	// ✅ Use correct host and port for your Docker host
	dsn := "host=127.0.0.1 user=postgres password=postgres dbname=postgres port=5555 sslmode=disable"

	// Try several times before giving up
	for i := 0; i < 10; i++ {
		DB, DBerr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if DBerr == nil {
			fmt.Println("Connected to TimescaleDB successfully.")
			return DB
		}
		fmt.Printf("DB connection failed (%v); retrying...\n", DBerr)
		time.Sleep(3 * time.Second)
	}

	// ❌ Still no success → just warn, do not crash
	fmt.Println("Warning: proceeding without DB connection (simulation mode).")
	DB = nil
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func Init() {
	DBConnect()
}
