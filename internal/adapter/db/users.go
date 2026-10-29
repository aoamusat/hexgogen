package db

import (
	"gorm.io/gorm"

	"github.com/aoamusat/hexgogen/domain/repositories"
)

type usersDB struct {
	dbConn *gorm.DB
}

func NewUsersDB(dbconn *gorm.DB) repositories.UserDB {
	return &usersDB{
		dbConn: dbconn,
	}
}
