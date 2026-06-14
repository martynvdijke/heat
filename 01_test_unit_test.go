package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestShorten(t *testing.T) {
	if shorten("short") != "short" {
		t.Error("shorten should return short strings unchanged")
	}
	long := "this is a very long string that should be shortened"
	result := shorten(long)
	if len(result) > 20 {
		t.Errorf("shorten should truncate long strings, got: %s", result)
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("shorten should add ellipsis, got: %s", result)
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{`"quote"`, "&quot;quote&quot;"},
		{"normal text", "normal text"},
		{"", ""},
	}

	for _, tt := range tests {
		result := escapeHTML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestPWAManifest(t *testing.T) {
	r := gin.New()
	r.StaticFile("/sw.js", "./static/sw.js")

	t.Run("serve service worker", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/sw.js", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		if rr.Body.Len() == 0 {
			t.Error("expected non-empty service worker")
		}
	})

	t.Run("manifest exists", func(t *testing.T) {
		data, err := os.ReadFile("static/manifest.json")
		if err != nil {
			t.Fatal("manifest.json not found")
		}
		var m map[string]any
		json.Unmarshal(data, &m)
		if m["name"] == nil || m["short_name"] == nil {
			t.Error("manifest missing required fields")
		}
	})

	t.Run("all HTML pages have manifest link", func(t *testing.T) {
		files, _ := filepath.Glob("static/*.html")
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			if !strings.Contains(string(data), "manifest.json") {
				t.Errorf("%s missing manifest link", f)
			}
			if !strings.Contains(string(data), "theme-color") {
				t.Errorf("%s missing theme-color", f)
			}
		}
	})
}
