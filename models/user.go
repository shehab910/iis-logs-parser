package models

import (
	"iis-logs-parser/utils"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Domains []Domain

	Email        string `json:"email" gorm:"unique" validate:"required,email"`
	Password     string `json:"password" gorm:"not null" validate:"required,strongpassword"`
	FirstName    string `json:"firstName" validate:"required"`
	LastName     string `json:"lastName" validate:"required"`
	JobTitle     string `json:"jobTitle" validate:"required"`
	Organization string `json:"organization" validate:"required"`
	PhoneNumber  string `json:"phoneNumber" validate:"required"`
	Role         string `validate:"oneof=admin user"`
	// If the user is not verified, LastLoginAt used as the last verification email sent
	LastLoginAt time.Time
	Verified    bool
}

func (u *User) Validate() error {
	return utils.Validate.Struct(u)
}
