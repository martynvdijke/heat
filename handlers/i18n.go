package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"heat/app"
)

var (
	translations   = make(map[string]map[string]string)
	translationsMu sync.RWMutex
)

func loadTranslations(lang string) map[string]string {
	translationsMu.RLock()
	cached, ok := translations[lang]
	translationsMu.RUnlock()
	if ok {
		return cached
	}

	path := filepath.Join(app.BasePath, "static", "locales", lang+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join(app.BasePath, "static", "locales", "en.json")
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

func detectLanguage(c *gin.Context) string {
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
		langs := strings.Split(accept, ",")
		for _, l := range langs {
			code := strings.Split(strings.TrimSpace(l), ";")[0]
			if strings.HasPrefix(code, "nl") {
				return "nl"
			}
		}
	}

	return "en"
}

func GetTranslations(c *gin.Context) {
	lang := detectLanguage(c)
	t := loadTranslations(lang)
	if t == nil {
		t = loadTranslations("en")
	}
	t["_lang"] = lang
	c.JSON(http.StatusOK, t)
}

func SetLanguage(c *gin.Context) {
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

	c.SetCookie("lang", req.Lang, 365*24*3600, "/", "", app.SecureCookies, true)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "lang": req.Lang})
}

func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := detectLanguage(c)
		c.Set("lang", lang)
		c.Next()
	}
}
