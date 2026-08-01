package paidsub

import (
	"strings"

	"gorm.io/gorm"
)

// EnsureSchema creates the module's tables and indexes idempotently. It is
// called from app wiring at startup so the module owns its schema without
// touching the central migration chain (cmd/migration) or database/db.go.
// Removing the module leaves these tables orphaned but harmless.
func EnsureSchema(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS paidsub_bindings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			tg_user_id INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS tariffs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			price INTEGER NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'RUB',
			stars_amount INTEGER NOT NULL DEFAULT 0,
			add_days INTEGER NOT NULL DEFAULT 0,
			add_traffic_bytes INTEGER NOT NULL DEFAULT 0,
			sort INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS payment_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			tariff_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			amount INTEGER NOT NULL DEFAULT 0,
			currency TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			telegram_user_id INTEGER NOT NULL DEFAULT 0,
			idempotency_key TEXT NOT NULL,
			provider_ref TEXT NOT NULL DEFAULT '',
			provider_charge_id TEXT,
			provider_payload BLOB,
			external_url TEXT,
			created_at INTEGER NOT NULL DEFAULT 0,
			paid_at INTEGER NOT NULL DEFAULT 0,
			expires_at INTEGER NOT NULL DEFAULT 0,
			granted_up INTEGER NOT NULL DEFAULT 0,
			granted_down INTEGER NOT NULL DEFAULT 0,
			granted_days INTEGER NOT NULL DEFAULT 0,
			granted_traffic_bytes INTEGER NOT NULL DEFAULT 0,
			snapshot_version INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS paidsub_poll_cursors (
			provider TEXT PRIMARY KEY,
			last_order_id INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS paidsub_invoice_cancellations (
			order_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			provider_ref TEXT NOT NULL,
			PRIMARY KEY(provider, provider_ref)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_paidsub_bindings_client ON paidsub_bindings(client_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_paidsub_bindings_tg ON paidsub_bindings(tg_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tariffs_enabled_sort ON tariffs(enabled, sort)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_client ON payment_orders(client_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_pending_poll ON payment_orders(provider, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_payment_orders_telegram ON payment_orders(telegram_user_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_idem ON payment_orders(idempotency_key)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	// Additive columns for upgraded installs. SQLite lacks ADD COLUMN IF NOT
	// EXISTS, so guard each migration through GORM's schema introspection.
	migrator := db.Migrator()
	for _, migration := range []struct {
		column string
		ddl    string
	}{
		{"granted_up", `ALTER TABLE payment_orders ADD COLUMN granted_up INTEGER NOT NULL DEFAULT 0`},
		{"granted_down", `ALTER TABLE payment_orders ADD COLUMN granted_down INTEGER NOT NULL DEFAULT 0`},
		{"provider_ref", `ALTER TABLE payment_orders ADD COLUMN provider_ref TEXT NOT NULL DEFAULT ''`},
		{"granted_days", `ALTER TABLE payment_orders ADD COLUMN granted_days INTEGER NOT NULL DEFAULT 0`},
		{"granted_traffic_bytes", `ALTER TABLE payment_orders ADD COLUMN granted_traffic_bytes INTEGER NOT NULL DEFAULT 0`},
		{"snapshot_version", `ALTER TABLE payment_orders ADD COLUMN snapshot_version INTEGER NOT NULL DEFAULT 0`},
	} {
		if migrator.HasColumn(&PaymentOrder{}, migration.column) {
			continue
		}
		if err := db.Exec(migration.ddl).Error; err != nil {
			return err
		}
	}
	if err := migratePaymentProviderRefs(db); err != nil {
		return err
	}
	if err := migratePaymentOrderSnapshots(db); err != nil {
		return err
	}
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_ref ON payment_orders(provider, provider_ref) WHERE provider_ref != ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_charge ON payment_orders(provider, provider_charge_id) WHERE provider_charge_id != ''`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func migratePaymentProviderRefs(db *gorm.DB) error {
	var orders []PaymentOrder
	if err := db.Where("provider = ? AND provider_ref = ''", string(ProviderCryptoBot)).Find(&orders).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, order := range orders {
			ref := extractProviderRef(order.ProviderPayload)
			if ref == "" {
				ref = strings.TrimPrefix(order.ProviderChargeID, "cryptobot:")
			}
			if ref == "" {
				continue
			}
			if err := tx.Model(&PaymentOrder{}).Where("id = ? AND provider_ref = ''", order.Id).
				Update("provider_ref", ref).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func migratePaymentOrderSnapshots(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&PaymentOrder{}).
			Where("status = ? AND snapshot_version = 0", StatusPaid).
			Updates(map[string]any{
				"status":           StatusManualReview,
				"snapshot_version": paymentOrderLegacyResolvedVersion,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&PaymentOrder{}).
			Where("provider = ? AND snapshot_version = 0 AND status IN ?", string(ProviderCryptoBot),
				[]string{StatusPending, StatusInvoiceCreating, StatusFailed, StatusExpired}).
			Update("status", StatusRecoverable).Error; err != nil {
			return err
		}
		return tx.Model(&PaymentOrder{}).
			Where("snapshot_version = 0 AND status IN ?", []string{StatusPending, StatusInvoiceCreating}).
			Update("status", StatusFailed).Error
	})
}
