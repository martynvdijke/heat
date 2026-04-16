package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestMain(m *testing.M) {
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	initDB()

	os.Exit(m.Run())
}

func TestHashPassword(t *testing.T) {
	password := "password123"
	
	hash := hashPassword(password)
	
	if hash == "" {
		t.Error("Expected hash to be non-empty")
	}
	
	// Test consistency
	if hash != hashPassword(password) {
		t.Error("Expected hash to be consistent")
	}
	
	// Test difference
	if hash == hashPassword("anotherpassword") {
		t.Error("Expected different passwords to have different hashes")
	}
}

func TestHandleCheckSetup(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/check-setup", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleCheckSetup)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]bool
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["setup"] != false {
		t.Errorf("expected setup to be false, got %v", response["setup"])
	}
}

func TestGetRacers(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/racers", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getRacers)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var racers []Racer
	err = json.Unmarshal(rr.Body.Bytes(), &racers)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(racers) != 5 {
		t.Errorf("expected 5 racers, got %d", len(racers))
	}
}

func TestAuthMiddleware(t *testing.T) {
	// Create a dummy handler that should only be called if authorized
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Authorized")
	})

	handler := authMiddleware(dummyHandler)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status %v, got %v", http.StatusUnauthorized, status)
		}
	})

	t.Run("Authorized", func(t *testing.T) {
		sessionID := "test-session"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		req, _ := http.NewRequest("GET", "/api/racers", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status %v, got %v", http.StatusOK, status)
		}

		if rr.Body.String() != "Authorized" {
			t.Errorf("expected body 'Authorized', got '%s'", rr.Body.String())
		}
	})
}

func TestRaceInfo(t *testing.T) {
	t.Run("GetRaceInfo", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-info", nil)
		rr := httptest.NewRecorder()
		getRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var ri RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &ri)
		if ri.Country != "Italy" || ri.Track != "Monza" {
			t.Errorf("unexpected race info: %+v", ri)
		}
	})

	t.Run("UpdateRaceInfo", func(t *testing.T) {
		ri := RaceInfo{Country: "Belgium", Track: "Spa", Laps: 44}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify update
		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		getRaceInfo(rr, req)
		var updatedRi RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updatedRi)
		if updatedRi.Country != "Belgium" || updatedRi.Track != "Spa" || updatedRi.Laps != 44 {
			t.Errorf("race info not updated correctly: %+v", updatedRi)
		}
	})
}

func TestUpdateAndDeleteRacer(t *testing.T) {
	var racerID int

	t.Run("InsertRacer", func(t *testing.T) {
		newRacer := Racer{Name: "L. HAMILTON", CarColor: "black", CarName: "Silver Arrow", Points: 0, Rank: 6}
		body, _ := json.Marshal(newRacer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRacer(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Get the inserted racer to find its ID
		rows, _ := db.Query("SELECT id FROM racers WHERE name='L. HAMILTON'")
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&racerID)
		} else {
			t.Fatal("racer not inserted")
		}
	})

	t.Run("UpdateRacer", func(t *testing.T) {
		updatedRacer := Racer{ID: racerID, Name: "L. HAMILTON", CarColor: "purple", CarName: "W12", Points: 25, Rank: 1}
		body, _ := json.Marshal(updatedRacer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRacer(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify update
		var name string
		db.QueryRow("SELECT name FROM racers WHERE id=?", racerID).Scan(&name)
		if name != "L. HAMILTON" {
			t.Errorf("expected name L. HAMILTON, got %s", name)
		}
	})

	t.Run("DeleteRacer", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/racers?id=%d", racerID), nil)
		rr := httptest.NewRecorder()
		deleteRacer(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify deletion
		var count int
		db.QueryRow("SELECT COUNT(*) FROM racers WHERE id=?", racerID).Scan(&count)
		if count != 0 {
			t.Error("racer not deleted")
		}
	})
}

func TestLoginAndSetup(t *testing.T) {
	// 1. Initial check-setup (should be false)
	t.Run("CheckSetupInitial", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/check-setup", nil)
		rr := httptest.NewRecorder()
		handleCheckSetup(rr, req)
		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["setup"] != false {
			t.Errorf("expected setup false, got %v", resp["setup"])
		}
	})

	// 2. Perform first-time setup (create admin)
	t.Run("FirstTimeSetup", func(t *testing.T) {
		loginData := map[string]interface{}{
			"username": "admin",
			"password": "password",
			"setup":    true,
		}
		body, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		handleLogin(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Check if cookie is set
		cookies := rr.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "session" {
				found = true
				break
			}
		}
		if !found {
			t.Error("session cookie not found after setup login")
		}
	})

	// 3. check-setup (should be true now)
	t.Run("CheckSetupAfter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/check-setup", nil)
		rr := httptest.NewRecorder()
		handleCheckSetup(rr, req)
		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["setup"] != true {
			t.Errorf("expected setup true, got %v", resp["setup"])
		}
	})
}
