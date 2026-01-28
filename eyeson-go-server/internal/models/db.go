// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

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

// >>> NEW SYNC ARCHITECTURE MODELS <<<

type SimCard struct {
	gorm.Model
	MSISDN      string    `gorm:"uniqueIndex;not null;size:20" json:"msisdn"`
	CLI         string    `gorm:"index;size:20" json:"cli"`
	IMSI        string    `gorm:"index;size:30" json:"imsi"`
	ICCID       string    `gorm:"size:30" json:"iccid"` // SimSwap field
	IMEI        string    `gorm:"size:30" json:"imei"`
	Status      string    `gorm:"index;size:50" json:"status"`
	RatePlan    string    `gorm:"index;size:100" json:"rate_plan"`
	Label1      string    `json:"label1"`
	Label2      string    `json:"label2"`
	Label3      string    `json:"label3"`
	APN         string    `json:"apn"`
	IP          string    `json:"ip"`
	UsageMB     float64   `json:"usage_mb"`
	AllocatedMB float64   `json:"allocated_mb"`
	LastSession time.Time `json:"last_session"`
	InSession   bool      `json:"in_session"`

	// Sync Metadata
	LastSyncAt time.Time `gorm:"index" json:"last_sync_at"`
	IsSyncing  bool      `gorm:"default:false" json:"is_syncing"`
}

type SyncTask struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Type    string `gorm:"index;size:50" json:"type"`                     // e.g., UPDATE_SIM, CHANGE_STATUS, SYNC_FULL
	Status  string `gorm:"index;default:'PENDING';size:20" json:"status"` // PENDING, PROCESSING, COMPLETED, FAILED
	Payload string `gorm:"type:text" json:"payload"`                      // JSON payload
	Result  string `gorm:"type:text" json:"result"`                       // Error or result

	TargetMSISDN string `gorm:"index;size:20" json:"target_msisdn"` // Optimized lookup for queue status

	Attempt     int       `gorm:"default:0" json:"attempt"`
	MaxAttempts int       `gorm:"default:5" json:"max_attempts"`
	NextRunAt   time.Time `gorm:"index" json:"next_run_at"`

	CreatedBy string `json:"created_by"`
	IPAddress string `json:"ip_address"`
}

type SimHistory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	SimID  uint   `gorm:"index" json:"sim_id"`
	MSISDN string `gorm:"index" json:"msisdn"`

	Action   string `json:"action"` // STATUS_CHANGE, SYNC_UPDATE
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`

	Source    string `json:"source"` // USER, SYNC
	ChangedBy string `json:"changed_by"`
	TaskID    *uint  `json:"task_id"`
}
