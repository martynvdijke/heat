package handlers

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultAIProviderVersion is used in the outbound User-Agent when the app
// version is not known.
const defaultAIProviderVersion = "0.0.0-dev"

// aiProviderUserAgent identifies this app to OpenAI-compatible AI providers.
// Matching the opencode/villum convention keeps requests attributable without
// pretending to be another client.
func aiProviderUserAgent(version string) string {
	if v := strings.TrimSpace(version); v != "" {
		version = v
	} else {
		version = defaultAIProviderVersion
	}
	return "heat/" + version
}

// resolveSessionID returns the inbound session id when one was supplied,
// otherwise it generates a new one. The id is sent as x-opencode-session to
// group AI provider requests that belong to the same conversation.
func resolveSessionID(inbound string) string {
	if s := strings.TrimSpace(inbound); s != "" {
		return s
	}
	if id := generateSessionID(); id != "" {
		return id
	}
	return hex.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
}

// setAIProviderHeaders tags an outbound provider request with the client
// User-Agent and a stable per-conversation x-opencode-session id.
func setAIProviderHeaders(req *http.Request, sessionID, version string) {
	req.Header.Set("User-Agent", aiProviderUserAgent(version))
	req.Header.Set("x-opencode-session", sessionID)
}

// aiRequestURL joins an OpenAI-compatible base URL with a path such as
// "/chat/completions", tolerating a missing or trailing slash on base.
func aiRequestURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	return base + "/" + strings.TrimLeft(path, "/")
}
