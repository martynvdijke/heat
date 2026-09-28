package telegram

// Telegram Bot API wire types (only the fields the bot needs).

type tgResult struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
}

type tgUpdates struct {
	Ok          bool       `json:"ok"`
	Result      []tgUpdate `json:"result"`
	Description string     `json:"description"`
}

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	MessageID int64   `json:"message_id"`
	From      *tgUser `json:"from"`
	Chat      *tgChat `json:"chat"`
	Text      string  `json:"text"`
}

type tgUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type tgChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// Types mirroring the app's public /api/telegram/summary payload.

type apiRaceResult struct {
	RacerName string `json:"racer_name"`
	Team      string `json:"team"`
	Position  int    `json:"position"`
	Points    int    `json:"points"`
}

type apiRace struct {
	Name      string          `json:"name"`
	RaceDate  string          `json:"race_date"`
	Round     int             `json:"round"`
	Country   string          `json:"country"`
	Track     string          `json:"track"`
	TotalLaps int             `json:"total_laps"`
	Results   []apiRaceResult `json:"results"`
}

type apiStanding struct {
	RacerName string `json:"racer_name"`
	TeamName  string `json:"team_name"`
	Races     int    `json:"races"`
	Wins      int    `json:"wins"`
	Points    int    `json:"points"`
}

type apiNextRace struct {
	RaceDate      string `json:"race_date"`
	Track         string `json:"track"`
	Country       string `json:"country"`
	TrackID       string `json:"track_id"`
	TotalLaps     int    `json:"total_laps"`
	DaysRemaining int    `json:"days_remaining"`
}

type apiSeason struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type apiSummary struct {
	LatestRace *apiRace      `json:"latest_race"`
	Standings  []apiStanding `json:"standings"`
	NextRace   *apiNextRace  `json:"next_race"`
	Season     *apiSeason    `json:"season"`
}

// Race archive payload from /api/race-history?id=.

type apiHistoryResult struct {
	RacerName  string `json:"racer_name"`
	Position   int    `json:"position"`
	Points     int    `json:"points"`
	FastestLap bool   `json:"fastest_lap"`
}

type apiHistory struct {
	ID        int                `json:"id"`
	Name      string             `json:"name"`
	RaceDate  string             `json:"race_date"`
	Country   string             `json:"country"`
	Track     string             `json:"track"`
	TotalLaps int                `json:"total_laps"`
	Results   []apiHistoryResult `json:"results"`
}

type apiRacerStats struct {
	RacerID int `json:"racer_id"`
	Races   int `json:"races"`
	Wins    int `json:"wins"`
	Gold    int `json:"gold"`
	Points  int `json:"points"`
}

type apiQuote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

type apiRacer struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
