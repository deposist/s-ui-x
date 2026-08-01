package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestRuntimeProjectionExcludesUnsupportedHistoricalRows(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	rows := []any{
		&model.Inbound{Type: "mieru", Tag: "historical-in", Options: json.RawMessage(`{}`)},
		&model.Outbound{Type: "sudoku", Tag: "historical-out", Options: json.RawMessage(`{}`)},
		&model.Endpoint{Type: "vpn", Tag: "historical-endpoint", Options: json.RawMessage(`{}`)},
		&model.Service{Type: "ccm", Tag: "historical-service", Options: json.RawMessage(`{}`)},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	assertAbsent := func(name, tag string, configs []json.RawMessage, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, config := range configs {
			var identity struct {
				Tag string `json:"tag"`
			}
			if err := json.Unmarshal(config, &identity); err != nil {
				t.Fatalf("%s config: %v", name, err)
			}
			if identity.Tag == tag {
				t.Fatalf("%s projected unsupported tag %q: %s", name, tag, config)
			}
		}
	}
	inbounds, err := (&InboundService{}).GetAllConfig(db)
	assertAbsent("inbounds", "historical-in", inbounds, err)
	outbounds, err := (&OutboundService{}).GetAllConfig(db)
	assertAbsent("outbounds", "historical-out", outbounds, err)
	endpoints, err := (&EndpointService{}).GetAllConfig(db)
	assertAbsent("endpoints", "historical-endpoint", endpoints, err)
	services, err := (&ServicesService{}).GetAllConfig(db)
	assertAbsent("services", "historical-service", services, err)

	tags := []string{"historical-in", "historical-out", "historical-endpoint", "historical-service"}
	for index, row := range rows {
		var count int64
		if err := db.Model(row).Where("tag = ?", tags[index]).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("historical row %d count = %d, want 1", index, count)
		}
	}
}
