package importxui

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDialectMHSanaeiDetectsFixture(t *testing.T) {
	db, err := sql.Open("sqlite3", sqliteReadOnlyForTest(t, fixturePath(t, "x-ui.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dialect := Dialect3XUIMHSanaei{}
	ok, err := dialect.Detect(db)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("MHSanaei dialect did not detect fixture")
	}
	inbounds, err := dialect.ReadInbounds(db)
	if err != nil {
		t.Fatal(err)
	}
	clients, err := dialect.ReadClients(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(inbounds) != 7 || len(clients) != 42 {
		t.Fatalf("unexpected fixture counts: inbounds=%d clients=%d", len(inbounds), len(clients))
	}
}

func TestDialectUnknown(t *testing.T) {
	dir := makeImportXUITempDir(t)
	path := filepath.Join(dir, "unknown.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE other (id integer primary key)").Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_, err = openSource(path)
	if !errors.Is(err, ErrDialectUnknown) {
		t.Fatalf("expected ErrDialectUnknown, got %v", err)
	}
}

func TestMapXrayRouting(t *testing.T) {
	raw := `{"routing":{"rules":[{"type":"field","domain":["geosite:cn"],"ip":["geoip:cn"],"outboundTag":"direct"},{"type":"field","balancerTag":"auto"}]}}`
	mapped, warnings, mappedCount, manualCount := MapXrayRouting(raw, nil)
	if mappedCount != 1 || manualCount != 1 {
		t.Fatalf("unexpected routing counts: mapped=%d manual=%d", mappedCount, manualCount)
	}
	if len(warnings) == 0 {
		t.Fatal("balancer rule should produce a warning")
	}
	route := mapped["route"].(map[string]any)
	// domain geosite:cn -> geosite-cn rule set, ip geoip:cn -> geoip-cn rule set.
	if len(route["rules"].([]any)) != 1 || len(route["rule_set"].([]any)) != 2 {
		t.Fatalf("unexpected mapped route: %#v", route)
	}
}

func TestSourceDBReadersRejectUnavailableHandle(t *testing.T) {
	src := &sourceDB{
		db:      &gorm.DB{Config: &gorm.Config{}},
		dialect: Dialect3XUIMHSanaei{},
	}
	historyApply := &applyState{
		plan: map[string]PlanItem{
			planKey(KindHistory, "traffic"): {
				Kind: KindHistory, SrcID: "traffic", Action: ActionCreate,
			},
		},
		report: &Report{},
	}
	cases := []struct {
		name string
		run  func() error
	}{
		{"each inbound", func() error { return src.eachInbound(func(xuiInboundRow) error { return nil }) }},
		{"each client traffic", func() error { return src.eachClientTraffic(func(xuiClientTraffic) error { return nil }) }},
		{"inbound count", func() error { _, err := src.inboundCount(); return err }},
		{"settings", func() error { _, err := src.settings(); return err }},
		{"users", func() error { _, err := src.users(); return err }},
		{"outbound traffics", func() error { _, err := src.outboundTraffics(); return err }},
		{"xray config", func() error { _, err := src.xrayConfig(); return err }},
		{"plan historical", func() error { return planHistorical(context.Background(), src, &MigrationPlan{}) }},
		{"apply historical", func() error { return historyApply.applyHistorical(context.Background(), nil, src, ApplyOptions{}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, gorm.ErrInvalidDB) {
				t.Fatalf("error = %v, want %v", err, gorm.ErrInvalidDB)
			}
		})
	}
}

func sqliteReadOnlyForTest(t *testing.T, path string) string {
	t.Helper()
	dsn, err := sqliteReadOnlyURI(path)
	if err != nil {
		t.Fatal(err)
	}
	return dsn
}
