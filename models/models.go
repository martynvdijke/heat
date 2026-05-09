package models

type Racer struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profile_picture"`
	CarColor       string `json:"car_color"`
	CarName        string `json:"car_name"`
	Points         int    `json:"points"`
	Rank           int    `json:"rank"`
	Position       int    `json:"position"`
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
