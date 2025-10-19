package models

import "time"

type Admin struct {
	ID           uint       `gorm:"primaryKey"                       json:"id"`
	Username     string     `gorm:"uniqueIndex;size:120"    json:"username"`
	FullName     string     `gorm:"size:180"                json:"full_name"`
	AdminCode    string     `gorm:"size:60"                          json:"admin_code"`
	Position     string     `gorm:"size:120"                         json:"position"`
	Phone        string     `gorm:"size:60"                          json:"phone"`
	Address      string     `gorm:"size:255"                         json:"address"`
	AvatarURL    string     `gorm:"size:255"                         json:"avatar_url"`
	PasswordHash string     `gorm:"size:255"                json:"-"` // disembunyikan di JSON
	IsActive     bool       `gorm:"default:true"                     json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
