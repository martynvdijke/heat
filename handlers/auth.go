package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"heat/app"
	"heat/models"
)

func HandleCheckSetup(c *gin.Context) {
	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	c.JSON(http.StatusOK, gin.H{"setup": count > 0})
}

func HandleLogin(c *gin.Context) {
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
	app.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)

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
		_, err := app.DB.Exec("INSERT INTO admin_users (username, password) VALUES (?, ?)", input.Username, hashed)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		sessionID := generateSessionID()
		app.SessionStoreMu.Lock()
		app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(24 * time.Hour).Unix(), IP: c.ClientIP()}
		app.SessionStoreMu.Unlock()
		setSessionCookie(c, sessionID)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	var user models.AdminUser
	err := app.DB.QueryRow("SELECT id, username, password FROM admin_users WHERE username = ?", input.Username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	sessionID := generateSessionID()
	app.SessionStoreMu.Lock()
	app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(24 * time.Hour).Unix(), IP: c.ClientIP()}
	app.SessionStoreMu.Unlock()
	setSessionCookie(c, sessionID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func HandleLogout(c *gin.Context) {
	cookie, err := c.Request.Cookie("session")
	if err == nil {
		app.SessionStoreMu.Lock()
		delete(app.SessionStore, cookie.Value)
		app.SessionStoreMu.Unlock()
	}
	c.SetCookie("session", "", -1, "/", "", app.SecureCookies, true)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func setSessionCookie(c *gin.Context, sessionID string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session", sessionID, 86400, "/", "", app.SecureCookies, true)
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}
