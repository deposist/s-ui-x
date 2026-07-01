package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"github.com/sagernet/sing-box/common/srs"
)

func makeTestV2RayGeositeDat(t *testing.T) []byte {
	t.Helper()
	var data []byte
	data = append(data, encodeTestGeoSite("direct-ru", []testGeoDomain{
		{typ: v2rayDomainTypeDomain, value: "gosuslugi.ru"},
		{typ: v2rayDomainTypeFull, value: "exact.example.ru"},
		{typ: v2rayDomainTypePlain, value: "russian-keyword"},
		{typ: v2rayDomainTypeRegex, value: `^.+\\.ru$`},
	})...)
	data = append(data, encodeTestGeoSite("other", []testGeoDomain{{typ: v2rayDomainTypeDomain, value: "example.com"}})...)
	return data
}

type testGeoDomain struct {
	typ   int32
	value string
}

func encodeTestGeoSite(code string, domains []testGeoDomain) []byte {
	var site []byte
	site = append(site, encodeTestStringField(1, code)...)
	for _, domain := range domains {
		var domainMsg []byte
		domainMsg = append(domainMsg, encodeTestVarintField(1, uint64(domain.typ))...)
		domainMsg = append(domainMsg, encodeTestStringField(2, domain.value)...)
		site = append(site, encodeTestBytesField(2, domainMsg)...)
	}
	return encodeTestBytesField(1, site)
}

func encodeTestStringField(number int, value string) []byte {
	return encodeTestBytesField(number, []byte(value))
}

func encodeTestBytesField(number int, value []byte) []byte {
	field := append([]byte{}, encodeTestVarint(uint64(number<<3|2))...)
	field = append(field, encodeTestVarint(uint64(len(value)))...)
	field = append(field, value...)
	return field
}

func encodeTestVarintField(number int, value uint64) []byte {
	field := append([]byte{}, encodeTestVarint(uint64(number<<3))...)
	field = append(field, encodeTestVarint(value)...)
	return field
}

func encodeTestVarint(value uint64) []byte {
	return binary.AppendUvarint(nil, value)
}

func TestCompileGeositeRuSmartDirectReadsV2RayGeositeDat(t *testing.T) {
	dat := makeTestV2RayGeositeDat(t)
	domains, err := readV2RayGeositeCategory(dat, managedRuSmartCategory)
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 4 || domains[0].Type != v2rayDomainTypeDomain || domains[0].Value != "gosuslugi.ru" {
		t.Fatalf("parsed domains = %#v", domains)
	}

	compiled, err := compileGeositeRuSmartDirect(dat)
	if err != nil {
		t.Fatal(err)
	}

	ruleSet, err := srs.Read(bytes.NewReader(compiled), true)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := ruleSet.Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(plain.Rules))
	}
	rule := plain.Rules[0].DefaultOptions
	if !containsTestString(rule.DomainSuffix, "gosuslugi.ru") {
		t.Fatalf("domain_suffix missing gosuslugi.ru: %#v", rule.DomainSuffix)
	}
	if !containsTestString(rule.Domain, "exact.example.ru") {
		t.Fatalf("domain missing exact.example.ru: %#v", rule.Domain)
	}
	if !containsTestString(rule.DomainKeyword, "russian-keyword") {
		t.Fatalf("domain_keyword missing russian-keyword: %#v", rule.DomainKeyword)
	}
	if !containsTestString(rule.DomainRegex, `^.+\\.ru$`) {
		t.Fatalf("domain_regex missing test regex: %#v", rule.DomainRegex)
	}
}

func TestNormalizeManagedRuSmartRuleSetForStorageAndRuntime(t *testing.T) {
	input := json.RawMessage(`{
		"route": {
			"rule_set": [
				{"type":"remote","tag":"preset-ru-direct-geosite","format":"binary","url":"https://example.test/geosite.dat","download_detour":"direct","update_interval":"24h"},
				{"type":"remote","tag":"custom","format":"binary","url":"https://example.test/custom.srs"}
			]
		}
	}`)

	storage, changed, err := normalizeManagedRuleSetsForStorage(input)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("storage normalization should report a change")
	}
	var storageMap map[string]any
	if err := json.Unmarshal(storage, &storageMap); err != nil {
		t.Fatal(err)
	}
	managed := storageMap["route"].(map[string]any)["rule_set"].([]any)[0].(map[string]any)
	if managed["type"] != "local" || managed["format"] != "binary" || managed["path"] != managedRuSmartRuleSetRelativePath {
		t.Fatalf("unexpected managed storage rule-set: %#v", managed)
	}
	if _, ok := managed["url"]; ok {
		t.Fatalf("storage rule-set kept remote url: %#v", managed)
	}

	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	runtimeData, err := rewriteManagedRuleSetsForRuntime(storage)
	if err != nil {
		t.Fatal(err)
	}
	var runtimeMap map[string]any
	if err := json.Unmarshal(runtimeData, &runtimeMap); err != nil {
		t.Fatal(err)
	}
	runtimeManaged := runtimeMap["route"].(map[string]any)["rule_set"].([]any)[0].(map[string]any)
	if got := runtimeManaged["path"]; got != managedRuSmartRuleSetPath() {
		t.Fatalf("runtime managed path = %v, want %s", got, managedRuSmartRuleSetPath())
	}
}

func TestEnsureManagedRuleSetsForConfigDownloadsAndCompilesRuSmart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dir)
	restore := setGeositeRuSmartDownloaderForTest(func(ctx context.Context, sourceURL string) ([]byte, error) {
		if sourceURL != managedRuSmartSourceURL {
			t.Fatalf("sourceURL = %s", sourceURL)
		}
		return makeTestV2RayGeositeDat(t), nil
	})
	defer restore()

	cfg := json.RawMessage(`{"route":{"rule_set":[{"type":"local","tag":"preset-ru-direct-geosite","format":"binary","path":"rulesets/geosite-ru-smart/direct-ru.srs"}]}}`)
	if err := ensureManagedRuleSetsForConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(managedRuSmartRuleSetRelativePath))); err != nil {
		t.Fatalf("compiled srs missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "rulesets", "geosite-ru-smart", "geosite.dat")); err != nil {
		t.Fatalf("downloaded geosite.dat missing: %v", err)
	}
}

func TestEnsureManagedRuleSetsKeepsValidCachedRuSmartWhenRefreshFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dir)
	ruleSetPath := filepath.Join(dir, filepath.FromSlash(managedRuSmartRuleSetRelativePath))
	compiled, err := compileGeositeRuSmartDirect(makeTestV2RayGeositeDat(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(ruleSetPath, compiled, 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-managedRuSmartRefreshInterval - time.Hour)
	if err := os.Chtimes(ruleSetPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	restore := setGeositeRuSmartDownloaderForTest(func(ctx context.Context, sourceURL string) ([]byte, error) {
		return nil, os.ErrNotExist
	})
	defer restore()

	got, err := ensureGeositeRuSmartDirectRuleSet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != ruleSetPath {
		t.Fatalf("path = %s, want %s", got, ruleSetPath)
	}
}

func TestGetConfigRewritesManagedRuSmartToAbsolutePathAndEnsuresFile(t *testing.T) {
	initSettingTestDB(t)
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	restore := setGeositeRuSmartDownloaderForTest(func(ctx context.Context, sourceURL string) ([]byte, error) {
		return makeTestV2RayGeositeDat(t), nil
	})
	defer restore()

	stored := `{
		"log":{"disabled":true},
		"dns":{"servers":[],"rules":[]},
		"route":{"rules":[],"rule_set":[{"type":"local","tag":"preset-ru-direct-geosite","format":"binary","path":"rulesets/geosite-ru-smart/direct-ru.srs"}]}
	}`
	rawConfig, err := (&ConfigService{}).GetConfig(stored)
	if err != nil {
		t.Fatal(err)
	}
	var runtimeMap map[string]any
	if err := json.Unmarshal(*rawConfig, &runtimeMap); err != nil {
		t.Fatal(err)
	}
	managed := runtimeMap["route"].(map[string]any)["rule_set"].([]any)[0].(map[string]any)
	if got := managed["path"]; got != managedRuSmartRuleSetPath() {
		t.Fatalf("runtime config path = %v, want %s", got, managedRuSmartRuleSetPath())
	}
	if _, err := os.Stat(filepath.Join(dbDir, filepath.FromSlash(managedRuSmartRuleSetRelativePath))); err != nil {
		t.Fatalf("GetConfig did not compile managed srs: %v", err)
	}
}

func TestConfigSaveNormalizesManagedRuSmartAndDoesNotPersistAbsolutePath(t *testing.T) {
	initSettingTestDB(t)
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	restore := setGeositeRuSmartDownloaderForTest(func(ctx context.Context, sourceURL string) ([]byte, error) {
		return makeTestV2RayGeositeDat(t), nil
	})
	defer restore()

	payload := json.RawMessage(`{
		"log":{"disabled":true},
		"route":{"rules":[],"rule_set":[{"type":"remote","tag":"preset-ru-direct-geosite","format":"binary","url":"https://example.test/wrong.srs","download_detour":"direct"}]},
		"dns":{"servers":[],"rules":[]}
	}`)
	configService := NewConfigServiceWithRuntime(NewRuntime(nil))
	if _, err := configService.Save("config", "set", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	var stored model.Setting
	if err := database.GetDB().Where("key = ?", "config").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.Value, dbDir) {
		t.Fatalf("stored config leaked absolute managed path: %s", stored.Value)
	}
	if !strings.Contains(stored.Value, managedRuSmartRuleSetRelativePath) {
		t.Fatalf("stored config missing managed relative path: %s", stored.Value)
	}
	if _, err := os.Stat(filepath.Join(dbDir, filepath.FromSlash(managedRuSmartRuleSetRelativePath))); err != nil {
		t.Fatalf("compiled managed srs missing: %v", err)
	}
}

func containsTestString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
