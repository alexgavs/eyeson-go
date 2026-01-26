package models

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"unique;not null" json:"name"`
	Description string `json:"description"`
	Permissions string `json:"permissions"`
}

type User struct {
	gorm.Model
	Username     string    `gorm:"unique;not null" json:"username"`
	Email        string    `gorm:"unique" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	RoleID       uint      `json:"role_id"`
	Role         Role      `json:"role"`
	LastSeen     time.Time `json:"last_seen"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
}

type ActivityLog struct {
	gorm.Model
	Username     string
	ActionType   string
	TargetCLI    string
	TargetMSISDN string
	OldValue     string
	NewValue     string
	Status       string
	ErrorMessage string
	IPAddress    string
	UserAgent    string
}
