package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestConfigSaveRejectsUnsupportedCapabilityBeforePersistence(t *testing.T) {
	tests := []struct {
		object string
		data   json.RawMessage
		model  any
		tag    string
	}{
		{object: "inbounds", data: json.RawMessage(`{"type":"mieru","tag":"forbidden-in"}`), model: &model.Inbound{}, tag: "forbidden-in"},
		{object: "outbounds", data: json.RawMessage(`{"type":"sudoku","tag":"forbidden-out"}`), model: &model.Outbound{}, tag: "forbidden-out"},
		{object: "endpoints", data: json.RawMessage(`{"type":"vpn","tag":"forbidden-endpoint"}`), model: &model.Endpoint{}, tag: "forbidden-endpoint"},
		{object: "services", data: json.RawMessage(`{"type":"ccm","tag":"forbidden-service"}`), model: &model.Service{}, tag: "forbidden-service"},
	}

	for _, test := range tests {
		t.Run(test.object, func(t *testing.T) {
			initSettingTestDB(t)
			configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
			_, err := configService.Save(test.object, "new", test.data, "", "admin", "example.com")
			if err == nil {
				t.Fatal("unsupported capability save succeeded")
			}
			if !strings.Contains(err.Error(), "unsupported") || !strings.Contains(err.Error(), "official core") {
				t.Fatalf("unexpected rejection: %v", err)
			}
			var count int64
			if err := database.GetDB().Model(test.model).Where("tag = ?", test.tag).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("unsupported %s row persisted", test.object)
			}
		})
	}
}

func TestConfigSaveRejectsUnavailableOfficialCapability(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("endpoints", "new", json.RawMessage(`{"type":"wireguard","tag":"wg-unavailable"}`), "", "admin", "example.com")
	if err == nil || !strings.Contains(err.Error(), "unavailable in this build") {
		t.Fatalf("unavailable official capability was not rejected: %v", err)
	}
}

func TestValidateCapabilitySaveResolvesWarpEndpointAlias(t *testing.T) {
	initSettingTestDB(t)
	if err := database.GetDB().Create(&model.Endpoint{
		Type:    "warp",
		Tag:     "warp-zWj",
		Options: json.RawMessage(`{}`),
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Editing an existing WARP endpoint must not be rejected as unsupported:
	// the panel-only alias resolves to wireguard.
	if err := validateCapabilitySave(database.GetDB(), "endpoints", "edit", json.RawMessage(`{"id":1,"type":"warp"}`)); err != nil {
		t.Fatalf("warp endpoint edit rejected: %v", err)
	}

	// A new WARP endpoint follows wireguard availability exactly; in builds
	// without the wireguard tag it is unavailable, never "unsupported".
	err := validateCapabilitySave(database.GetDB(), "endpoints", "new", json.RawMessage(`{"type":"warp"}`))
	if err == nil {
		return // wireguard compiled in this build; new warp is allowed
	}
	if strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("warp endpoint must not be reported as unsupported: %v", err)
	}
}

func TestConfigSaveAllowsDeletionOfUnsupportedHistoricalRow(t *testing.T) {
	initSettingTestDB(t)
	historical := model.Service{Type: "ccm", Tag: "historical-unsupported", Options: json.RawMessage(`{}`)}
	if err := database.GetDB().Create(&historical).Error; err != nil {
		t.Fatal(err)
	}
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	if _, err := configService.Save("services", "del", json.RawMessage(`"historical-unsupported"`), "", "admin", "example.com"); err != nil {
		t.Fatalf("unsupported historical row could not be deleted: %v", err)
	}
}
