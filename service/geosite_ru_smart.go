package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/util/common"

	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

const (
	managedRuSmartRuleSetTag          = "preset-ru-direct-geosite"
	managedRuSmartRuleSetRelativePath = "rulesets/geosite-ru-smart/direct-ru.srs"
	managedRuSmartDatFilename         = "geosite.dat"
	managedRuSmartCategory            = "direct-ru"
	managedRuSmartSourceURL           = "https://github.com/wastrel-g/geosite-ru-smart/releases/latest/download/geosite.dat"
	managedRuSmartDownloadLimit       = 16 << 20
	managedRuSmartDownloadTimeout     = 45 * time.Second
	managedRuSmartRefreshInterval     = 24 * time.Hour
)

type geositeRuSmartDownloader func(ctx context.Context, sourceURL string) ([]byte, error)

var downloadGeositeRuSmart geositeRuSmartDownloader = downloadGeositeRuSmartHTTP

func managedRuSmartRuleSetPath() string {
	return filepath.Join(config.GetDBFolderPath(), filepath.FromSlash(managedRuSmartRuleSetRelativePath))
}

func managedRuSmartDatPath() string {
	return filepath.Join(filepath.Dir(managedRuSmartRuleSetPath()), managedRuSmartDatFilename)
}

func managedRuleSetConfigUsesRuSmart(data json.RawMessage) (bool, error) {
	var top struct {
		Route struct {
			RuleSet []struct {
				Tag string `json:"tag"`
			} `json:"rule_set"`
		} `json:"route"`
	}
	if err := json.Unmarshal(data, &top); err != nil {
		return false, err
	}
	for _, ruleSet := range top.Route.RuleSet {
		if ruleSet.Tag == managedRuSmartRuleSetTag {
			return true, nil
		}
	}
	return false, nil
}

func ensureManagedRuleSetsForConfig(data json.RawMessage) error {
	usesRuSmart, err := managedRuleSetConfigUsesRuSmart(data)
	if err != nil {
		return err
	}
	if !usesRuSmart {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), managedRuSmartDownloadTimeout)
	defer cancel()
	_, err = ensureGeositeRuSmartDirectRuleSet(ctx)
	if err != nil {
		return common.NewError("prepare RU smart rule-set: ", err)
	}
	return nil
}

func normalizeManagedRuleSetsForStorage(data json.RawMessage) (json.RawMessage, bool, error) {
	return rewriteManagedRuSmartRuleSet(data, managedRuSmartRuleSetRelativePath)
}

func rewriteManagedRuleSetsForRuntime(data json.RawMessage) (json.RawMessage, error) {
	rewritten, _, err := rewriteManagedRuSmartRuleSet(data, managedRuSmartRuleSetPath())
	return rewritten, err
}

func rewriteManagedRuSmartRuleSet(data json.RawMessage, path string) (json.RawMessage, bool, error) {
	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		return data, false, err
	}
	route, ok := top["route"].(map[string]any)
	if !ok {
		return data, false, nil
	}
	ruleSets, ok := route["rule_set"].([]any)
	if !ok {
		return data, false, nil
	}

	changed := false
	for _, item := range ruleSets {
		ruleSet, ok := item.(map[string]any)
		if !ok || stringAny(ruleSet["tag"]) != managedRuSmartRuleSetTag {
			continue
		}
		if stringAny(ruleSet["type"]) != C.RuleSetTypeLocal {
			ruleSet["type"] = C.RuleSetTypeLocal
			changed = true
		}
		if stringAny(ruleSet["format"]) != C.RuleSetFormatBinary {
			ruleSet["format"] = C.RuleSetFormatBinary
			changed = true
		}
		if stringAny(ruleSet["path"]) != path {
			ruleSet["path"] = path
			changed = true
		}
		if _, exists := ruleSet["url"]; exists {
			delete(ruleSet, "url")
			changed = true
		}
		if _, exists := ruleSet["download_detour"]; exists {
			delete(ruleSet, "download_detour")
			changed = true
		}
		if _, exists := ruleSet["update_interval"]; exists {
			delete(ruleSet, "update_interval")
			changed = true
		}
	}
	if !changed {
		return data, false, nil
	}
	rewritten, err := json.Marshal(top)
	if err != nil {
		return data, false, err
	}
	return rewritten, true, nil
}

func stringAny(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func ensureGeositeRuSmartDirectRuleSet(ctx context.Context) (string, error) {
	ruleSetPath := managedRuSmartRuleSetPath()
	ready := geositeRuSmartRuleSetReady(ruleSetPath)
	if ready && !geositeRuSmartRuleSetStale(ruleSetPath) {
		return ruleSetPath, nil
	}

	data, err := downloadGeositeRuSmart(ctx, managedRuSmartSourceURL)
	if err != nil {
		if ready {
			return ruleSetPath, nil
		}
		return "", err
	}
	ruleSetBytes, err := compileGeositeRuSmartDirect(data)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(ruleSetPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	if err := writeFileAtomic(managedRuSmartDatPath(), data, 0o644); err != nil {
		return "", err
	}
	if err := writeFileAtomic(ruleSetPath, ruleSetBytes, 0o644); err != nil {
		return "", err
	}
	if !geositeRuSmartRuleSetReady(ruleSetPath) {
		return "", common.NewError("compiled RU smart rule-set did not validate")
	}
	return ruleSetPath, nil
}

func geositeRuSmartRuleSetStale(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) >= managedRuSmartRefreshInterval
}

func geositeRuSmartRuleSetReady(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	ruleSet, err := srs.Read(file, false)
	if err != nil {
		return false
	}
	plain, err := ruleSet.Upgrade()
	return err == nil && len(plain.Rules) > 0
}

func compileGeositeRuSmartDirect(data []byte) ([]byte, error) {
	domains, err := readV2RayGeositeCategory(data, managedRuSmartCategory)
	if err != nil {
		return nil, err
	}

	var headless option.DefaultHeadlessRule
	for _, domain := range domains {
		value := strings.TrimSpace(domain.Value)
		if value == "" {
			continue
		}
		switch domain.Type {
		case v2rayDomainTypePlain:
			headless.DomainKeyword = append(headless.DomainKeyword, value)
		case v2rayDomainTypeRegex:
			headless.DomainRegex = append(headless.DomainRegex, value)
		case v2rayDomainTypeDomain:
			headless.DomainSuffix = append(headless.DomainSuffix, value)
		case v2rayDomainTypeFull:
			headless.Domain = append(headless.Domain, value)
		default:
			return nil, common.NewError("unsupported geosite domain type: ", domain.Type)
		}
	}
	if len(headless.Domain) == 0 && len(headless.DomainSuffix) == 0 && len(headless.DomainKeyword) == 0 && len(headless.DomainRegex) == 0 {
		return nil, common.NewError("geosite category ", managedRuSmartCategory, " is empty")
	}

	plain := option.PlainRuleSet{
		Rules: []option.HeadlessRule{
			{
				Type:           C.RuleTypeDefault,
				DefaultOptions: headless,
			},
		},
	}
	var output bytes.Buffer
	if err := srs.Write(&output, plain, C.RuleSetVersion2); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func downloadGeositeRuSmartHTTP(ctx context.Context, sourceURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "s-ui-x-geosite-ru-smart/1")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, common.NewError("unexpected geosite-ru-smart status: ", response.Status)
	}
	if response.ContentLength > managedRuSmartDownloadLimit {
		return nil, common.NewError("geosite-ru-smart download is too large")
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, managedRuSmartDownloadLimit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > managedRuSmartDownloadLimit {
		return nil, common.NewError("geosite-ru-smart download is too large")
	}
	return body, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return err
	}
	if err = os.Chmod(path, perm); err != nil {
		return err
	}
	return nil
}

func setGeositeRuSmartDownloaderForTest(fn geositeRuSmartDownloader) func() {
	previous := downloadGeositeRuSmart
	if fn == nil {
		downloadGeositeRuSmart = downloadGeositeRuSmartHTTP
	} else {
		downloadGeositeRuSmart = fn
	}
	return func() { downloadGeositeRuSmart = previous }
}
