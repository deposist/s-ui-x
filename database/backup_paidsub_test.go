package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupIncludesPaidSubscriptionRecoveryState(t *testing.T) {
	dbPath := filepath.Join(makeDBTempDir(t, "s-ui-paidsub-backup-*"), "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", filepath.Dir(dbPath))
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	db := GetDB()
	models := []any{
		&model.PaidSubBinding{},
		&model.PaidSubTariff{},
		&model.PaidSubPaymentOrder{},
		&model.PaidSubPollCursor{},
		&model.PaidSubInvoiceCancellation{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	binding := model.PaidSubBinding{ClientId: 11, TgUserId: 22}
	tariff := model.PaidSubTariff{Name: "Monthly", Currency: "RUB", AddDays: 30, AddTrafficBytes: 4096}
	order := model.PaidSubPaymentOrder{
		ClientId: 11, TariffId: 1, Provider: "cryptobot", Amount: 100, Currency: "RUB",
		Status: "paid", IdempotencyKey: "backup-order", ProviderRef: "provider-123",
		ProviderChargeID: "cryptobot:provider-123", GrantedDays: 30,
		GrantedTrafficBytes: 4096, SnapshotVersion: 1,
	}
	cursor := model.PaidSubPollCursor{Provider: "cryptobot:poll", LastOrderID: 17}
	cancellation := model.PaidSubInvoiceCancellation{OrderID: 1, Provider: "cryptobot", ProviderRef: "provider-456"}
	for _, value := range []any{&binding, &tariff, &order, &cursor, &cancellation} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}

	backupPath, cleanup, err := PrepareDbBackup("")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, dbErr := backupDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	for _, value := range models {
		var count int64
		if err := backupDB.Model(value).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("backup count for %T = %d; want 1", value, count)
		}
	}
	var restored model.PaidSubPaymentOrder
	if err := backupDB.Where("idempotency_key = ?", order.IdempotencyKey).First(&restored).Error; err != nil {
		t.Fatal(err)
	}
	if restored.ProviderRef != order.ProviderRef || restored.ProviderChargeID != order.ProviderChargeID ||
		restored.GrantedDays != order.GrantedDays || restored.GrantedTrafficBytes != order.GrantedTrafficBytes ||
		restored.SnapshotVersion != order.SnapshotVersion || restored.Status != order.Status {
		t.Fatalf("payment snapshot was not preserved: %+v", restored)
	}
	var restoredCursor model.PaidSubPollCursor
	if err := backupDB.First(&restoredCursor, "provider = ?", cursor.Provider).Error; err != nil {
		t.Fatal(err)
	}
	if restoredCursor.LastOrderID != cursor.LastOrderID {
		t.Fatalf("poll cursor = %+v; want %+v", restoredCursor, cursor)
	}
	var restoredCancellation model.PaidSubInvoiceCancellation
	if err := backupDB.First(&restoredCancellation, "provider_ref = ?", cancellation.ProviderRef).Error; err != nil {
		t.Fatal(err)
	}
	if restoredCancellation.OrderID != cancellation.OrderID {
		t.Fatalf("cancellation = %+v; want %+v", restoredCancellation, cancellation)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatal(err)
	}
}

func TestBackupCreatesEmptyPaidSubscriptionTables(t *testing.T) {
	dbPath := filepath.Join(makeDBTempDir(t, "s-ui-paidsub-empty-backup-*"), "s-ui.db")
	t.Setenv("SUI_DB_FOLDER", filepath.Dir(dbPath))
	if err := InitDB(dbPath); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeMainDB(t)
		cleanupBackupSidecars(dbPath)
	})

	backupPath, cleanup, err := PrepareDbBackup("")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	backupDB, err := gorm.Open(sqlite.Open(backupPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, dbErr := backupDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	for _, table := range []string{
		"paidsub_bindings", "tariffs", "payment_orders", "paidsub_poll_cursors", "paidsub_invoice_cancellations",
	} {
		if !backupDB.Migrator().HasTable(table) {
			t.Fatalf("optional backup table %q is missing", table)
		}
	}
}
