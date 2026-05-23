package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/middleware"
)

func TestHtmxQuotesCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"text":   "HTMX Test Quote",
		"author": "HTMX Tester",
	}
	req := newHtmxAdminFormRequest("/api/html/quotes", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Quote") {
		t.Errorf("expected new quote in table, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxQuotesDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	testServer.DB.QueryRow("SELECT id FROM quotes LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no quotes to delete")
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/quotes/%d", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "quote-list") {
		t.Errorf("expected quote list after delete")
	}
}

func TestHtmxQuotesEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	testServer.DB.QueryRow("SELECT id FROM quotes LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no quotes")
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/html/quotes/%d/edit", id), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Save Quote") {
		t.Errorf("expected quote edit form, got: %s", body[:min(len(body), 200)])
	}
}

func TestHtmxQuotesTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/quotes", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "quote-list") {
		t.Errorf("expected quote-list tbody")
	}
}

func TestHtmxRacersCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name":            "HTMX Racer",
		"car_name":        "HTMX Car",
		"car_color":       "#ff0000",
		"points":          "100",
		"rank":            "1",
		"position":        "0",
		"profile_picture": "/static/images/helmet.svg",
	}
	req := newHtmxAdminFormRequest("/api/html/racers", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Racer") {
		t.Errorf("expected new racer in table HTML, got: %s", body[:min(len(body), 300)])
	}
	trigger := rr.Header().Get("HX-Trigger")
	if !strings.Contains(trigger, "closeRacerModal") {
		t.Errorf("expected HX-Trigger closeRacerModal, got: %s", trigger)
	}
}

func TestHtmxRacersCreateMissingName(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name": "",
	}
	req := newHtmxAdminFormRequest("/api/html/racers", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", rr.Code)
	}
}

func TestHtmxRacersDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	testServer.DB.QueryRow("SELECT id FROM racers LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no racers to delete")
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/racers/%d", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<tbody") {
		t.Errorf("expected HTML table after delete")
	}
}

func TestHtmxRacersEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	testServer.DB.QueryRow("SELECT id FROM racers LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no racers")
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/html/racers/%d/edit", id), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "hx-post") || !strings.Contains(body, "Save Racer") {
		t.Errorf("expected edit form with htmx attributes, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxRacersGenerateShare(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	testServer.DB.QueryRow("SELECT id FROM racers LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no racers")
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/html/racers/%d/share", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var token string
	err := testServer.DB.QueryRow("SELECT token FROM driver_shares WHERE racer_id = ?", id).Scan(&token)
	if err != nil {
		t.Errorf("expected share token to exist: %v", err)
	}
}

func TestHtmxRacersNewForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/racers/0/edit", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "hx-post") || !strings.Contains(body, "Save Racer") {
		t.Errorf("expected new racer form, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxRacersTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/racers", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<tbody") {
		t.Errorf("expected HTML table body, got: %s", body[:min(len(body), 200)])
	}
	if !strings.Contains(body, "racer-list") {
		t.Errorf("expected racer-list id in response")
	}
	if strings.Count(body, "<tr>") < 2 {
		t.Errorf("expected at least 2 racer rows (5 seeded), got fewer")
	}
}

func TestHtmxSeasonsArchive(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req := newHtmxAdminFormRequest("/api/html/seasons", map[string]string{"name": "Archive Test Season"}, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create season: expected 200, got %d", rr.Code)
	}

	var id int
	testServer.DB.QueryRow("SELECT id FROM seasons WHERE name = 'Archive Test Season'").Scan(&id)
	if id == 0 {
		t.Fatal("no season created")
	}

	req2, _ := http.NewRequest("POST", fmt.Sprintf("/api/html/seasons/%d/archive", id), nil)
	req2.Header.Set("Origin", "http://127.0.0.1:6270")
	req2.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("archive: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var status string
	testServer.DB.QueryRow("SELECT status FROM seasons WHERE id = ?", id).Scan(&status)
	if status != "archived" {
		t.Errorf("expected status 'archived', got %q", status)
	}
}

func TestHtmxSeasonsCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name": "HTMX Test Season",
	}
	req := newHtmxAdminFormRequest("/api/html/seasons", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Season") {
		t.Errorf("expected new season in table, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxSeasonsDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req := newHtmxAdminFormRequest("/api/html/seasons", map[string]string{"name": "Del Test Season"}, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", rr.Code)
	}

	var id int
	testServer.DB.QueryRow("SELECT id FROM seasons WHERE name = 'Del Test Season'").Scan(&id)
	if id == 0 {
		t.Fatal("no season created")
	}

	req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/seasons/%d", id), nil)
	req2.Header.Set("Origin", "http://127.0.0.1:6270")
	req2.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestHtmxSeasonsNewForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/seasons/new", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Season Name") || !strings.Contains(body, "hx-post") {
		t.Errorf("expected season creation form, got: %s", body[:min(len(body), 200)])
	}
}

func TestHtmxSeasonsTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/seasons", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "seasons-list") {
		t.Errorf("expected seasons-list tbody")
	}
}

func TestHtmxTeamsCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name":  "HTMX Test Team",
		"color": "#00ff00",
	}
	req := newHtmxAdminFormRequest("/api/html/teams", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Team") {
		t.Errorf("expected new team in table")
	}
}

func TestHtmxTeamsDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	testServer.DB.Exec("INSERT INTO teams (name, color) VALUES ('Del Team', '#000')")
	var id int
	testServer.DB.QueryRow("SELECT id FROM teams WHERE name = 'Del Team'").Scan(&id)
	if id == 0 {
		t.Fatal("no team to delete")
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/teams/%d", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHtmxTeamsEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	testServer.DB.Exec("INSERT INTO teams (name, color) VALUES ('Edit Team Test', '#fff')")
	var id int
	testServer.DB.QueryRow("SELECT id FROM teams WHERE name = 'Edit Team Test'").Scan(&id)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/html/teams/%d/edit", id), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Save Team") {
		t.Errorf("expected team edit form, got: %s", body[:min(len(body), 200)])
	}
	testServer.DB.Exec("DELETE FROM teams WHERE id = ?", id)
}

func TestHtmxTeamsTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/teams", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "team-list") {
		t.Errorf("expected team-list tbody")
	}
}

func TestHtmxTracksCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"id":         "htmx-test-track",
		"id_visible": "htmx-test-track",
		"name":       "HTMX Test Track",
		"country":    "Testland",
		"length_km":  "42",
		"lap_record": "1:30.000",
	}
	req := newHtmxAdminFormRequest("/api/html/tracks", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Track") {
		t.Errorf("expected new track in table, got: %s", body[:min(len(body), 300)])
	}
	testServer.DB.Exec("DELETE FROM tracks WHERE id = 'htmx-test-track'")
}

func TestHtmxTracksDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	testServer.DB.Exec("INSERT OR REPLACE INTO tracks (id, name, country, length_km, lap_record) VALUES ('htmx-del-track', 'Del Track', 'Nowhere', 10, '--')")

	req, _ := http.NewRequest("DELETE", "/api/html/tracks/htmx-del-track", nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "track-list") {
		t.Errorf("expected track list after delete")
	}
}

func TestHtmxTracksEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	testServer.DB.Exec("INSERT OR REPLACE INTO tracks (id, name, country, length_km, lap_record) VALUES ('edit-test', 'Edit Test', 'Test', 10, '--')")

	req, _ := http.NewRequest("GET", "/api/html/tracks/edit-test/edit", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Save Track") {
		t.Errorf("expected track edit form, got: %s", body[:min(len(body), 200)])
	}
	testServer.DB.Exec("DELETE FROM tracks WHERE id = 'edit-test'")
}

func TestHtmxTracksTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/tracks", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "track-list") {
		t.Errorf("expected track-list tbody")
	}
}

func TestHtmxUnauthorized(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.GET("/html/racers", testHandler.HtmxRacersTable)

	req, _ := http.NewRequest("GET", "/api/html/racers", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without session, got %d", rr.Code)
	}
}
