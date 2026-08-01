package service

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func initDoctorTestDB(t *testing.T) {
	t.Helper()
	if err := database.InitDB(filepath.Join(t.TempDir(), "s-ui.db")); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() {
		if db := database.GetDB(); db != nil {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
				time.Sleep(25 * time.Millisecond)
			}
		}
	})
}

func TestDoctorRunReportsMalformedConfig(t *testing.T) {
	initDoctorTestDB(t)
	if err := (&SettingService{}).SetConfig("{bad json"); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	if report.Status != DoctorSeverityError {
		t.Fatalf("status = %s, want error: %#v", report.Status, report.Items)
	}
}

func TestDoctorRunReportsMissingReferences(t *testing.T) {
	initDoctorTestDB(t)
	config := `{"log":{"disabled":true},"dns":{"servers":[],"final":"missing-dns","rules":[]},"route":{"final":"missing-out","rules":[{"outbound":"missing-rule"}],"rule_set":[]}}`
	if err := (&SettingService{}).SetConfig(config); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	if !doctorReportHas(report, "dns-references", DoctorSeverityError) {
		t.Fatalf("missing dns reference error: %#v", report.Items)
	}
	if !doctorReportHas(report, "route-references", DoctorSeverityError) {
		t.Fatalf("missing route reference error: %#v", report.Items)
	}
}

func TestDoctorRunReportsRuleConditionProblems(t *testing.T) {
	initDoctorTestDB(t)
	config := `{"log":{"disabled":true},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"final":"direct","rules":[{"type":"logical","mode":"and","rules":[]}]}}`
	if err := (&SettingService{}).SetConfig(config); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	item := doctorReportItem(report, "rule-conditions")
	if item == nil || item.Severity != DoctorSeverityError {
		t.Fatalf("missing rule condition error: %#v", report.Items)
	}
	if !strings.Contains(item.Message, "no conditions") {
		t.Fatalf("unexpected rule condition message: %#v", item)
	}
	if details, ok := item.Details.([]string); !ok || len(details) != 1 || !strings.Contains(details[0], "route.rules[0].rules") {
		t.Fatalf("unexpected rule condition details: %#v", item.Details)
	}
}

func TestDoctorRunWarnsOnDroppedRule(t *testing.T) {
	initDoctorTestDB(t)
	config := `{"log":{"disabled":true},"outbounds":[{"type":"direct","tag":"direct"}],"dns":{"servers":[],"rules":[{}]},"route":{"final":"direct","rules":[]}}`
	if err := (&SettingService{}).SetConfig(config); err != nil {
		t.Fatalf("set config: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	item := doctorReportItem(report, "rule-conditions")
	if item == nil || item.Severity != DoctorSeverityWarn {
		t.Fatalf("missing dropped-rule warning: %#v", report.Items)
	}
	if !strings.Contains(item.Message, "silently discarded") {
		t.Fatalf("unexpected dropped-rule message: %#v", item)
	}
}

func TestDoctorRunDistinguishesUnsupportedAndUnavailableCapabilities(t *testing.T) {
	initDoctorTestDB(t)

	if err := database.GetDB().Create(&model.Service{
		Type:    "sudoku",
		Tag:     "legacy-sudoku",
		Options: json.RawMessage(`{}`),
	}).Error; err != nil {
		t.Fatalf("create unsupported service: %v", err)
	}
	if err := database.GetDB().Create(&model.Endpoint{
		Type:    "wireguard",
		Tag:     "wireguard-build-tag",
		Options: json.RawMessage(`{"system":false,"interface_name":"wg-test","server":"127.0.0.1","server_port":1,"local_address":["10.0.0.2/32"],"private_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`),
	}).Error; err != nil {
		t.Fatalf("create unavailable endpoint: %v", err)
	}

	report := (&DoctorService{}).Run("example.com")
	var unsupported, unavailable bool
	for _, item := range report.Items {
		if item.ID == "capability-services-1" && strings.Contains(item.Message, "unsupported by official core") {
			unsupported = true
		}
		if item.ID == "capability-endpoints-1" && strings.Contains(item.Message, "unavailable in this build") {
			unavailable = true
		}
	}
	if !unsupported {
		t.Fatalf("doctor did not report unsupported historical service: %#v", report.Items)
	}
	if !unavailable {
		t.Fatalf("doctor did not report unavailable historical endpoint: %#v", report.Items)
	}
}

func doctorReportItem(report DoctorReport, id string) *DoctorItem {
	for i := range report.Items {
		if report.Items[i].ID == id {
			return &report.Items[i]
		}
	}
	return nil
}

func TestDiagnoseClientReportsDisabledExpiredAndOverLimit(t *testing.T) {
	initDoctorTestDB(t)
	inbounds, _ := json.Marshal([]uint{})
	client := model.Client{
		Enable:   false,
		Name:     "alice",
		Inbounds: inbounds,
		Volume:   10,
		Up:       10,
		Down:     1,
		Expiry:   1,
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatalf("create client: %v", err)
	}

	report, err := (&DoctorService{}).DiagnoseClient(DoctorClientRequest{ClientID: client.Id}, "example.com")
	if err != nil {
		t.Fatalf("DiagnoseClient: %v", err)
	}
	for _, id := range []string{"client-enabled", "client-expiry", "client-traffic", "client-inbounds"} {
		if !doctorReportHas(report, id, DoctorSeverityError) {
			t.Fatalf("missing %s error: %#v", id, report.Items)
		}
	}
}

func TestDiagnoseClientTrafficBoundary(t *testing.T) {
	initDoctorTestDB(t)
	inbounds, _ := json.Marshal([]uint{})
	atLimit := model.Client{Enable: true, Name: "atlimit", Inbounds: inbounds, Volume: 10, Up: 6, Down: 4}
	if err := database.GetDB().Create(&atLimit).Error; err != nil {
		t.Fatalf("create atlimit client: %v", err)
	}
	report, err := (&DoctorService{}).DiagnoseClient(DoctorClientRequest{ClientID: atLimit.Id}, "example.com")
	if err != nil {
		t.Fatalf("DiagnoseClient atlimit: %v", err)
	}
	if !doctorReportHas(report, "client-traffic", DoctorSeverityError) {
		t.Fatalf("used==Volume must be over-limit error: %#v", report.Items)
	}
	if !doctorReportHas(report, "client-enabled", DoctorSeverityOK) {
		t.Fatalf("enabled client must report client-enabled OK: %#v", report.Items)
	}
	if !doctorReportHas(report, "client-expiry", DoctorSeverityOK) {
		t.Fatalf("non-expired client must report client-expiry OK: %#v", report.Items)
	}

	under := model.Client{Enable: true, Name: "under", Inbounds: inbounds, Volume: 10, Up: 5, Down: 4}
	if err := database.GetDB().Create(&under).Error; err != nil {
		t.Fatalf("create under client: %v", err)
	}
	report2, err := (&DoctorService{}).DiagnoseClient(DoctorClientRequest{ClientID: under.Id}, "example.com")
	if err != nil {
		t.Fatalf("DiagnoseClient under: %v", err)
	}
	if !doctorReportHas(report2, "client-traffic", DoctorSeverityOK) {
		t.Fatalf("used==Volume-1 must be within limit (OK): %#v", report2.Items)
	}
}

func doctorReportHas(report DoctorReport, id string, severity DoctorSeverity) bool {
	for _, item := range report.Items {
		if item.ID == id && item.Severity == severity {
			return true
		}
	}
	return false
}
