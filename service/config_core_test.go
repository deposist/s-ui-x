package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/core"
)

func TestConfigCoreMethodsHandleNilCore(t *testing.T) {
	t.Cleanup(ReplaceDefaultRuntimeForTest(NewRuntimeWithCoreProvider(nil)))

	configService := &ConfigService{}
	if configService.IsCoreRunning() {
		t.Fatal("nil core should not report running")
	}

	tests := map[string]func() error{
		"StartCore":   configService.StartCore,
		"RestartCore": configService.RestartCore,
		"StopCore":    configService.StopCore,
	}
	for name, call := range tests {
		err := call()
		if err == nil || !strings.Contains(err.Error(), "core not initialized") {
			t.Fatalf("%s returned %v, want core not initialized", name, err)
		}
	}
}

func TestConfigCoreLifecycleStartReloadStop(t *testing.T) {
	initSettingTestDB(t)

	coreInstance := core.NewCore()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	t.Cleanup(func() {
		_ = configService.StopCore()
	})

	if err := configService.StartCore(); err != nil {
		t.Fatalf("start core: %v", err)
	}
	if !configService.IsCoreRunning() || coreInstance.GetInstance() == nil {
		t.Fatal("core should be running after start")
	}
	started := coreInstance.GetInstance()

	if err := configService.RestartCore(); err != nil {
		t.Fatalf("reload core: %v", err)
	}
	if !configService.IsCoreRunning() || coreInstance.GetInstance() == nil {
		t.Fatal("core should be running after reload")
	}
	if coreInstance.GetInstance() == started {
		t.Fatal("reload should replace the running core instance")
	}

	if err := configService.StopCore(); err != nil {
		t.Fatalf("stop core: %v", err)
	}
	if configService.IsCoreRunning() || coreInstance.GetInstance() != nil {
		t.Fatal("core should be stopped after stop")
	}
}

func TestGetConfigPreservesTopLevelCertificateAndUnknownFields(t *testing.T) {
	initSettingTestDB(t)

	input := `{
  "log": { "level": "info" },
  "dns": { "servers": [], "rules": [] },
  "route": { "rules": [] },
  "experimental": {},
  "certificate": {
    "store": "mozilla",
    "certificate_path": ["/etc/ssl/custom.pem"]
  },
  "future_top_level": { "enabled": true }
}`

	rawConfig, err := (&ConfigService{}).GetConfig(input)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(*rawConfig, &config); err != nil {
		t.Fatal(err)
	}
	if _, ok := config["certificate"]; !ok {
		t.Fatalf("certificate was not preserved in runtime config: %s", string(*rawConfig))
	}
	if _, ok := config["future_top_level"]; !ok {
		t.Fatalf("unknown top-level field was not preserved in runtime config: %s", string(*rawConfig))
	}
}
