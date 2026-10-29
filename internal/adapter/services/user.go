package services

import (
	"github.com/aoamusat/hexgogen/configs"
	"github.com/aoamusat/hexgogen/domain/repositories"
	"github.com/aoamusat/hexgogen/domain/services"
)

type userService struct {
	usersDB repositories.UserDB

	config configs.Config
}

func NewUserService(cfg configs.Config, usersDB repositories.UserDB) services.UserService {
	return &userService{
		config:  cfg,
		usersDB: usersDB,
	}
}
