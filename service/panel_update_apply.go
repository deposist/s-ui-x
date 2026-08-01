package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/logger"
)

const (
	backupSuffix     = ".bak"
	pendingSuffix    = ".update-pending"
	maxArtifactBytes = 512 << 20 // 512 MiB ceiling for the release tarball
	maxChecksumBytes = 4 << 10
	downloadTimeout  = 5 * time.Minute
	// rollbackAfterAttempts is how many failed boots of a freshly-applied binary
	// trigger an automatic restore of the backup (SR-012).
	rollbackAfterAttempts = 2
)

var errChecksumMismatch = errors.New("artifact checksum does not match the published value")

type panelUpdateDeps struct {
	client   httpDoer
	execPath string
}

func defaultPanelUpdateDeps() panelUpdateDeps {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	return panelUpdateDeps{
		client:   &http.Client{Timeout: downloadTimeout},
		execPath: exe,
	}
}

// applyPipeline downloads, integrity-checks, extracts and atomically swaps the
// panel binary. It only mutates the live executable at the final rename, and
// only after a successful checksum verification (SR-002, SR-007).
func applyPipeline(target ReleaseTarget, deps panelUpdateDeps, setStage func(UpdateStage)) (bool, error) {
	if deps.execPath == "" {
		return false, errors.New("cannot locate current executable")
	}
	dir := filepath.Dir(deps.execPath)

	setStage(UpdateStageDownloading)
	archive := filepath.Join(dir, ".sui-update.tar.gz")
	defer os.Remove(archive)
	if err := downloadToFile(deps.client, target.AssetURL, archive); err != nil {
		return false, err
	}
	expected, err := downloadChecksum(deps.client, target.ChecksumURL)
	if err != nil {
		return false, err
	}

	setStage(UpdateStageVerifying)
	if err := verifySHA256(archive, expected); err != nil {
		return false, err
	}

	setStage(UpdateStageApplying)
	return swapBinary(archive, deps.execPath)
}

// swapBinary extracts the new binary next to the current one and atomically
// replaces it, keeping the previous binary as <exec>.bak for rollback.
func swapBinary(archive string, execPath string) (bool, error) {
	newBin := execPath + ".new"
	backup := execPath + backupSuffix
	if err := extractBinary(archive, newBin); err != nil {
		return false, errors.Join(err, removeUpdateFile(newBin))
	}
	// #nosec G302 -- the replacement panel binary must remain executable.
	if err := os.Chmod(newBin, 0o755); err != nil {
		return false, errors.Join(err, removeUpdateFile(newBin))
	}
	if err := copyFile(execPath, backup); err != nil {
		return false, errors.Join(err, removeUpdateFile(newBin))
	}
	if err := panelUpdateMarkerWriter(execPath, newBin); err != nil {
		return false, errors.Join(err, removeUpdateFile(newBin), removeUpdateFile(backup))
	}
	if err := os.Rename(newBin, execPath); err != nil {
		marker, markerErr := readPendingUpdateMarker(execPath)
		if markerErr == nil {
			markerErr = removePendingDatabaseSnapshot(execPath, marker)
		}
		return false, errors.Join(err, markerErr, removeUpdateFile(newBin), removeUpdateFile(backup), removeUpdateFile(execPath+pendingSuffix))
	}
	if err := panelUpdatePostSwapSync(filepath.Dir(execPath)); err != nil {
		return true, err
	}
	if err := markPendingUpdateApplied(execPath); err != nil {
		return true, err
	}
	return true, nil
}

func downloadToFile(client httpDoer, url string, dest string) error {
	if !strings.HasPrefix(url, "https://") { // SR-003/SR-004: TLS-only, template URLs
		return fmt.Errorf("refusing non-https artifact url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "s-ui-self-update")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("artifact download failed: status %d", resp.StatusCode)
	}
	// #nosec G304 -- dest is the fixed self-update staging path derived from the current executable directory.
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, maxArtifactBytes)); err != nil {
		return err
	}
	return f.Sync()
}

func downloadChecksum(client httpDoer, url string) (string, error) {
	if !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("refusing non-https checksum url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "s-ui-self-update")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("checksum download failed: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumBytes))
	if err != nil {
		return "", err
	}
	// Format produced by `sha256sum`: "<hex>  <filename>".
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty checksum file")
	}
	return strings.ToLower(fields[0]), nil
}

func verifySHA256(path string, expectedHex string) error {
	// #nosec G304 -- path is the previously downloaded self-update artifact staged by applyPipeline.
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expectedHex) {
		return errChecksumMismatch
	}
	return nil
}

// extractBinary writes the `s-ui/sui` entry from the gzip tarball to dest. Header
// paths are never honored for the output location (extraction is pinned to dest),
// preventing path traversal.
func extractBinary(archive string, dest string) error {
	// #nosec G304 -- archive is the checksum-verified self-update tarball staged by applyPipeline.
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("archive does not contain s-ui/sui")
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.ToSlash(header.Name)
		if name != "s-ui/sui" && filepath.Base(name) != "sui" {
			continue
		}
		// #nosec G302,G304 -- dest is the fixed replacement binary path; it must be executable after extraction.
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, io.LimitReader(tr, maxArtifactBytes)); err != nil {
			return err
		}
		return out.Sync()
	}
}

func copyFile(src string, dst string) error {
	// #nosec G304 -- callers pass controlled panel binary/backup paths under the executable directory.
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// #nosec G302,G304 -- dst is a controlled panel binary/backup path and must remain executable for rollback.
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// RestoreBackup restores <execPath>.bak over execPath (rollback, SR-012).
// Rename is required on Unix: writing a running executable can fail with
// ETXTBSY, while replacing its directory entry remains atomic.
func RestoreBackup(execPath string) error {
	backup := execPath + backupSuffix
	if _, err := os.Stat(backup); err != nil {
		return err
	}
	if err := os.Rename(backup, execPath); err != nil {
		return err
	}
	return syncUpdateDirectory(filepath.Dir(execPath))
}

func removeUpdateFileBestEffort(path string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Warning("panel update: cleanup failed:", err)
	}
}
