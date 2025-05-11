package db

import (
    "database/sql"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitDB() {
    var err error
    DB, err = sql.Open("mysql", "user:password@tcp(borrow-db:3306)/borrow_service")
    if err != nil {
        panic(fmt.Sprintf("Failed to connect to DB: %s", err))
    }
    DB.SetMaxOpenConns(10)
    DB.SetMaxIdleConns(5)
}