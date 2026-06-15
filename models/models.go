package models

import "encoding/json"

type Racer struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profile_picture"`
	CarColor       string `json:"car_color"`
	CarName        string `json:"car_name"`
	Points         int    `json:"points"`
	Rank           int    `json:"rank"`
	Position       int    `json:"position"`
	TeamID         int    `json:"team_id,omitempty"`
}

type Team struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"created_at,omitempty"`
}

type RaceInfo struct {
	ID      int    `json:"id"`
	Country string `json:"country"`
	Track   string `json:"track"`
	Laps    int    `json:"laps"`
	TrackID string `json:"track_id"`
}

type Track struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Country        string `json:"country"`
	GeoJSON        string `json:"geojson"`
	Length         int    `json:"length_km"`
	LapRecord      string `json:"lap_record"`
	UseMapImage    bool   `json:"use_map_image"`
	MapImageURL    string `json:"map_image_url"`
	RefreshGeoJSON bool   `json:"refresh_geojson"`
}

type RaceResult struct {
	ID         int    `json:"id"`
	RaceID     int    `json:"race_id"`
	RacerID    int    `json:"racer_id"`
	RacerName  string `json:"racer_name"`
	Position   int    `json:"position"`
	Points     int    `json:"points"`
	FastestLap bool   `json:"fastest_lap"`
	Finished   bool   `json:"finished"`
}

type RacerStats struct {
	ID          int `json:"id"`
	RacerID     int `json:"racer_id"`
	Races       int `json:"races"`
	Wins        int `json:"wins"`
	Gold        int `json:"gold"`
	Silver      int `json:"silver"`
	Bronze      int `json:"bronze"`
	FastestLaps int `json:"fastest_laps"`
	Points      int `json:"points"`
	DNF         int `json:"dnf"`
	DNS         int `json:"dns"`
}

type Quote struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

type RaceHistory struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Date      string       `json:"race_date"`
	Country   string       `json:"country"`
	Track     string       `json:"track"`
	TrackID   string       `json:"track_id"`
	TotalLaps int          `json:"total_laps"`
	RaceType  string       `json:"race_type,omitempty"`
	Results   []RaceResult `json:"results,omitempty"`
}

type UmamiSettings struct {
	ID        int    `json:"id"`
	URL       string `json:"url"`
	WebsiteID string `json:"website_id"`
	Enabled   bool   `json:"enabled"`
}

type NotificationSettings struct {
	ID              int    `json:"id"`
	GotiFyURL       string `json:"gotify_url"`
	GotiFyToken     string `json:"gotify_token"`
	NotifyWinner    bool   `json:"notify_winner"`
	NotifyRaceStart bool   `json:"notify_race_start"`
	NotifyPodium    bool   `json:"notify_podium"`
}

type AdminUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type OTelSettings struct {
	ID             int    `json:"id"`
	Endpoint       string `json:"endpoint"`
	TracesEnabled  bool   `json:"traces_enabled"`
	MetricsEnabled bool   `json:"metrics_enabled"`
	LogsEnabled    bool   `json:"logs_enabled"`
}

type AISettings struct {
	ID              int    `json:"id"`
	TrackExtractURL string `json:"track_extract_url"`
	APIKey          string `json:"api_key"`
	Enabled         bool   `json:"enabled"`
}

type EmailSettings struct {
	ID       int    `json:"id"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	FromAddr string `json:"from_addr"`
	Enabled  bool   `json:"enabled"`
}

type RacerEmail struct {
	ID      int    `json:"id"`
	RacerID int    `json:"racer_id"`
	Email   string `json:"email"`
}

type Upload struct {
	ID           int    `json:"id"`
	Hash         string `json:"hash"`
	Ext          string `json:"ext"`
	URL          string `json:"url"`
	ResizedURL   string `json:"resized_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	CreatedAt    string `json:"created_at"`
}

type TrackStats struct {
	TrackID    string `json:"track_id"`
	TrackName  string `json:"track_name"`
	Country    string `json:"country"`
	RacesCount int    `json:"races_count"`
	Winner     string `json:"winner"`
	FastestLap string `json:"fastest_lap"`
}

type HeadToHead struct {
	Racer1     string  `json:"racer1"`
	Racer2     string  `json:"racer2"`
	Races      int     `json:"races"`
	Racer1Wins int     `json:"racer1_wins"`
	Racer2Wins int     `json:"racer2_wins"`
	Racer1Avg  float64 `json:"racer1_avg_position"`
	Racer2Avg  float64 `json:"racer2_avg_position"`
}

type PointsProgression struct {
	RaceID   int    `json:"race_id"`
	RaceName string `json:"race_name"`
	RaceDate string `json:"race_date"`
	Points   int    `json:"points"`
}

type StreakInfo struct {
	RacerName    string `json:"racer_name"`
	StreakType   string `json:"streak_type"`
	CurrentValue int    `json:"current_value"`
	BestValue    int    `json:"best_value"`
	BestStart    string `json:"best_start"`
	BestEnd      string `json:"best_end"`
}

type ELORating struct {
	RacerID   int     `json:"racer_id"`
	RacerName string  `json:"racer_name"`
	Rating    float64 `json:"rating"`
	Races     int     `json:"races"`
}

type EInkSettings struct {
	ID      int  `json:"id"`
	Enabled bool `json:"enabled"`
}

type BackupSettings struct {
	ID             int  `json:"id"`
	Enabled        bool `json:"enabled"`
	IntervalHrs    int  `json:"interval_hrs"`
	RetentionCount int  `json:"retention_count"`
}

type Season struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type RoundSnapshot struct {
	ID        int                  `json:"id"`
	SeasonID  int                  `json:"season_id"`
	RaceName  string               `json:"race_name"`
	RaceDate  string               `json:"race_date"`
	Round     int                  `json:"round"`
	CreatedAt string               `json:"created_at"`
	Scores    []RoundSnapshotScore `json:"scores,omitempty"`
}

type RoundSnapshotScore struct {
	ID         int    `json:"id"`
	SnapshotID int    `json:"snapshot_id"`
	RacerID    int    `json:"racer_id"`
	RacerName  string `json:"racer_name"`
	Points     int    `json:"points"`
	Position   int    `json:"position"`
}

type FlagCommand struct {
	Type      string `json:"type"`
	Flag      string `json:"flag"`
	State     string `json:"state,omitempty"`
	RacerID   int    `json:"racer_id,omitempty"`
	RacerName string `json:"racer_name,omitempty"`
}

// Game Mechanics

type HeatCard struct {
	ID       int    `json:"id"`
	RacerID  int    `json:"racer_id"`
	Location string `json:"location"`  // engine, hand, discard, deck
	CardType string `json:"card_type"` // heat, stress, boost
	LapAdded int    `json:"lap_added"`
}

type GearShift struct {
	ID      int `json:"id"`
	RacerID int `json:"racer_id"`
	RaceID  int `json:"race_id"`
	Lap     int `json:"lap"`
	Gear    int `json:"gear"`
	Stress  int `json:"stress"` // stress accumulated this shift
}

type UpgradeCard struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CardType    string `json:"card_type"` // upgrade, legendary, sponsorship
	Cost        int    `json:"cost"`
	Effects     string `json:"effects"` // JSON string of effects
}

type PlayerUpgrade struct {
	ID          int          `json:"id"`
	RacerID     int          `json:"racer_id"`
	UpgradeID   int          `json:"upgrade_id"`
	SeasonID    int          `json:"season_id"`
	Equipped    bool         `json:"equipped"`
	RoundBought int          `json:"round_bought"`
	Upgrade     *UpgradeCard `json:"upgrade,omitempty"`
}

type LegendAbility struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AbilityType string `json:"ability_type"`
	RacerName   string `json:"racer_name"`
}

type RacerLegendAbility struct {
	ID        int            `json:"id"`
	RacerID   int            `json:"racer_id"`
	AbilityID int            `json:"ability_id"`
	Active    bool           `json:"active"`
	Ability   *LegendAbility `json:"ability,omitempty"`
}

// Multi-User

type PlayerSession struct {
	ID         int    `json:"id"`
	RacerID    int    `json:"racer_id"`
	Token      string `json:"token"`
	DeviceName string `json:"device_name"`
	LastSeen   string `json:"last_seen"`
	CreatedAt  string `json:"created_at"`
}

type SelfServiceAction struct {
	Type      string `json:"type"`
	RacerID   int    `json:"racer_id"`
	Token     string `json:"token"`
	Lap       int    `json:"lap,omitempty"`
	Gear      int    `json:"gear,omitempty"`
	Stress    int    `json:"stress,omitempty"`
	HeatAdded int    `json:"heat_added,omitempty"`
	Position  int    `json:"position,omitempty"`
	TurboUsed bool   `json:"turbo_used,omitempty"`
}

// Race Enhancements

type WeatherCondition struct {
	ID           int     `json:"id"`
	RaceID       int     `json:"race_id"`
	Condition    string  `json:"condition"` // dry, wet, damp, torrential
	LapStart     int     `json:"lap_start"`
	LapEnd       int     `json:"lap_end"`
	GripModifier float64 `json:"grip_modifier"`
}

type TurboLog struct {
	ID        int `json:"id"`
	RacerID   int `json:"racer_id"`
	RaceID    int `json:"race_id"`
	Lap       int `json:"lap"`
	TimesUsed int `json:"times_used"`
}

type LapRecord struct {
	ID            int    `json:"id"`
	RaceID        int    `json:"race_id"`
	RacerID       int    `json:"racer_id"`
	LapNumber     int    `json:"lap_number"`
	Position      int    `json:"position"`
	GearUsed      int    `json:"gear_used"`
	HeatGenerated int    `json:"heat_generated"`
	TurboUsed     bool   `json:"turbo_used"`
	Timestamp     string `json:"timestamp"`
}

type Sector struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	TrackID string `json:"track_id"`
	Order   int    `json:"order"`
}

type RacerSector struct {
	ID        int    `json:"id"`
	RaceID    int    `json:"race_id"`
	RacerID   int    `json:"racer_id"`
	SectorID  int    `json:"sector_id"`
	Lap       int    `json:"lap"`
	EntryTime string `json:"entry_time"`
	ExitTime  string `json:"exit_time"`
}

type RaceEvent struct {
	ID        int    `json:"id"`
	RaceID    int    `json:"race_id"`
	Lap       int    `json:"lap"`
	EventType string `json:"event_type"` // overtake, crash, spin, safety_car, pit_stop
	RacerID   int    `json:"racer_id"`
	RacerID2  int    `json:"racer_id2,omitempty"`
	Note      string `json:"note,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type DriverShare struct {
	ID        int    `json:"id"`
	RacerID   int    `json:"racer_id"`
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
}

type RaceRadioMessage struct {
	ID        int    `json:"id"`
	RaceID    int    `json:"race_id"`
	RacerID   int    `json:"racer_id"`
	RacerName string `json:"racer_name,omitempty"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp,omitempty"`
}

// WebSocket message wrappers

type GameMechanicsUpdate struct {
	Type    string          `json:"type"` // heat_cards, gear_shifts, turbo, upgrades
	RacerID int             `json:"racer_id"`
	Action  string          `json:"action"` // added, removed, shifted, used
	Data    json.RawMessage `json:"data,omitempty"`
}

type LapReplayFrame struct {
	Type      string          `json:"type"` // lap_replay
	RaceID    int             `json:"race_id"`
	Lap       int             `json:"lap"`
	Positions []RacerPosition `json:"positions"`
	Events    []RaceEvent     `json:"events,omitempty"`
}

type RacerPosition struct {
	RacerID    int    `json:"racer_id"`
	RacerName  string `json:"racer_name"`
	Position   int    `json:"position"`
	CarColor   string `json:"car_color"`
	Lap        int    `json:"lap"`
	HeatInHand int    `json:"heat_in_hand,omitempty"`
	Gear       int    `json:"gear,omitempty"`
}

// Sound FX
type SoundCommand struct {
	Type  string          `json:"type"`  // sound
	Sound string          `json:"sound"` // engine, horn, finish, flag, crash
	Data  json.RawMessage `json:"data,omitempty"`
}

type LogEntry struct {
	ID        int64  `json:"id"`
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Module    string `json:"module"`
	Message   string `json:"message"`
	Data      string `json:"data,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type LogSetting struct {
	ID     int    `json:"id"`
	Module string `json:"module"`
	Level  string `json:"level"`
}
