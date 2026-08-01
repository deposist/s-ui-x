package web

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestServeAssetsFSServesJavaScriptWithMime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/assets/*filepath", serveAssetsFS(fstest.MapFS{
		"chunk.js": {Data: []byte("export default 1")},
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/chunk.js", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "javascript") {
		t.Fatalf("Content-Type=%q, want JavaScript MIME", contentType)
	}
}

func TestServeAssetsFSMissingFileReturns404WithoutHTMLFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/assets/*filepath", serveAssetsFS(fstest.MapFS{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusNotFound)
	}
	if strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("missing asset used HTML fallback: Content-Type=%q", rec.Header().Get("Content-Type"))
	}
}

func TestEmbeddedBuiltJavaScriptAssetServesWithMime(t *testing.T) {
	assetsFS, err := fs.Sub(content, "html/assets")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(assetsFS, ".")
	if err != nil {
		t.Fatal(err)
	}

	var jsAsset string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
			jsAsset = entry.Name()
			break
		}
	}
	if jsAsset == "" {
		t.Fatal("expected at least one built JavaScript asset in embedded html/assets")
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/app/assets/*filepath", serveAssetsFS(assetsFS))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app/assets/"+jsAsset, nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d for %s", rec.Code, http.StatusOK, jsAsset)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "javascript") {
		t.Fatalf("Content-Type=%q, want JavaScript MIME for %s", contentType, jsAsset)
	}
}
func TestEmbeddedProductionAssetsMatchEmbedFS(t *testing.T) {
	disk := os.DirFS(".")
	diskFiles := 0
	err := fs.WalkDir(disk, "html", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			t.Errorf("installed asset is not a regular file: %s", path)
			return nil
		}
		diskFiles++
		embeddedPath := path
		diskData, readErr := fs.ReadFile(disk, path)
		if readErr != nil {
			return readErr
		}
		embeddedData, embeddedErr := fs.ReadFile(content, embeddedPath)
		if embeddedErr != nil {
			t.Errorf("installed asset is missing from embed.FS: %s: %v", path, embeddedErr)
			return nil
		}
		if !bytes.Equal(diskData, embeddedData) {
			t.Errorf("embedded asset differs from installed asset: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if diskFiles == 0 {
		t.Fatal("expected installed web/html assets")
	}

	embeddedFiles := 0
	err = fs.WalkDir(content, "html", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			embeddedFiles++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if embeddedFiles != diskFiles {
		t.Fatalf("embed.FS file count=%d, installed asset file count=%d", embeddedFiles, diskFiles)
	}
}
