package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"heat/app"
	"heat/models"
)

// @Summary Check if admin is set up
// @Description Returns whether an admin user has been created
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]bool
// @Router /api/check-setup [get]
func (h *Handler) HandleCheckSetup(c *gin.Context) {
	var count int
	h.S.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	c.JSON(http.StatusOK, gin.H{"setup": count > 0})
}

// @Summary Login or create admin account
// @Description Login with existing credentials or create the first admin account during setup
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body object true "Login credentials"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/login [post]
func (h *Handler) HandleLogin(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Setup    bool   `json:"setup"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[LOGIN] Failed to decode JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int
	h.S.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)

	if input.Setup && count > 0 {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Setup already completed"})
		return
	}

	if count == 0 {
		if len(input.Password) < 8 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters"})
			return
		}
		hashed := hashPassword(input.Password)
		_, err := h.S.DB.Exec("INSERT INTO admin_users (username, password) VALUES (?, ?)", input.Username, hashed)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		sessionID := generateSessionID()
		h.S.SessionStoreMu.Lock()
		h.S.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(24 * time.Hour).Unix(), IP: c.ClientIP()}
		h.S.SessionStoreMu.Unlock()
		setSessionCookie(c, sessionID, h.S.SecureCookies)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	var user models.AdminUser
	err := h.S.DB.QueryRow("SELECT id, username, password FROM admin_users WHERE username = ?", input.Username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		if !checkLegacyPassword(user.Password, input.Password) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		upgradePassword(user.ID, input.Password, h.S.DB)
	}

	sessionID := generateSessionID()
	h.S.SessionStoreMu.Lock()
	h.S.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(24 * time.Hour).Unix(), IP: c.ClientIP()}
	h.S.SessionStoreMu.Unlock()
	setSessionCookie(c, sessionID, h.S.SecureCookies)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Logout
// @Description Clear the session cookie and logout
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/logout [post]
func (h *Handler) HandleLogout(c *gin.Context) {
	cookie, err := c.Request.Cookie("session")
	if err == nil {
		h.S.SessionStoreMu.Lock()
		delete(h.S.SessionStore, cookie.Value)
		h.S.SessionStoreMu.Unlock()
	}
	c.SetCookie("session", "", -1, "/", "", h.S.SecureCookies, true)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func setSessionCookie(c *gin.Context, sessionID string, secureCookies bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session", sessionID, 86400, "/", "", secureCookies, true)
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

func checkLegacyPassword(stored, input string) bool {
	if len(stored) == 0 || stored[0] == '$' {
		return false
	}
	hash := sha256.Sum256([]byte(input))
	return base64.StdEncoding.EncodeToString(hash[:]) == stored
}

func upgradePassword(userID int, password string, db *sql.DB) {
	newHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	db.Exec("UPDATE admin_users SET password = ? WHERE id = ?", string(newHash), userID)
}
