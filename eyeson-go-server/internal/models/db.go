package models

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"unique;not null"`
	Description string
	Permissions string
}

type User struct {
	gorm.Model
	Username     string `gorm:"unique;not null"`
	Email        string `gorm:"unique"`
	PasswordHash string `gorm:"not null"`
	RoleID       uint
	Role         Role
	LastSeen     time.Time
	IsActive     bool `gorm:"default:true"`
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
