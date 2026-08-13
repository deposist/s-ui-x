package importxui

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/deposist/s-ui-x/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type sourceDB struct {
	db      *gorm.DB
	dialect Dialect
}

type xuiInboundRow struct {
	ID                   int64
	UserID               int64
	Up                   int64
	Down                 int64
	Total                int64
	AllTime              int64
	Remark               string
	Enable               bool
	ExpiryTime           int64
	TrafficReset         string
	LastTrafficResetTime int64
	Listen               string
	Port                 int
	Protocol             string
	Settings             json.RawMessage
	StreamSettings       json.RawMessage
	Tag                  string
	Sniffing             json.RawMessage
}

type xuiClientTraffic struct {
	ID         int64
	InboundID  int64
	Enable     bool
	Email      string
	Up         int64
	Down       int64
	AllTime    int64
	ExpiryTime int64
	Total      int64
	Reset      int64
	LastOnline int64
}

type xuiSetting struct {
	ID    int64
	Key   string
	Value string
}

type xuiOutboundTraffic struct {
	ID   int64
	Tag  string
	Up   int64
	Down int64
}

func openSource(path string) (*sourceDB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("missing source path")
	}
	dsn, err := sqliteReadOnlyURI(path)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		return nil, err
	}
	src := &sourceDB{db: db}
	if err := src.validate(); err != nil {
		src.close()
		return nil, err
	}
	return src, nil
}

func sqliteReadOnlyURI(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		return path + sep + "mode=ro&immutable=1&_pragma=query_only(true)&_pragma=trusted_schema(OFF)", nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	urlPath := filepath.ToSlash(abs)
	if runtime.GOOS == "windows" && !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	u := url.URL{
		Scheme: "file",
		Path:   urlPath,
	}
	values := url.Values{}
	values.Set("mode", "ro")
	values.Set("immutable", "1")
	values.Add("_pragma", "query_only(true)")
	values.Add("_pragma", "trusted_schema(OFF)")
	u.RawQuery = values.Encode()
	return u.String(), nil
}

func (s *sourceDB) close() {
	sqlDB, err := s.sqlDB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

func (s *sourceDB) validate() error {
	_ = s.db.Exec("PRAGMA trusted_schema=OFF").Error
	sqlDB, err := s.sqlDB()
	if err != nil {
		return err
	}
	for _, dialect := range RegisteredDialects() {
		ok, err := dialect.Detect(sqlDB)
		if err != nil {
			return err
		}
		if ok {
			s.dialect = dialect
			return nil
		}
	}
	return ErrDialectUnknown
}

func hashSource(path string) (string, error) {
	// #nosec G304 -- path is an operator-supplied import source file validated by the caller.
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func (s *sourceDB) eachInbound(fn func(xuiInboundRow) error) error {
	db, err := s.sqlDB()
	if err != nil {
		return err
	}
	rows, err := s.dialect.ReadInbounds(db)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := fn(row); err != nil {
			return err
		}
	}
	return nil
}

func (s *sourceDB) eachClientTraffic(fn func(xuiClientTraffic) error) error {
	db, err := s.sqlDB()
	if err != nil {
		return err
	}
	rows, err := s.dialect.ReadClients(db)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := fn(row); err != nil {
			return err
		}
	}
	return nil
}

func (s *sourceDB) inboundCount() (int, error) {
	db, err := s.sqlDB()
	if err != nil {
		return 0, err
	}
	rows, err := s.dialect.ReadInbounds(db)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *sourceDB) settings() ([]xuiSetting, error) {
	db, err := s.sqlDB()
	if err != nil {
		return nil, err
	}
	return s.dialect.ReadSettings(db)
}

type xuiUser struct {
	ID       int64
	Username string
	Password string
}

func (s *sourceDB) users() ([]xuiUser, error) {
	db, err := s.sqlDB()
	if err != nil {
		return nil, err
	}
	return s.dialect.ReadUsers(db)
}

func (s *sourceDB) outboundTraffics() ([]xuiOutboundTraffic, error) {
	db, err := s.sqlDB()
	if err != nil {
		return nil, err
	}
	return s.dialect.ReadOutboundTraffics(db)
}

func (s *sourceDB) xrayConfig() (string, error) {
	db, err := s.sqlDB()
	if err != nil {
		return "", err
	}
	return s.dialect.ReadXrayConfig(db)
}

var errXUISourceUnavailable = errors.New("xui source database is unavailable")

func (s *sourceDB) sqlDB() (*sql.DB, error) {
	if s == nil || s.db == nil {
		return nil, errXUISourceUnavailable
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil, fmt.Errorf("get xui source database: %w", err)
	}
	if sqlDB == nil {
		return nil, errXUISourceUnavailable
	}
	return sqlDB, nil
}

func nullString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func nullJSON(v sql.NullString) json.RawMessage {
	value := strings.TrimSpace(nullString(v))
	if value == "" {
		return nil
	}
	return json.RawMessage(value)
}

// ValidateSQLiteSource confirms that path is a readable, integrity-clean SQLite
// file in a recognised x-ui dialect before it is used as an import source.
func ValidateSQLiteSource(path string) error {
	// #nosec G304 -- path is an operator-supplied import source file validated by the caller.
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	ok, err := database.IsSQLiteDB(file)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("not_sqlite")
	}
	src, err := openSource(path)
	if err != nil {
		return err
	}
	defer src.close()
	var result string
	if err := src.db.Raw("PRAGMA integrity_check").Scan(&result).Error; err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("invalid sqlite integrity: %s", result)
	}
	return nil
}
