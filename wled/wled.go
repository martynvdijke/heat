package wled

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"heat/app"
	"heat/models"
)

// Client syncs race-control flags to a WLED device over its JSON API.
type Client struct {
	S    *app.Server
	HTTP *http.Client
}

func New(s *app.Server) *Client {
	return &Client{S: s, HTTP: &http.Client{Timeout: 3 * time.Second}}
}

// Run consumes flag broadcasts and mirrors them to WLED until the channel closes.
func (c *Client) Run() {
	for cmd := range c.S.WLEDBroadcast {
		c.Sync(cmd)
	}
}

// Sync loads settings and pushes the flag state to WLED. No-op unless enabled,
// configured, and armed.
func (c *Client) Sync(cmd models.FlagCommand) {
	s, err := LoadSettings(c.S)
	if err != nil {
		c.S.Log.Warnf("wled", "load settings: %v", err)
		return
	}
	if !s.Enabled || s.URL == "" || !c.S.WLEDArmed.Load() {
		return
	}
	payload := buildPayload(cmd, s.Presets)
	if payload == nil {
		return
	}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(s.URL, "/") + "/json/state"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		c.S.Log.Warnf("wled", "build request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		c.S.Log.Warnf("wled", "post %s: %v", url, err)
		return
	}
	resp.Body.Close()
}

// SendTest pushes a test color to the given WLED base URL.
func (c *Client) SendTest(baseURL string) error {
	payload := map[string]any{
		"on":  true,
		"bri": 255,
		"seg": []map[string]any{{"col": [][3]int{{0, 128, 255}}}},
	}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(baseURL, "/") + "/json/state"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// LoadSettings reads the single WLED settings row.
func LoadSettings(s *app.Server) (models.WLEDSettings, error) {
	out := models.WLEDSettings{ID: 1, Presets: map[string]int{}}
	var enabled int
	var presets string
	err := s.DB.QueryRow("SELECT id, COALESCE(url, ''), COALESCE(enabled, 0), COALESCE(presets, '{}') FROM wled_settings WHERE id = 1").
		Scan(&out.ID, &out.URL, &enabled, &presets)
	if err != nil {
		return out, err
	}
	out.Enabled = enabled != 0
	if presets != "" {
		_ = json.Unmarshal([]byte(presets), &out.Presets)
	}
	return out, nil
}

// defaultColors maps a flag to an RGB color used when no preset is configured.
var defaultColors = map[string][3]int{
	"safety":      {255, 140, 0},
	"red":         {255, 0, 0},
	"blue":        {0, 0, 255},
	"yellow":      {255, 255, 0},
	"chequered":   {255, 255, 255},
	"blackwhite":  {255, 255, 255},
	"clear":       {0, 255, 0},
	"green":       {0, 255, 0},
	"startlights": {255, 0, 0},
}

// buildPayload turns a flag command into a WLED JSON state payload. Returns nil
// when the command carries no actionable state.
func buildPayload(cmd models.FlagCommand, presets map[string]int) map[string]any {
	key := cmd.Flag
	switch cmd.State {
	case "", "on", "sequence":
		// keep the flag key
	default:
		key = "clear"
	}
	if key == "" {
		return nil
	}
	if presets != nil {
		if p, ok := presets[key]; ok && p > 0 {
			return map[string]any{"ps": p}
		}
	}
	col, ok := defaultColors[key]
	if !ok {
		return nil
	}
	return map[string]any{
		"on":  true,
		"bri": 255,
		"seg": []map[string]any{{"col": [][3]int{col}}},
	}
}
