package models

import "time"

type Permission struct {
	ID        uint      `gorm:"primaryKey"                       json:"id"`
	Code      string    `gorm:"uniqueIndex;size:80;not null"     json:"code"` // e.g. CREATE_ITEM
	Name      string    `gorm:"size:180;not null"                json:"name"` // label di UI
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserPermission struct {
	UserID       uint      `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	PermissionID uint      `gorm:"primaryKey;autoIncrement:false" json:"permission_id"`
	GrantedAt    time.Time `json:"granted_at"`
}
