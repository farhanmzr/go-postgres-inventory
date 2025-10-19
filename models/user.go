package models

import "time"

type User struct {
	ID           uint       `gorm:"primaryKey"                       json:"id"`
	Username     string     `gorm:"uniqueIndex;size:120"    json:"username"`
	FullName     string     `gorm:"size:180"                json:"full_name"`
	UserCode     string     `gorm:"size:60"                          json:"user_code"`
	Position     string     `gorm:"size:120"                         json:"position"`
	WorkLocation string     `gorm:"size:120"                         json:"work_location"`
	Phone        string     `gorm:"size:60"                          json:"phone"`
	Address      string     `gorm:"size:255"                         json:"address"`
	AvatarURL    string     `gorm:"size:255"                         json:"avatar_url"`
	PasswordHash string     `gorm:"size:255"                json:"-"` // jangan dikirim ke client
	IsActive     bool       `gorm:"default:true"                     json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
