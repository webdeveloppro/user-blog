package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// Storage provider that can handle read/write operation to database/file/bytes
type Storage interface {
	GetUserByEmail(*User) error
	CreateUser(*User) error
}

// PGStorage provider that can handle read/write from database
type PGStorage struct {
	con *sql.DB
}

// NewPostgres will open db connection or return error
func NewPostgres(host, user, password, dbname string) (pg PGStorage, err error) {

	if host == "" {
		log.Fatal("Empty host string, setup DB_HOST env")
		host = "localhost"
	}

	if user == "" {
		return pg, fmt.Errorf("Empty user string, setup DB_USER env")
	}

	if dbname == "" {
		return pg, fmt.Errorf("Empty dbname string, setup DB_DBNAME env")
	}

	connectionString :=
		fmt.Sprintf("host=%s user=%s password='%s' dbname=%s sslmode=disable", host, user, password, dbname)

	pg.con, err = sql.Open("postgres", connectionString)
	if err != nil {
		return pg, fmt.Errorf("Cannot open postgresql connection: %v", err)
	}
	return pg, nil
}

// CreateUser pull user from postgresql database
func (pg PGStorage) CreateUser(u *User) error {

	err := pg.con.QueryRow("INSERT INTO users(email, password) VALUES($1, $2) RETURNING id",
		u.Email,
		u.Password,
	).Scan(&u.ID)

	return err
}

// GetUserByEmail pull user from postgresql database
func (pg PGStorage) GetUserByEmail(u *User) (err error) {
	err = pg.con.QueryRow("SELECT id, password FROM users WHERE email=$1",
		u.Email,
	).Scan(&u.ID, &u.Password)

	return err
}
