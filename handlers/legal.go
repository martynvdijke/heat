package handlers

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// LegalData is everything the public policy pages are allowed to assert. Each
// processor flag is read from the live settings row (or environment) at
// request time, so the text cannot describe a feature this deployment does not
// actually use, and cannot silently omit one it does.
type LegalData struct {
	Controller     string
	ContactEmail   string
	ContactAddress string
	HostingRegion  string
	BaseURL        string
	LastUpdated    string

	AdminAccounts bool
	RacerIdentity bool

	SMTP     bool
	Telegram bool
	Gotify   bool
	Umami    bool
	OTLP     bool
	WLED     bool
	AI       bool
	OIDC     bool
}

type legalPageData struct {
	Version         string
	Title           string
	MetaDescription string
	Legal           LegalData
}

var legalTemplates sync.Map

func loadLegalTemplate(basePath, name string) (*template.Template, error) {
	key := filepath.Join(basePath, "static/templates", name)
	if cached, ok := legalTemplates.Load(key); ok {
		return cached.(*template.Template), nil
	}
	tmpl, err := template.ParseFiles(
		filepath.Join(basePath, "static/templates/base.html"),
		filepath.Join(basePath, "static/templates", name),
	)
	if err != nil {
		return nil, err
	}
	legalTemplates.Store(key, tmpl)
	return tmpl, nil
}

func (h *Handler) legal() LegalData {
	cfg := h.S.Legal
	d := LegalData{
		Controller:     cfg.Controller,
		ContactEmail:   cfg.ContactEmail,
		ContactAddress: cfg.ContactAddress,
		HostingRegion:  cfg.HostingRegion,
		BaseURL:        cfg.BaseURL,
		LastUpdated:    "1 November 2026",
		AdminAccounts:  true,
		RacerIdentity:  true,
		OIDC:           h.S.OIDC.Enabled,
	}
	if d.Controller == "" {
		d.Controller = "the operator of this HEAT instance"
	}
	if d.HostingRegion == "" {
		d.HostingRegion = "the European Union"
	}
	if d.BaseURL == "" {
		d.BaseURL = "https://heat.vandijke.xyz"
	}

	if h.S.DB == nil {
		return d
	}
	var enabled int
	var value string
	if err := h.S.DB.QueryRow("SELECT enabled, COALESCE(smtp_host, '') FROM email_settings WHERE id = 1").Scan(&enabled, &value); err == nil {
		d.SMTP = enabled == 1 && strings.TrimSpace(value) != ""
	}
	if err := h.S.DB.QueryRow("SELECT enabled, COALESCE(bot_token, '') FROM telegram_settings WHERE id = 1").Scan(&enabled, &value); err == nil {
		d.Telegram = enabled == 1 && strings.TrimSpace(value) != ""
	}
	if err := h.S.DB.QueryRow("SELECT COALESCE(gotify_url, '') FROM notification_settings WHERE id = 1").Scan(&value); err == nil {
		d.Gotify = strings.TrimSpace(value) != ""
	}
	if err := h.S.DB.QueryRow("SELECT enabled, COALESCE(url, '') FROM umami_settings WHERE id = 1").Scan(&enabled, &value); err == nil {
		d.Umami = enabled == 1 && strings.TrimSpace(value) != ""
	}
	var otelTraces, otelMetrics, otelLogs int
	var otelEndpoint string
	if err := h.S.DB.QueryRow("SELECT traces_enabled, metrics_enabled, logs_enabled, COALESCE(endpoint, '') FROM otel_settings WHERE id = 1").Scan(&otelTraces, &otelMetrics, &otelLogs, &otelEndpoint); err == nil {
		d.OTLP = strings.TrimSpace(otelEndpoint) != "" && (otelTraces == 1 || otelMetrics == 1 || otelLogs == 1)
	}
	if err := h.S.DB.QueryRow("SELECT enabled, COALESCE(url, '') FROM wled_settings WHERE id = 1").Scan(&enabled, &value); err == nil {
		d.WLED = enabled == 1 && strings.TrimSpace(value) != ""
	}
	var aiEnabled int
	var aiTrackURL, aiTextURL string
	if err := h.S.DB.QueryRow("SELECT enabled, COALESCE(track_extract_url, ''), COALESCE(text_gen_url, '') FROM ai_settings WHERE id = 1").Scan(&aiEnabled, &aiTrackURL, &aiTextURL); err == nil {
		d.AI = aiEnabled == 1 && (strings.TrimSpace(aiTrackURL) != "" || strings.TrimSpace(aiTextURL) != "")
	}
	return d
}

func (h *Handler) renderLegalPage(c *gin.Context, name, title, description string) {
	tmpl, err := loadLegalTemplate(h.S.BasePath, name)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	data := legalPageData{
		Version:         h.S.CurrentVersion,
		Title:           title,
		MetaDescription: description,
		Legal:           h.legal(),
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// PrivacyPage serves the GDPR Article 13 privacy notice. It is public: the
// policy must be readable before anyone creates an identity.
func (h *Handler) PrivacyPage(c *gin.Context) {
	h.renderLegalPage(c, "privacy.html", "Privacy", "What this HEAT instance stores, why, who processes it, and how to get it back or delete it.")
}

// TermsPage serves the acceptable-use terms.
func (h *Handler) TermsPage(c *gin.Context) {
	h.renderLegalPage(c, "terms.html", "Terms of use", "The rules for using this HEAT instance and what you keep ownership of.")
}

// AccountDeletePage is the public deletion instructions page Google Play
// requires for apps with accounts. It is reachable while signed out so a user
// who has lost access, and a store reviewer, can both follow it.
func (h *Handler) AccountDeletePage(c *gin.Context) {
	h.renderLegalPage(c, "account-delete.html", "Delete your account", "How to delete a HEAT admin account or racer identity, and what is removed or retained.")
}
