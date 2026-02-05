// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package database

import (
	"log"
	"strings"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/models"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg *config.Config) {
	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established")

	// Migrate all models including new AuditLog and SyncTaskExtended
	err = DB.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.ActivityLog{},
		&models.SystemSetting{},
		&models.SimCard{},
		&models.SyncTask{},
		&models.SimHistory{},
		&models.AuditLog{},
		&models.SyncTaskExtended{},
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Create indexes for audit_logs table
	createAuditIndexes()

	// Migrate old activity logs to new audit format (one-time migration)
	migrateActivityLogs()

	seedDatabase(cfg)
}

// createAuditIndexes creates indexes for efficient audit log queries
func createAuditIndexes() {
	// Index for entity queries (entity_type + entity_id)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity_type, entity_id)`)

	// Index for user activity queries (user_id + created_at DESC)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_user_time ON audit_logs(user_id, created_at DESC)`)

	// Index for action filtering (action + created_at DESC)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_action_time ON audit_logs(action, created_at DESC)`)

	// Index for status filtering (status + created_at DESC)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_status_time ON audit_logs(status, created_at DESC)`)

	// Index for batch operations
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_batch ON audit_logs(batch_id)`)

	// Index for session tracking
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_session ON audit_logs(session_id)`)

	// Indexes for sync_tasks table (used by SyncTaskExtended)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_queue_status_priority ON sync_tasks(status, priority DESC)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_queue_user ON sync_tasks(user_id, created_at DESC)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_queue_batch ON sync_tasks(batch_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_queue_request ON sync_tasks(request_id)`)

	log.Println("Audit and queue indexes created/verified")
}

// migrateActivityLogs migrates old ActivityLog entries to new AuditLog format
func migrateActivityLogs() {
	var count int64
	DB.Model(&models.AuditLog{}).Count(&count)
	if count > 0 {
		// Already migrated
		return
	}

	var activityLogs []models.ActivityLog
	if err := DB.Find(&activityLogs).Error; err != nil {
		log.Printf("Warning: Could not read activity logs for migration: %v", err)
		return
	}

	if len(activityLogs) == 0 {
		return
	}

	log.Printf("Migrating %d activity logs to new audit format...", len(activityLogs))

	for _, al := range activityLogs {
		auditLog := models.AuditLog{
			Username:   al.Username,
			IPAddress:  al.IPAddress,
			UserAgent:  al.UserAgent,
			EntityType: models.EntitySIM,
			EntityID:   al.TargetMSISDN,
			Action:     mapActivityAction(al.ActionType),
			OldValue:   al.OldValue,
			NewValue:   al.NewValue,
			Source:     models.SourceWeb,
			Status:     models.AuditStatusSuccess,
		}
		auditLog.CreatedAt = al.CreatedAt

		DB.Create(&auditLog)
	}

	log.Printf("Migration complete: %d activity logs migrated", len(activityLogs))
}

// mapActivityAction maps old action names to new AuditAction constants
func mapActivityAction(oldAction string) models.AuditAction {
	switch oldAction {
	case "status_change":
		return models.ActionStatusChange
	case "create":
		return models.ActionCreate
	case "update":
		return models.ActionUpdate
	case "delete":
		return models.ActionDelete
	case "sync":
		return models.ActionSync
	default:
		return models.AuditAction(oldAction)
	}
}

func seedDatabase(cfg *config.Config) {
	var count int64
	DB.Model(&models.Role{}).Count(&count)
	if count == 0 {
		// Create roles
		adminRole := models.Role{
			Name:        "Administrator",
			Description: "Full system access - can manage users, roles, and all SIM operations",
			Permissions: "admin,users.read,users.write,users.delete,roles.read,roles.write,roles.delete,sims.read,sims.write,sims.delete,jobs.read,stats.read",
		}
		moderatorRole := models.Role{
			Name:        "Moderator",
			Description: "Can manage SIMs and view activity, but cannot manage users",
			Permissions: "sims.read,sims.write,sims.delete,jobs.read,stats.read",
		}
		viewerRole := models.Role{
			Name:        "Viewer",
			Description: "Read-only access to SIM data",
			Permissions: "sims.read,jobs.read,stats.read",
		}

		DB.Create(&adminRole)
		DB.Create(&moderatorRole)
		DB.Create(&viewerRole)
		log.Println("Created default roles: Administrator, Moderator, Viewer")

		if cfg != nil && cfg.SeedDefaultAdmin {
			adminPassword := strings.TrimSpace(cfg.DefaultAdminPassword)
			if adminPassword == "" {
				adminPassword = "admin"
			}

			// Create default admin user with bcrypt password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash default admin password: %v", err)
			}

			adminUser := models.User{
				Username:     "admin",
				Email:        "admin@eyeson.local",
				PasswordHash: string(hashedPassword),
				RoleID:       adminRole.ID,
				IsActive:     true,
			}
			DB.Create(&adminUser)
			log.Println("Database seeded with default admin user (username: admin)")
			log.Println("⚠️  IMPORTANT: Change the default password immediately!")
		} else {
			log.Println("Default admin user seeding is disabled")
		}
	}
}
