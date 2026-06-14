package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	translations   = make(map[string]map[string]string)
	translationsMu sync.RWMutex
)

func (h *Handler) loadTranslations(lang string) map[string]string {
	translationsMu.RLock()
	cached, ok := translations[lang]
	translationsMu.RUnlock()
	if ok {
		return cached
	}

	path := filepath.Join(h.S.BasePath, "static", "locales", lang+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join(h.S.BasePath, "static", "locales", "en.json")
		data, err = os.ReadFile(path)
		if err != nil {
			return nil
		}
	}

	var t map[string]string
	if err := json.Unmarshal(data, &t); err != nil {
		return nil
	}

	translationsMu.Lock()
	translations[lang] = t
	translationsMu.Unlock()

	return t
}

func (h *Handler) detectLanguage(c *gin.Context) string {
	lang := c.Query("lang")
	if lang != "" {
		return lang
	}

	cookie, err := c.Request.Cookie("lang")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	accept := c.Request.Header.Get("Accept-Language")
	if accept != "" {
		langs := strings.SplitSeq(accept, ",")
		for l := range langs {
			code := strings.Split(strings.TrimSpace(l), ";")[0]
			if strings.HasPrefix(code, "nl") {
				return "nl"
			}
		}
	}

	return "en"
}

func (h *Handler) GetTranslations(c *gin.Context) {
	lang := h.detectLanguage(c)
	t := h.loadTranslations(lang)
	if t == nil {
		t = h.loadTranslations("en")
	}
	t["_lang"] = lang
	c.JSON(http.StatusOK, t)
}

func (h *Handler) SetLanguage(c *gin.Context) {
	var req struct {
		Lang string `json:"lang"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Lang == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "lang required"})
		return
	}

	validLangs := map[string]bool{"en": true, "nl": true}
	if !validLangs[req.Lang] {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unsupported language"})
		return
	}

	c.SetCookie("lang", req.Lang, 365*24*3600, "/", "", h.S.SecureCookies, true)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "lang": req.Lang})
}

func (h *Handler) I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := h.detectLanguage(c)
		c.Set("lang", lang)
		c.Next()
	}
}
