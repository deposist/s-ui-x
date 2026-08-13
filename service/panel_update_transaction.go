package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/deposist/s-ui-x/logger"
)

const updateMarkerVersion = 2

type updateTransactionPhase string

const (
	updatePhasePrepared  updateTransactionPhase = "prepared"
	updatePhaseApplied   updateTransactionPhase = "applied"
	updatePhaseBooting   updateTransactionPhase = "booting"
	updatePhaseConfirmed updateTransactionPhase = "confirmed"
)

type pendingUpdateMarker struct {
	Version         int                    `json:"version"`
	TransactionID   string                 `json:"transactionId"`
	Phase           updateTransactionPhase `json:"phase"`
	Attempts        int                    `json:"attempts"`
	CandidateSHA256 string                 `json:"candidateSha256"`
	BackupSHA256    string                 `json:"backupSha256"`
	DatabasePath    string                 `json:"databasePath,omitempty"`
	DatabaseSHA256  string                 `json:"databaseSha256,omitempty"`
}

// Installed by app.Init to avoid a service/database import cycle.
var PanelUpdateDatabaseSnapshot func() (snapshotPath string, databasePath string, err error)
var PanelUpdateDatabaseRestore func(snapshotPath string, databasePath string) error

var panelUpdateMarkerWriter = writePendingMarkerForCandidate
var panelUpdatePostSwapSync = syncUpdateDirectory

func writePendingMarker(execPath string) error {
	return writePendingMarkerForCandidate(execPath, execPath)
}

func writePendingMarkerForCandidate(execPath, candidatePath string) error {
	candidateDigest, err := updateFileSHA256(candidatePath)
	if err != nil {
		return err
	}
	backupDigest, err := updateFileSHA256(execPath + backupSuffix)
	if err != nil {
		return err
	}
	transactionID, err := newUpdateTransactionID()
	if err != nil {
		return err
	}
	marker := pendingUpdateMarker{
		Version: updateMarkerVersion, TransactionID: transactionID, Phase: updatePhasePrepared,
		CandidateSHA256: candidateDigest, BackupSHA256: backupDigest,
	}
	if PanelUpdateDatabaseSnapshot == nil {
		return errors.New("database snapshot handler is unavailable")
	}
	snapshotPath, databasePath, err := PanelUpdateDatabaseSnapshot()
	if err != nil {
		return fmt.Errorf("snapshot database before update: %w", err)
	}
	if snapshotPath != "" {
		ownedPath := pendingDatabaseSnapshotPath(execPath, pendingUpdateMarker{TransactionID: transactionID, DatabasePath: databasePath, DatabaseSHA256: "pending"})
		if err := os.Rename(snapshotPath, ownedPath); err != nil {
			return fmt.Errorf("install update database snapshot: %w", err)
		}
		digest, err := updateFileSHA256(ownedPath)
		if err != nil {
			return errors.Join(err, removeUpdateFile(ownedPath))
		}
		marker.DatabasePath = databasePath
		marker.DatabaseSHA256 = digest
	}
	if err := writePendingUpdateMarker(execPath, marker); err != nil {
		return errors.Join(err, removePendingDatabaseSnapshot(execPath, marker))
	}
	return nil
}

func newUpdateTransactionID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("generate update transaction id: %w", err)
	}
	return hex.EncodeToString(id[:]), nil
}

func pendingDatabaseSnapshotPath(execPath string, marker pendingUpdateMarker) string {
	if marker.TransactionID == "" || marker.DatabasePath == "" || marker.DatabaseSHA256 == "" {
		return ""
	}
	return execPath + ".update-db-" + marker.TransactionID + ".bak"
}

func removePendingDatabaseSnapshot(execPath string, marker pendingUpdateMarker) error {
	if path := pendingDatabaseSnapshotPath(execPath, marker); path != "" {
		return removeUpdateFile(path)
	}
	return nil
}

func writePendingUpdateMarker(execPath string, marker pendingUpdateMarker) error {
	encoded, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	return writeAtomicUpdateFile(execPath+pendingSuffix, encoded, 0o600)
}

func readPendingUpdateMarker(execPath string) (pendingUpdateMarker, error) {
	raw, err := os.ReadFile(execPath + pendingSuffix) // #nosec G304 -- fixed updater path.
	if err != nil {
		return pendingUpdateMarker{}, err
	}
	var marker pendingUpdateMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		return marker, fmt.Errorf("decode pending marker: %w", err)
	}
	validPhase := marker.Phase == updatePhasePrepared || marker.Phase == updatePhaseApplied || marker.Phase == updatePhaseBooting || marker.Phase == updatePhaseConfirmed
	if marker.Version != updateMarkerVersion || len(marker.TransactionID) != 32 || !validPhase || marker.Attempts < 0 || len(marker.CandidateSHA256) != sha256.Size*2 || len(marker.BackupSHA256) != sha256.Size*2 {
		return marker, errors.New("invalid pending update marker")
	}
	for name, value := range map[string]string{"transaction id": marker.TransactionID, "candidate digest": marker.CandidateSHA256, "backup digest": marker.BackupSHA256} {
		if _, err := hex.DecodeString(value); err != nil {
			return marker, fmt.Errorf("invalid %s: %w", name, err)
		}
	}
	if (marker.DatabasePath == "") != (marker.DatabaseSHA256 == "") {
		return marker, errors.New("invalid pending database snapshot metadata")
	}
	if marker.DatabaseSHA256 != "" {
		if len(marker.DatabaseSHA256) != sha256.Size*2 {
			return marker, errors.New("invalid database snapshot digest")
		}
		if _, err := hex.DecodeString(marker.DatabaseSHA256); err != nil {
			return marker, fmt.Errorf("invalid database snapshot digest: %w", err)
		}
		if PanelUpdateDatabaseRestore == nil {
			return marker, errors.New("database rollback handler is unavailable")
		}
	}
	return marker, nil
}

func markPendingUpdateApplied(execPath string) error {
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		return err
	}
	if marker.Phase != updatePhasePrepared {
		return fmt.Errorf("update transaction %s is in unexpected phase %q", marker.TransactionID, marker.Phase)
	}
	marker.Phase = updatePhaseApplied
	return writePendingUpdateMarker(execPath, marker)
}

// RecoverPendingUpdate runs before migration. Recovery failures are fatal.
func RecoverPendingUpdate(execPath string) (bool, error) {
	lock, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		return false, err
	}
	defer func() { _ = lock.release() }()
	return recoverPendingUpdateLocked(execPath)
}

func recoverPendingUpdateLocked(execPath string) (bool, error) {
	marker, err := readPendingUpdateMarker(execPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	backupDigest, err := updateFileSHA256(execPath + backupSuffix)
	if err != nil || !strings.EqualFold(backupDigest, marker.BackupSHA256) {
		return false, errors.Join(fmt.Errorf("update transaction %s backup identity mismatch", marker.TransactionID), err)
	}
	liveDigest, err := updateFileSHA256(execPath)
	if err != nil {
		return false, fmt.Errorf("verify transaction %s live binary: %w", marker.TransactionID, err)
	}
	if strings.EqualFold(liveDigest, marker.BackupSHA256) {
		if marker.Phase != updatePhasePrepared {
			return false, fmt.Errorf("applied transaction %s unexpectedly runs its backup binary", marker.TransactionID)
		}
		return false, cleanupPreparedUpdateLocked(execPath, marker)
	}
	if !strings.EqualFold(liveDigest, marker.CandidateSHA256) {
		return false, fmt.Errorf("update transaction %s candidate identity mismatch", marker.TransactionID)
	}
	if marker.Phase == updatePhasePrepared {
		marker.Phase = updatePhaseApplied
		if err := writePendingUpdateMarker(execPath, marker); err != nil {
			return rollbackPendingUpdateLocked(execPath, marker, err)
		}
	}
	if marker.Phase == updatePhaseBooting {
		return rollbackPendingUpdateLocked(execPath, marker, nil)
	}
	marker.Attempts++
	if marker.Attempts >= rollbackAfterAttempts {
		return rollbackPendingUpdateLocked(execPath, marker, nil)
	}
	if err := writePendingUpdateMarker(execPath, marker); err != nil {
		return rollbackPendingUpdateLocked(execPath, marker, err)
	}
	return false, nil
}

// MarkPendingUpdateBooting records that startup is about to migrate the DB.
func MarkPendingUpdateBooting(execPath string) error {
	lock, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		return err
	}
	defer func() { _ = lock.release() }()
	marker, err := readPendingUpdateMarker(execPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if marker.Phase != updatePhaseApplied {
		return fmt.Errorf("update transaction %s cannot boot from phase %q", marker.TransactionID, marker.Phase)
	}
	marker.Phase = updatePhaseBooting
	return writePendingUpdateMarker(execPath, marker)
}

// ConfirmPendingUpdate removes rollback artifacts only after a healthy start.
func ConfirmPendingUpdate(execPath string) error {
	lock, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		return err
	}
	defer func() { _ = lock.release() }()
	marker, err := readPendingUpdateMarker(execPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if marker.Phase != updatePhaseBooting && marker.Phase != updatePhaseConfirmed {
		return fmt.Errorf("update transaction %s did not complete its boot", marker.TransactionID)
	}
	if marker.Phase == updatePhaseBooting {
		liveDigest, liveErr := updateFileSHA256(execPath)
		backupDigest, backupErr := updateFileSHA256(execPath + backupSuffix)
		if liveErr != nil || backupErr != nil || !strings.EqualFold(liveDigest, marker.CandidateSHA256) || !strings.EqualFold(backupDigest, marker.BackupSHA256) {
			return errors.Join(fmt.Errorf("update transaction %s artifact identity mismatch", marker.TransactionID), liveErr, backupErr)
		}
		marker.Phase = updatePhaseConfirmed
		if err := writePendingUpdateMarker(execPath, marker); err != nil {
			return err
		}
	}
	if err := errors.Join(removeUpdateFile(execPath+backupSuffix), cleanupUpdateStagingFiles(execPath), removePendingDatabaseSnapshot(execPath, marker)); err != nil {
		return fmt.Errorf("clean confirmed update artifacts: %w", err)
	}
	if err := syncUpdateDirectory(filepath.Dir(execPath)); err != nil {
		return err
	}
	if err := removeUpdateFile(execPath + pendingSuffix); err != nil {
		return err
	}
	return syncUpdateDirectory(filepath.Dir(execPath))
}

func ClearPendingUpdate(execPath string) {
	if err := ConfirmPendingUpdate(execPath); err != nil {
		logger.Warning("panel update: could not confirm pending transaction: ", err)
	}
}

func CheckPendingUpdate(execPath string) bool {
	rolledBack, err := RecoverPendingUpdate(execPath)
	if err != nil {
		logger.Error("panel update: pending recovery failed: ", err)
	}
	return rolledBack
}

func rollbackCurrentUpdate(execPath string) error {
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		return err
	}
	_, err = rollbackPendingUpdateLocked(execPath, marker, nil)
	return err
}

func rollbackPendingUpdateLocked(execPath string, marker pendingUpdateMarker, cause error) (bool, error) {
	if snapshotPath := pendingDatabaseSnapshotPath(execPath, marker); snapshotPath != "" {
		digest, err := updateFileSHA256(snapshotPath)
		if err != nil || !strings.EqualFold(digest, marker.DatabaseSHA256) {
			return false, errors.Join(cause, fmt.Errorf("verify transaction %s database snapshot: %w", marker.TransactionID, err))
		}
		if err := PanelUpdateDatabaseRestore(snapshotPath, marker.DatabasePath); err != nil {
			return false, errors.Join(cause, fmt.Errorf("rollback transaction %s database: %w", marker.TransactionID, err))
		}
	}
	if err := RestoreBackup(execPath); err != nil {
		return false, errors.Join(cause, fmt.Errorf("rollback transaction %s binary: %w", marker.TransactionID, err))
	}
	if err := errors.Join(cleanupUpdateStagingFiles(execPath), removePendingDatabaseSnapshot(execPath, marker), removeUpdateFile(execPath+pendingSuffix), syncUpdateDirectory(filepath.Dir(execPath))); err != nil {
		return true, errors.Join(cause, err)
	}
	return true, nil
}

func cleanupPreparedUpdateLocked(execPath string, marker pendingUpdateMarker) error {
	return errors.Join(removeUpdateFile(execPath+backupSuffix), cleanupUpdateStagingFiles(execPath), removePendingDatabaseSnapshot(execPath, marker), removeUpdateFile(execPath+pendingSuffix), syncUpdateDirectory(filepath.Dir(execPath)))
}

func cleanupUpdateStagingFiles(execPath string) error {
	return errors.Join(removeUpdateFile(execPath+".new"), removeUpdateFile(execPath+backupSuffix+".new"), removeUpdateFile(execPath+backupSuffix+".previous"), removeUpdateFile(filepath.Join(filepath.Dir(execPath), ".sui-update.tar.gz")))
}

func updateFileSHA256(path string) (string, error) {
	file, err := os.Open(path) // #nosec G304 -- fixed updater-owned path.
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func removeUpdateFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func writeAtomicUpdateFile(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = removeUpdateFile(tmpPath) }()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return syncUpdateDirectory(filepath.Dir(path))
}

func syncUpdateDirectory(path string) error {
	// #nosec G304 -- path is the directory containing updater-owned files derived from the current executable path.
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if runtime.GOOS == "windows" {
		syncErr = nil
	}
	return errors.Join(syncErr, closeErr)
}
