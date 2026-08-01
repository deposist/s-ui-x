package paidsub

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestApplyPaidOrderIdempotentRenewal(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	// Disabled, expired-by-default client with usage, no inbounds (no restart).
	client := model.Client{
		Enable:    false,
		Name:      "tg42",
		Inbounds:  json.RawMessage("[]"),
		Volume:    0,
		Expiry:    0,
		Up:        100,
		Down:      200,
		TotalUp:   0,
		TotalDown: 0,
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatalf("create client: %v", err)
	}

	tariff := Tariff{Name: "Month", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	if err := db.Create(&tariff).Error; err != nil {
		t.Fatalf("create tariff: %v", err)
	}

	order := PaymentOrder{
		ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa",
		Amount: 10000, Currency: "RUB", Status: StatusPending,
		TelegramUserId: 42, IdempotencyKey: "key-1", CreatedAt: time.Now().Unix(),
		GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	ps := NewPaymentService()
	applied, tgID, err := ps.ApplyPaidOrder(order.Id, "charge-1", nil)
	if err != nil {
		t.Fatalf("ApplyPaidOrder: %v", err)
	}
	if !applied {
		t.Fatal("expected first apply to succeed")
	}
	if tgID != 42 {
		t.Fatalf("expected tgID 42, got %d", tgID)
	}

	var got model.Client
	db.Where("id = ?", client.Id).First(&got)
	if !got.Enable {
		t.Error("client should be re-enabled")
	}
	if got.Volume != 1<<30 {
		t.Errorf("volume = %d, want %d", got.Volume, int64(1<<30))
	}
	if got.Up != 0 || got.Down != 0 {
		t.Errorf("up/down should reset, got up=%d down=%d", got.Up, got.Down)
	}
	if got.TotalUp != 100 || got.TotalDown != 200 {
		t.Errorf("totals = %d/%d, want 100/200", got.TotalUp, got.TotalDown)
	}
	now := time.Now().Unix()
	if got.Expiry < now+29*86400 || got.Expiry > now+31*86400 {
		t.Errorf("expiry not extended ~30d: %d (now %d)", got.Expiry, now)
	}

	var paidOrder PaymentOrder
	db.Where("id = ?", order.Id).First(&paidOrder)
	if paidOrder.Status != StatusPaid || paidOrder.ProviderChargeID != "charge-1" {
		t.Errorf("order not marked paid: %+v", paidOrder)
	}

	// Second apply must be an idempotent no-op (no double renewal).
	applied2, _, err := ps.ApplyPaidOrder(order.Id, "charge-1", nil)
	if err != nil {
		t.Fatalf("second ApplyPaidOrder: %v", err)
	}
	if applied2 {
		t.Fatal("second apply must be a no-op")
	}
	var got2 model.Client
	db.Where("id = ?", client.Id).First(&got2)
	if got2.Volume != 1<<30 {
		t.Errorf("volume changed on replay: %d", got2.Volume)
	}
	if got2.Expiry != got.Expiry {
		t.Errorf("expiry changed on replay: %d != %d", got2.Expiry, got.Expiry)
	}
}

func TestApplyPaidOrderUsesImmutableSnapshotAfterTariffChange(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: false, Name: "immutable-snapshot", Inbounds: json.RawMessage("[]")}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	tariff := Tariff{Name: "Original", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 4096, Enabled: true}
	if err := db.Create(&tariff).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: tariff.Price,
		Currency: tariff.Currency, Status: StatusPending, IdempotencyKey: "immutable-payment",
		GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes,
		SnapshotVersion: paymentOrderSnapshotVersion,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&Tariff{}).Where("id = ?", tariff.Id).
		Updates(map[string]any{"add_days": 1, "add_traffic_bytes": 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&Tariff{}, tariff.Id).Error; err != nil {
		t.Fatal(err)
	}

	applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "immutable-charge", nil)
	if err != nil || !applied {
		t.Fatalf("ApplyPaidOrder = applied %v, err %v", applied, err)
	}
	var got model.Client
	if err := db.First(&got, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Volume != order.GrantedTrafficBytes {
		t.Fatalf("volume = %d; want immutable grant %d", got.Volume, order.GrantedTrafficBytes)
	}
	now := time.Now().Unix()
	if got.Expiry < now+29*86400 || got.Expiry > now+31*86400 {
		t.Fatalf("expiry did not use immutable 30-day grant: %d", got.Expiry)
	}
}

func TestApplyPaidOrderRejectsLegacyResolvedVersionWithoutStateChange(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: false, Name: "legacy-resolved-payment", Inbounds: json.RawMessage("[]"), Volume: 1024, Expiry: 100}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: client.Id, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100,
		Currency: "RUB", Status: StatusPending, IdempotencyKey: "legacy-resolved-payment",
		GrantedDays: 30, GrantedTrafficBytes: 2048, SnapshotVersion: paymentOrderLegacyResolvedVersion,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "cryptobot:legacy", nil)
	if err == nil || applied {
		t.Fatalf("legacy-resolved order = applied %v, err %v; want rejection", applied, err)
	}
	var storedOrder PaymentOrder
	if err := db.First(&storedOrder, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedOrder.Status != StatusPending || storedOrder.ProviderChargeID != "" {
		t.Fatalf("legacy-resolved apply mutated order: %+v", storedOrder)
	}
	var storedClient model.Client
	if err := db.First(&storedClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedClient.Enable != client.Enable || storedClient.Volume != client.Volume || storedClient.Expiry != client.Expiry {
		t.Fatalf("legacy-resolved apply mutated client: before=%+v after=%+v", client, storedClient)
	}
}

func TestApplyPaidOrderRejectsZeroPriceTariff(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: false, Name: "tg99", Inbounds: json.RawMessage("[]"), Expiry: 100}
	db.Create(&client)
	// Price 0 and StarsAmount 0 → must never grant a renewal.
	tariff := Tariff{Name: "Free", Price: 0, StarsAmount: 0, Currency: "RUB", AddDays: 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 0, Currency: "RUB", Status: StatusPending, IdempotencyKey: "zero"}
	db.Create(&order)

	ps := NewPaymentService()
	applied, _, err := ps.ApplyPaidOrder(order.Id, "c", nil)
	if err == nil {
		t.Fatal("expected error for zero-price tariff")
	}
	if applied {
		t.Fatal("zero-price tariff must not apply a renewal")
	}
	// Transaction rolled back: order stays pending, client not renewed.
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusPending {
		t.Errorf("order should remain pending after rejected apply, got %s", o.Status)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Enable || c.Expiry != 100 {
		t.Errorf("client must be unchanged, got enable=%v expiry=%d", c.Enable, c.Expiry)
	}
}

func TestExpireStaleOrders(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	// Non-polled provider: short order-TTL expiry applies.
	stale := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stripe", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "stale", ExpiresAt: now - 10}
	fresh := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stripe", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "fresh", ExpiresAt: now + 3600}
	// Polled provider (cryptobot) past its short TTL must NOT be expired here:
	// it stays pending so a late out-of-band payment is still caught by polling.
	cbStale := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "cryptobot", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "cb-stale", ExpiresAt: now - 10}
	db.Create(&stale)
	db.Create(&fresh)
	db.Create(&cbStale)

	ps := NewPaymentService()
	if err := ps.ExpireStaleOrders(); err != nil {
		t.Fatalf("ExpireStaleOrders: %v", err)
	}
	var s, f, cb PaymentOrder
	db.Where("idempotency_key = ?", "stale").First(&s)
	db.Where("idempotency_key = ?", "fresh").First(&f)
	db.Where("idempotency_key = ?", "cb-stale").First(&cb)
	if s.Status != StatusExpired {
		t.Errorf("stale order not expired: %s", s.Status)
	}
	if f.Status != StatusPending {
		t.Errorf("fresh order should stay pending: %s", f.Status)
	}
	if cb.Status != StatusPending {
		t.Errorf("polled (cryptobot) order must NOT be short-TTL expired: %s", cb.Status)
	}
	_ = database.GetDB()
}

func TestExpireTerminalPolledOrders(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	grace := int64(3600)
	oldTerminal := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "cryptobot", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "cb-old-terminal", CreatedAt: now - grace - 10}
	oldUnconfirmed := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "cryptobot", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "cb-old-unconfirmed", CreatedAt: now - grace - 10}
	recentTerminal := PaymentOrder{ClientId: 1, TariffId: 1, Provider: "cryptobot", Amount: 1, Currency: "RUB", Status: StatusPending, IdempotencyKey: "cb-recent-terminal", CreatedAt: now - 10}
	for _, order := range []*PaymentOrder{&oldTerminal, &oldUnconfirmed, &recentTerminal} {
		if err := db.Create(order).Error; err != nil {
			t.Fatal(err)
		}
	}

	ps := NewPaymentService()
	if err := ps.ExpireTerminalPolledOrders([]uint{oldTerminal.Id, recentTerminal.Id}, grace); err != nil {
		t.Fatalf("ExpireTerminalPolledOrders: %v", err)
	}
	var terminal, unconfirmed, recent PaymentOrder
	db.Where("idempotency_key = ?", oldTerminal.IdempotencyKey).First(&terminal)
	db.Where("idempotency_key = ?", oldUnconfirmed.IdempotencyKey).First(&unconfirmed)
	db.Where("idempotency_key = ?", recentTerminal.IdempotencyKey).First(&recent)
	if terminal.Status != StatusExpired {
		t.Errorf("old provider-terminal order not expired: %s", terminal.Status)
	}
	if unconfirmed.Status != StatusPending {
		t.Errorf("old unconfirmed order must stay pending: %s", unconfirmed.Status)
	}
	if recent.Status != StatusPending {
		t.Errorf("recent provider-terminal order should stay pending through grace: %s", recent.Status)
	}
}

func TestOrdersForTgUserScoped(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	db.Create(&PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stars", Amount: 5, Currency: "XTR", Status: StatusPaid, TelegramUserId: 100, IdempotencyKey: "a"})
	db.Create(&PaymentOrder{ClientId: 1, TariffId: 1, Provider: "stars", Amount: 6, Currency: "XTR", Status: StatusPending, TelegramUserId: 100, IdempotencyKey: "b"})
	db.Create(&PaymentOrder{ClientId: 2, TariffId: 1, Provider: "stars", Amount: 7, Currency: "XTR", Status: StatusPaid, TelegramUserId: 200, IdempotencyKey: "c"})

	ps := NewPaymentService()
	got, err := ps.OrdersForTgUser(100, 20)
	if err != nil {
		t.Fatalf("OrdersForTgUser: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("OrdersForTgUser(100) = %d orders, want 2", len(got))
	}
	for _, o := range got {
		if o.TelegramUserId != 100 {
			t.Errorf("leaked order belonging to tg %d", o.TelegramUserId)
		}
	}
	// Refundable = paid only.
	ref, err := ps.RefundableOrdersForTgUser(100, 20)
	if err != nil {
		t.Fatalf("RefundableOrdersForTgUser: %v", err)
	}
	if len(ref) != 1 || ref[0].Status != StatusPaid {
		t.Errorf("RefundableOrdersForTgUser(100) = %+v, want exactly 1 paid", ref)
	}
}

func TestFinalizeRefundRevokeRollsBackOnce(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	client := model.Client{Enable: true, Name: "tg7", Inbounds: json.RawMessage("[]"), Volume: 5 << 30, Expiry: now + 40*86400}
	db.Create(&client)
	tariff := Tariff{Name: "M", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPaid, TelegramUserId: 7, IdempotencyKey: "r1", GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion}
	db.Create(&order)

	ps := NewPaymentService()
	if err := ps.finalizeRefund(order.Id, true); err != nil {
		t.Fatalf("finalizeRefund: %v", err)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusRefunded {
		t.Errorf("status = %s, want refunded", o.Status)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	wantExpiry := (now + 40*86400) - 30*86400
	if c.Expiry < wantExpiry-2 || c.Expiry > wantExpiry+2 {
		t.Errorf("expiry = %d, want ~%d", c.Expiry, wantExpiry)
	}
	if c.Volume != (5<<30)-(1<<30) {
		t.Errorf("volume = %d, want %d", c.Volume, int64((5<<30)-(1<<30)))
	}
	if !c.Enable {
		t.Error("client must not be disabled by a refund")
	}

	// Second call must be an idempotent no-op (no double roll-back).
	if err := ps.finalizeRefund(order.Id, true); !errors.Is(err, errAlreadyApplied) {
		t.Errorf("second finalizeRefund = %v, want errAlreadyApplied", err)
	}
	var c2 model.Client
	db.Where("id = ?", client.Id).First(&c2)
	if c2.Volume != c.Volume || c2.Expiry != c.Expiry {
		t.Error("second refund must not change the client again")
	}
}

func TestFinalizeRefundUsesImmutableSnapshotAfterTariffChange(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	client := model.Client{Enable: true, Name: "immutable-refund", Inbounds: json.RawMessage("[]"), Volume: 8 << 30, Expiry: now + 40*86400}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	tariff := Tariff{Name: "Original refund", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 2 << 30, Enabled: true}
	if err := db.Create(&tariff).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: tariff.Price,
		Currency: tariff.Currency, Status: StatusPaid, IdempotencyKey: "immutable-refund",
		GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes,
		SnapshotVersion: paymentOrderSnapshotVersion,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&Tariff{}).Where("id = ?", tariff.Id).
		Updates(map[string]any{"add_days": 1, "add_traffic_bytes": 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&Tariff{}, tariff.Id).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewPaymentService().finalizeRefund(order.Id, true); err != nil {
		t.Fatalf("finalizeRefund: %v", err)
	}
	var got model.Client
	if err := db.First(&got, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Volume != client.Volume-order.GrantedTrafficBytes {
		t.Fatalf("volume = %d; want immutable rollback %d", got.Volume, client.Volume-order.GrantedTrafficBytes)
	}
	wantExpiry := client.Expiry - int64(order.GrantedDays)*86400
	if got.Expiry != wantExpiry {
		t.Fatalf("expiry = %d; want immutable rollback %d", got.Expiry, wantExpiry)
	}
}

func TestFinalizeRefundRejectsInvalidSnapshotWithoutStateChange(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	client := model.Client{Enable: true, Name: "invalid-refund-snapshot", Inbounds: json.RawMessage("[]"), Volume: 4096, Expiry: time.Now().Unix() + 86400}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: client.Id, TariffId: 1, Provider: "yookassa", Amount: 100,
		Currency: "RUB", Status: StatusPaid, IdempotencyKey: "invalid-refund-snapshot",
		GrantedDays: 30, GrantedTrafficBytes: 2048, SnapshotVersion: 0,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewPaymentService().finalizeRefund(order.Id, true); err == nil {
		t.Fatal("invalid snapshot refund unexpectedly succeeded")
	}
	var storedOrder PaymentOrder
	if err := db.First(&storedOrder, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedOrder.Status != StatusPaid {
		t.Fatalf("invalid refund changed order status to %q", storedOrder.Status)
	}
	var storedClient model.Client
	if err := db.First(&storedClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedClient.Volume != client.Volume || storedClient.Expiry != client.Expiry || storedClient.Enable != client.Enable {
		t.Fatalf("invalid refund mutated client: before=%+v after=%+v", client, storedClient)
	}
}

func TestFinalizeRefundNoRevokeKeepsClient(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	client := model.Client{Enable: true, Name: "tg8", Inbounds: json.RawMessage("[]"), Volume: 2 << 30, Expiry: now + 10*86400}
	db.Create(&client)
	tariff := Tariff{Name: "M", Price: 10000, Currency: "RUB", AddDays: 30, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 10000, Currency: "RUB", Status: StatusPaid, TelegramUserId: 8, IdempotencyKey: "r2", GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion}
	db.Create(&order)

	ps := NewPaymentService()
	if err := ps.finalizeRefund(order.Id, false); err != nil {
		t.Fatalf("finalizeRefund: %v", err)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Volume != 2<<30 || c.Expiry != now+10*86400 {
		t.Errorf("client changed despite revoke=false: volume=%d expiry=%d", c.Volume, c.Expiry)
	}
	var o PaymentOrder
	db.Where("id = ?", order.Id).First(&o)
	if o.Status != StatusRefunded {
		t.Errorf("status = %s, want refunded", o.Status)
	}
}

func TestFinalizeRefundFloorsExpiryAndVolume(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	now := time.Now().Unix()
	// addDays (365) exceeds remaining (5d) and addTraffic exceeds volume → floor.
	client := model.Client{Enable: true, Name: "tg9", Inbounds: json.RawMessage("[]"), Volume: 1 << 20, Expiry: now + 5*86400}
	db.Create(&client)
	tariff := Tariff{Name: "Y", Price: 1, Currency: "RUB", AddDays: 365, AddTrafficBytes: 1 << 30, Enabled: true}
	db.Create(&tariff)
	order := PaymentOrder{ClientId: client.Id, TariffId: tariff.Id, Provider: "yookassa", Amount: 1, Currency: "RUB", Status: StatusPaid, TelegramUserId: 9, IdempotencyKey: "r3", GrantedDays: tariff.AddDays, GrantedTrafficBytes: tariff.AddTrafficBytes, SnapshotVersion: paymentOrderSnapshotVersion}
	db.Create(&order)

	ps := NewPaymentService()
	if err := ps.finalizeRefund(order.Id, true); err != nil {
		t.Fatalf("finalizeRefund: %v", err)
	}
	var c model.Client
	db.Where("id = ?", client.Id).First(&c)
	if c.Expiry < now-2 || c.Expiry > now+2 {
		t.Errorf("expiry floor = %d, want ~now %d", c.Expiry, now)
	}
	if c.Volume != 0 {
		t.Errorf("volume floor = %d, want 0", c.Volume)
	}
}
