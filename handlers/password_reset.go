package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

const resetTokenTTL = 30 * time.Minute

const resetTimeLayout = "2006-01-02 15:04:05"

// @Summary Request a password reset
// @Description Request a password reset link for an admin account. Always returns 200 to avoid user enumeration.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body object true "Email address"
// @Success 200 {object} map[string]string
// @Router /api/forgot-password [post]
func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	input.Email = strings.TrimSpace(input.Email)

	// Anti-enumeration: always respond 200, even for unknown emails.
	var userID int
	err := h.S.DB.QueryRow("SELECT id FROM admin_users WHERE email = ?", input.Email).Scan(&userID)
	if err != nil {
		if err != sql.ErrNoRows {
			h.S.Log.Errorf("password-reset", "RequestPasswordReset: lookup error: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	token := generateResetToken()
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	expiresAt := time.Now().Add(resetTokenTTL).Format(resetTimeLayout)
	if _, err := h.S.DB.Exec("INSERT INTO password_reset_tokens (admin_user_id, token, expires_at) VALUES (?, ?, ?)", userID, token, expiresAt); err != nil {
		h.S.Log.Errorf("password-reset", "RequestPasswordReset: failed to store token: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	resetURL := trmnlAbsoluteURL(c, "/reset-password.html?token="+token)
	if err := h.sendPasswordResetEmail(input.Email, resetURL); err != nil {
		h.S.Log.Errorf("password-reset", "RequestPasswordReset: failed to send email to %q: %v", input.Email, err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Validate a password reset token
// @Description Check whether a password reset token is valid and unexpired
// @Tags Auth
// @Produce json
// @Param token query string true "Reset token"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]bool
// @Router /api/reset-password/validate [get]
func (h *Handler) ValidateResetToken(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if !h.isValidResetToken(token) {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": "Invalid or expired reset token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// @Summary Reset password with a token
// @Description Set a new password using a valid reset token, then log in automatically
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body object true "Reset token and new password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var input struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Token = strings.TrimSpace(input.Token)

	if len(input.Password) < 8 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters"})
		return
	}

	var userID int
	var used int
	var expiresAt string
	err := h.S.DB.QueryRow("SELECT admin_user_id, used, expires_at FROM password_reset_tokens WHERE token = ?", input.Token).Scan(&userID, &used, &expiresAt)
	if err != nil || used != 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}
	exp, err := time.Parse(resetTimeLayout, expiresAt)
	if err != nil || !time.Now().Before(exp) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}

	hashed := hashPassword(input.Password)
	if _, err := h.S.DB.Exec("UPDATE admin_users SET password = ? WHERE id = ?", hashed, userID); err != nil {
		h.S.Log.Errorf("password-reset", "ResetPassword: failed to update password for user %d: %v", userID, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}
	h.S.DB.Exec("UPDATE password_reset_tokens SET used = 1 WHERE token = ?", input.Token)

	// Auto-login: create a session for the admin whose password was reset.
	sessionID := generateSessionID()
	h.S.SessionStoreMu.Lock()
	h.S.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(24 * time.Hour).Unix(), IP: c.ClientIP()}
	h.S.SessionStoreMu.Unlock()
	setSessionCookie(c, sessionID, h.S.SecureCookies)

	h.S.Log.Infof("password-reset", "Password reset for user %d", userID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) isValidResetToken(token string) bool {
	if token == "" {
		return false
	}
	var used int
	var expiresAt string
	if err := h.S.DB.QueryRow("SELECT used, expires_at FROM password_reset_tokens WHERE token = ?", token).Scan(&used, &expiresAt); err != nil {
		return false
	}
	if used != 0 {
		return false
	}
	exp, err := time.Parse(resetTimeLayout, expiresAt)
	if err != nil {
		return false
	}
	return time.Now().Before(exp)
}

func generateResetToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (h *Handler) sendPasswordResetEmail(to, resetURL string) error {
	var s models.EmailSettings
	var enabled int
	err := h.S.DB.QueryRow("SELECT smtp_host, COALESCE(smtp_port, 587), username, password, from_addr, COALESCE(enabled, 0) FROM email_settings WHERE id = 1").
		Scan(&s.SMTPHost, &s.SMTPPort, &s.Username, &s.Password, &s.FromAddr, &enabled)
	if err != nil || enabled == 0 || s.SMTPHost == "" || s.FromAddr == "" {
		return fmt.Errorf("SMTP not configured")
	}
	return sendSMTP(s, to, buildPasswordResetEmailContent(resetURL))
}

func buildPasswordResetEmailContent(resetURL string) string {
	var b strings.Builder
	b.WriteString("Subject: HEAT - Reset your password\n")
	b.WriteString("MIME-Version: 1.0\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\n\n")
	b.WriteString("<!DOCTYPE html><html><head><style>")
	b.WriteString("body{font-family:Arial,sans-serif;background:#111;color:#eee;padding:20px}")
	b.WriteString("h1{color:#d40000;border-bottom:3px solid #d40000;padding-bottom:10px}")
	b.WriteString(".btn{display:inline-block;background:#d40000;color:#fff;padding:12px 24px;text-decoration:none;border-radius:5px;font-weight:bold}")
	b.WriteString("</style></head><body>")
	b.WriteString("<h1>HEAT - Reset your password</h1>")
	b.WriteString("<p>We received a request to reset the password for your HEAT admin account. Click the button below to choose a new one. This link expires in 30 minutes.</p>")
	b.WriteString(fmt.Sprintf("<p><a class=\"btn\" href=\"%s\">Reset Password</a></p>", html.EscapeString(resetURL)))
	b.WriteString("<p style=\"color:#666;font-size:12px;margin-top:30px\">If you didn't request this, you can safely ignore this email.</p>")
	b.WriteString("</body></html>")
	return b.String()
}
