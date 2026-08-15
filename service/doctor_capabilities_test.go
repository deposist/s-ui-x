package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestCapabilityContractChecksReportExactUnsupportedReasons(t *testing.T) {
	initSettingTestDB(t)
	rows := []any{
		&model.Inbound{Type: "mieru", Tag: "historical-mieru", Options: json.RawMessage(`{}`)},
		&model.Endpoint{Type: "wireguard", Tag: "historical-wireguard", Options: json.RawMessage(`{}`)},
	}
	for _, row := range rows {
		if err := database.GetDB().Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	items := capabilityContractChecks(database.GetDB())
	var unsupported, unavailable bool
	for _, item := range items {
		unsupported = unsupported || strings.Contains(item.Message, `inbound "historical-mieru"`) && strings.Contains(item.Message, "unsupported by official core")
		unavailable = unavailable || strings.Contains(item.Message, `endpoint "historical-wireguard"`) && strings.Contains(item.Message, "unavailable in this build")
	}
	if !unsupported || !unavailable {
		t.Fatalf("missing exact capability reasons: %#v", items)
	}
}

func TestCapabilityContractChecksResolvesWarpEndpointAlias(t *testing.T) {
	initSettingTestDB(t)
	if err := database.GetDB().Create(&model.Endpoint{
		Type:    "warp",
		Tag:     "warp-zWj",
		Options: json.RawMessage(`{}`),
	}).Error; err != nil {
		t.Fatal(err)
	}

	items := capabilityContractChecks(database.GetDB())
	for _, item := range items {
		if item.ID != "capability-endpoints-1" {
			continue
		}
		if strings.Contains(item.Message, "unsupported by official core") {
			t.Fatalf("warp endpoint must resolve to wireguard, got: %#v", item)
		}
	}
}
