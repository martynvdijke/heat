package middleware

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

type umamiResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *umamiResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func UmamiMiddleware(s *app.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		w := &umamiResponseWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Next()

		ct := w.Header().Get("Content-Type")
		if strings.Contains(ct, "text/html") && w.body.Len() > 0 {
			html := w.body.String()
			var umami models.UmamiSettings
			err := s.DB.QueryRow("SELECT COALESCE(url, ''), COALESCE(website_id, ''), COALESCE(enabled, 0) FROM umami_settings WHERE id = 1").
				Scan(&umami.URL, &umami.WebsiteID, &umami.Enabled)
			if err == nil && umami.Enabled && umami.URL != "" && umami.WebsiteID != "" {
				script := fmt.Sprintf(`<script defer src="%s/script.js" data-website-id="%s"></script>`, umami.URL, umami.WebsiteID)
				html = strings.Replace(html, "</head>", script+"\n</head>", 1)
			}
			w.ResponseWriter.WriteHeader(w.Status())
			w.ResponseWriter.Write([]byte(html))
		} else {
			w.ResponseWriter.WriteHeader(w.Status())
			w.ResponseWriter.Write(w.body.Bytes())
		}
	}
}
