package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Nama     string `gorm:"unique;not null" json:"nama"`
	Password string `json:"password"`
	Role     string `json:"role"` // "admin" atau "user"
	
}
