package database

import (
	"fmt"
	"log"

	"fintech_wallet_api/config"
	"fintech_wallet_api/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// DB is the global database handle with read/write routing.
var DB *gorm.DB

// Init opens connections to both the primary and replica databases,
// registers the DBResolver plugin for automatic routing, and runs migrations.
func Init(cfg *config.Config) {
	primaryDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PrimaryDBHost, cfg.PrimaryDBPort,
		cfg.PrimaryDBUser, cfg.PrimaryDBPassword, cfg.PrimaryDBName,
	)

	replicaDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.ReplicaDBHost, cfg.ReplicaDBPort,
		cfg.PrimaryDBUser, cfg.PrimaryDBPassword, cfg.PrimaryDBName,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(primaryDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to primary database: %v", err)
	}

	// Register DBResolver: writes → primary, reads → replica.
	err = DB.Use(dbresolver.Register(dbresolver.Config{
		Replicas: []gorm.Dialector{postgres.Open(replicaDSN)},
		Policy:   dbresolver.RandomPolicy{},
	}))
	if err != nil {
		log.Fatalf("failed to register dbresolver: %v", err)
	}

	// Auto-migrate the wallet table on the primary.
	if err := DB.AutoMigrate(&models.UserWallet{}); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("database connections established (primary + replica)")
}
