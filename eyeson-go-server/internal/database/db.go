package database

import (
	"log"

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

	err = DB.AutoMigrate(&models.User{}, &models.Role{}, &models.ActivityLog{}, &models.SimCard{}, &models.SyncTask{}, &models.SimHistory{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	seedDatabase()
}

func seedDatabase() {
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

		// Create default admin user with bcrypt password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
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
		log.Println("Database seeded with default admin user (username: admin, password: admin)")
		log.Println("⚠️  IMPORTANT: Change the default password immediately!")
	}
}
