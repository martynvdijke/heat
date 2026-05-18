package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
	"heat/ws"
)

var racersTableTmpl = template.Must(template.New("racers_table").Parse(`<tbody id="racer-list">
{{range .}}
<tr>
    <td class="ps-4 fw-bold">#{{.Rank}}</td>
    <td>
        <div class="d-flex align-items-center">
            <img src="{{.ProfilePicture}}" class="rounded-circle me-3" width="32" height="32" style="object-fit:cover" onerror="this.src='/static/images/helmet.svg'">
            <div><div class="fw-bold">{{.Name}}</div></div>
        </div>
    </td>
    <td><span class="color-dot" style="background:{{.CarColor}}"></span> {{.CarName}}</td>
    <td><span class="badge bg-dark">{{.Points}} pts</span></td>
    <td class="small text-muted">{{.Position}}</td>
    <td>
        {{if .ShareLink}}<button class="btn btn-sm btn-outline-success" onclick="copyShareLink({{.ID}})" title="Copy share link"><i class="fa-solid fa-link"></i></button>{{end}}
        {{if not .ShareLink}}<button class="btn btn-sm btn-outline-secondary" hx-post="/api/html/racers/{{.ID}}/share" hx-target="#racer-list" hx-swap="outerHTML" title="Generate share link"><i class="fa-solid fa-share-nodes"></i></button>{{end}}
    </td>
    <td class="text-end pe-4">
        <button class="btn btn-sm btn-outline-primary" hx-get="/api/html/racers/{{.ID}}/edit" hx-target="#racerModal .modal-body" hx-swap="innerHTML" data-bs-toggle="modal" data-bs-target="#racerModal"><i class="fa-solid fa-pen"></i></button>
        <button class="btn btn-sm btn-outline-danger" hx-delete="/api/html/racers/{{.ID}}" hx-target="#racer-list" hx-swap="outerHTML" hx-confirm="Delete this racer?"><i class="fa-solid fa-trash"></i></button>
    </td>
</tr>
{{end}}
</tbody>`))

var racerEditFormTmpl = template.Must(template.New("racer_form").Parse(`<form id="racer-form" hx-post="/api/html/racers" hx-target="#racer-list" hx-swap="outerHTML" hx-trigger="submit">
    <input type="hidden" name="id" value="{{if .}}{{.ID}}{{end}}">
    <div class="mb-3 text-center">
        <img src="{{if .}}{{.ProfilePicture}}{{end}}" class="rounded-circle mb-2" width="80" height="80" style="object-fit:cover;border:3px solid var(--heat-red)" onerror="this.style.display='none'">
    </div>
    <div class="mb-3">
        <label class="form-label small fw-bold">Name</label>
        <input type="text" class="form-control" name="name" value="{{if .}}{{.Name}}{{end}}" required>
    </div>
    <div class="mb-3">
        <label class="form-label small fw-bold">Profile Picture URL</label>
        <input type="text" class="form-control" name="profile_picture" value="{{if .}}{{.ProfilePicture}}{{end}}">
    </div>
    <div class="row g-3 mb-3">
        <div class="col-6">
            <label class="form-label small fw-bold">Car Name</label>
            <input type="text" class="form-control" name="car_name" value="{{if .}}{{.CarName}}{{end}}" required>
        </div>
        <div class="col-6">
            <label class="form-label small fw-bold">Car Color</label>
            <input type="color" class="form-control form-control-color" name="car_color" value="{{if .}}{{.CarColor}}{{else}}#d40000{{end}}" style="width:100%;height:38px;padding:2px">
        </div>
    </div>
    <div class="row g-3 mb-3">
        <div class="col-4">
            <label class="form-label small fw-bold">Points</label>
            <input type="number" class="form-control" name="points" value="{{if .}}{{.Points}}{{else}}0{{end}}" required>
        </div>
        <div class="col-4">
            <label class="form-label small fw-bold">Rank</label>
            <input type="number" class="form-control" name="rank" value="{{if .}}{{.Rank}}{{else}}0{{end}}" required>
        </div>
        <div class="col-4">
            <label class="form-label small fw-bold">Board Gap</label>
            <input type="number" class="form-control" name="position" value="{{if .}}{{.Position}}{{else}}0{{end}}" required>
        </div>
    </div>
    <button type="submit" class="btn btn-primary w-100">Save Racer</button>
</form>`))

type racerRow struct {
	models.Racer
	ShareLink string
}

func HtmxRacersTable(c *gin.Context) {
	shares := make(map[int]string)
	srows, err := app.DB.Query("SELECT racer_id, token FROM driver_shares")
	if err == nil {
		for srows.Next() {
			var rid int
			var tok string
			if srows.Scan(&rid, &tok) == nil {
				shares[rid] = tok
			}
		}
		srows.Close()
	}

	rows, err := app.DB.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank, position, COALESCE(team_id, 0) FROM racers ORDER BY rank ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		origin = c.Request.Header.Get("Referer")
	}

	racers := make([]racerRow, 0)
	for rows.Next() {
		var r models.Racer
		if err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position, &r.TeamID); err != nil {
			continue
		}
		row := racerRow{Racer: r}
		if token, ok := shares[r.ID]; ok {
			row.ShareLink = origin + "/driver.html?token=" + token
		}
		racers = append(racers, row)
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	racersTableTmpl.Execute(c.Writer, racers)
}

func HtmxRacersEditForm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if id == 0 {
		racerEditFormTmpl.Execute(c.Writer, nil)
		return
	}

	var r models.Racer
	err = app.DB.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points, rank, position FROM racers WHERE id=?", id).
		Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position)
	if err != nil {
		racerEditFormTmpl.Execute(c.Writer, nil)
		return
	}

	racerEditFormTmpl.Execute(c.Writer, &r)
}

func HtmxRacersSave(c *gin.Context) {
	idStr := strings.TrimSpace(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))
	profilePicture := strings.TrimSpace(c.PostForm("profile_picture"))
	carColor := strings.TrimSpace(c.PostForm("car_color"))
	carName := strings.TrimSpace(c.PostForm("car_name"))
	pointsStr := strings.TrimSpace(c.PostForm("points"))
	rankStr := strings.TrimSpace(c.PostForm("rank"))
	positionStr := strings.TrimSpace(c.PostForm("position"))

	if name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	points, _ := strconv.Atoi(pointsStr)
	rank, _ := strconv.Atoi(rankStr)
	position, _ := strconv.Atoi(positionStr)

	if profilePicture == "" {
		profilePicture = "/static/images/helmet.svg"
	}

	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		_, err := app.DB.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank, position) VALUES (?, ?, ?, ?, ?, ?, ?)",
			name, profilePicture, carColor, carName, points, rank, position)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := app.DB.Exec("UPDATE racers SET name=?, profile_picture=?, car_color=?, car_name=?, points=?, rank=?, position=? WHERE id=?",
			name, profilePicture, carColor, carName, points, rank, position, id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	ws.BroadcastRacers()
	c.Header("HX-Trigger", `{"closeRacerModal":true}`)
	HtmxRacersTable(c)
}

func HtmxRacersDelete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	app.DB.Exec("DELETE FROM racers WHERE id=?", id)
	ws.BroadcastRacers()
	HtmxRacersTable(c)
}

func HtmxRacersGenerateShare(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	_, err = app.DB.Exec("DELETE FROM driver_shares WHERE racer_id=?", id)
	if err == nil {
		var token string
		app.DB.QueryRow("SELECT hex(randomblob(16))").Scan(&token)
		app.DB.Exec("INSERT INTO driver_shares (racer_id, token) VALUES (?, ?)", id, token)
	}
	HtmxRacersTable(c)
}

// Tracks

var tracksTableTmpl = template.Must(template.New("tracks_table").Parse(`<tbody id="track-list">
{{range .}}
<tr>
    <td class="ps-4"><code>{{.ID}}</code></td>
    <td class="fw-bold">{{.Name}}</td>
    <td>{{.Country}}</td>
    <td>{{.Length}} km</td>
    <td>
        {{if .UseMapImage}}<span class="badge bg-info text-dark">Image Map</span>{{end}}
        {{if not .UseMapImage}}<span class="badge bg-secondary">GeoJSON</span>{{end}}
        {{if .RefreshGeoJSON}}<i class="fa-solid fa-sync fa-spin ms-1 text-success small" title="Live Refresh On"></i>{{end}}
    </td>
    <td class="text-end pe-4">
        <button class="btn btn-sm btn-outline-primary" hx-get="/api/html/tracks/{{.ID}}/edit" hx-target="#trackModal .modal-body" hx-swap="innerHTML" data-bs-toggle="modal" data-bs-target="#trackModal"><i class="fa-solid fa-pen"></i></button>
        <button class="btn btn-sm btn-outline-danger" hx-delete="/api/html/tracks/{{.ID}}" hx-target="#track-list" hx-swap="outerHTML" hx-confirm="Delete this track?"><i class="fa-solid fa-trash"></i></button>
    </td>
</tr>
{{end}}
</tbody>`))

var trackEditFormTmpl = template.Must(template.New("track_form").Parse(`<form id="track-form" hx-post="/api/html/tracks" hx-target="#track-list" hx-swap="outerHTML" hx-trigger="submit">
    <input type="hidden" name="id" value="{{if .}}{{.ID}}{{end}}">
    <div class="mb-3">
        <label class="form-label small fw-bold">Track ID</label>
        <input type="text" class="form-control" name="id_visible" value="{{if .}}{{.ID}}{{end}}" required>
    </div>
    <div class="mb-3">
        <label class="form-label small fw-bold">Track Name</label>
        <input type="text" class="form-control" name="name" value="{{if .}}{{.Name}}{{end}}" required>
    </div>
    <div class="mb-3">
        <label class="form-label small fw-bold">Country</label>
        <input type="text" class="form-control" name="country" value="{{if .}}{{.Country}}{{end}}" required>
    </div>
    <div class="row g-3 mb-3">
        <div class="col-6">
            <label class="form-label small fw-bold">Length (km)</label>
            <input type="number" class="form-control" name="length_km" value="{{if .}}{{.Length}}{{end}}" required>
        </div>
        <div class="col-6">
            <label class="form-label small fw-bold">Lap Record</label>
            <input type="text" class="form-control" name="lap_record" value="{{if .}}{{.LapRecord}}{{end}}">
        </div>
    </div>
    <button type="submit" class="btn btn-primary w-100">Save Track</button>
</form>`))

func HtmxTracksTable(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, country, length_km, lap_record, COALESCE(use_map_image, 0), COALESCE(map_image_url, ''), COALESCE(refresh_geojson, 1) FROM tracks ORDER BY name")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	tracks := make([]models.Track, 0)
	for rows.Next() {
		var t models.Track
		var useMapImage, refreshGeoJSON int
		if err := rows.Scan(&t.ID, &t.Name, &t.Country, &t.Length, &t.LapRecord, &useMapImage, &t.MapImageURL, &refreshGeoJSON); err != nil {
			continue
		}
		t.UseMapImage = useMapImage == 1
		t.RefreshGeoJSON = refreshGeoJSON == 1
		tracks = append(tracks, t)
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	tracksTableTmpl.Execute(c.Writer, tracks)
}

func HtmxTracksEditForm(c *gin.Context) {
	id := c.Param("id")

	var t models.Track
	var useMapImage, refreshGeoJSON int
	err := app.DB.QueryRow("SELECT id, name, country, length_km, lap_record, COALESCE(use_map_image, 0), COALESCE(map_image_url, ''), COALESCE(refresh_geojson, 1) FROM tracks WHERE id=?", id).
		Scan(&t.ID, &t.Name, &t.Country, &t.Length, &t.LapRecord, &useMapImage, &t.MapImageURL, &refreshGeoJSON)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "track not found"})
		return
	}
	t.UseMapImage = useMapImage == 1
	t.RefreshGeoJSON = refreshGeoJSON == 1

	c.Header("Content-Type", "text/html; charset=utf-8")
	trackEditFormTmpl.Execute(c.Writer, &t)
}

func HtmxTracksSave(c *gin.Context) {
	id := strings.TrimSpace(c.PostForm("id"))
	idVisible := strings.TrimSpace(c.PostForm("id_visible"))
	name := strings.TrimSpace(c.PostForm("name"))
	country := strings.TrimSpace(c.PostForm("country"))
	lengthStr := strings.TrimSpace(c.PostForm("length_km"))
	lapRecord := strings.TrimSpace(c.PostForm("lap_record"))

	if name == "" || country == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name and country required"})
		return
	}

	length, _ := strconv.Atoi(lengthStr)
	if length <= 0 {
		length = 5
	}

	if id == "" {
		id = idVisible
	}

	_, err := app.DB.Exec(`INSERT OR REPLACE INTO tracks (id, name, country, geojson, length_km, lap_record, use_map_image, map_image_url, refresh_geojson) VALUES (?, ?, ?, ?, ?, ?, 0, '', 1)`,
		id, name, country, id, length, lapRecord)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Trigger", `{"closeTrackModal":true}`)
	HtmxTracksTable(c)
}

func HtmxTracksDelete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	app.DB.Exec("DELETE FROM tracks WHERE id = ?", id)
	HtmxTracksTable(c)
}

// Quotes

var quotesTableTmpl = template.Must(template.New("quotes_table").Parse(`<tbody id="quote-list">
{{range .}}
<tr>
    <td class="ps-4"><em>"{{.Text}}"</em></td>
    <td class="fw-bold">{{.Author}}</td>
    <td class="text-end pe-4">
        <button class="btn btn-sm btn-outline-primary" hx-get="/api/html/quotes/{{.ID}}/edit" hx-target="#quoteModal .modal-body" hx-swap="innerHTML" data-bs-toggle="modal" data-bs-target="#quoteModal"><i class="fa-solid fa-pen"></i></button>
        <button class="btn btn-sm btn-outline-danger" hx-delete="/api/html/quotes/{{.ID}}" hx-target="#quote-list" hx-swap="outerHTML" hx-confirm="Delete this quote?"><i class="fa-solid fa-trash"></i></button>
    </td>
</tr>
{{end}}
</tbody>`))

var quoteEditFormTmpl = template.Must(template.New("quote_form").Parse(`<form id="quote-form" hx-post="/api/html/quotes" hx-target="#quote-list" hx-swap="outerHTML" hx-trigger="submit">
    <input type="hidden" name="id" value="{{if .}}{{.ID}}{{end}}">
    <div class="mb-3">
        <label class="form-label small fw-bold">Quote Text</label>
        <textarea class="form-control" name="text" rows="3" required>{{if .}}{{.Text}}{{end}}</textarea>
    </div>
    <div class="mb-3">
        <label class="form-label small fw-bold">Author</label>
        <input type="text" class="form-control" name="author" value="{{if .}}{{.Author}}{{end}}">
    </div>
    <button type="submit" class="btn btn-primary w-100">Save Quote</button>
</form>`))

func HtmxQuotesTable(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, text, author FROM quotes ORDER BY id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	quotes := make([]models.Quote, 0)
	for rows.Next() {
		var q models.Quote
		if err := rows.Scan(&q.ID, &q.Text, &q.Author); err != nil {
			continue
		}
		quotes = append(quotes, q)
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	quotesTableTmpl.Execute(c.Writer, quotes)
}

func HtmxQuotesEditForm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var q models.Quote
	err = app.DB.QueryRow("SELECT id, text, author FROM quotes WHERE id=?", id).Scan(&q.ID, &q.Text, &q.Author)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "quote not found"})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	quoteEditFormTmpl.Execute(c.Writer, &q)
}

func HtmxQuotesSave(c *gin.Context) {
	idStr := strings.TrimSpace(c.PostForm("id"))
	text := strings.TrimSpace(c.PostForm("text"))
	author := strings.TrimSpace(c.PostForm("author"))

	if text == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "text required"})
		return
	}
	if author == "" {
		author = "Commentator"
	}

	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		_, err := app.DB.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", text, author)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := app.DB.Exec("UPDATE quotes SET text=?, author=? WHERE id=?", text, author, id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.Header("HX-Trigger", `{"closeQuoteModal":true}`)
	HtmxQuotesTable(c)
}

func HtmxQuotesDelete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	app.DB.Exec("DELETE FROM quotes WHERE id=?", id)
	HtmxQuotesTable(c)
}

// Seasons

var seasonsTableTmpl = template.Must(template.New("seasons_table").Parse(`<tbody id="seasons-list">
{{range .}}
<tr>
    <td class="ps-4 fw-bold">{{.Name}}</td>
    <td>{{.StartDate}}</td>
    <td>{{if .EndDate}}{{.EndDate}}{{else}}-{{end}}</td>
    <td><span class="badge {{if eq .Status "active"}}bg-success{{else}}bg-secondary{{end}}">{{.Status}}</span></td>
    <td class="text-end pe-4">
        {{if eq .Status "active"}}
        <button class="btn btn-sm btn-outline-warning" hx-post="/api/html/seasons/{{.ID}}/archive" hx-target="#seasons-list" hx-swap="outerHTML" hx-confirm="Archive this season?"><i class="fa-solid fa-box-archive"></i></button>
        {{end}}
        <button class="btn btn-sm btn-outline-danger" hx-delete="/api/html/seasons/{{.ID}}" hx-target="#seasons-list" hx-swap="outerHTML" hx-confirm="Delete this season? All rounds will also be deleted."><i class="fa-solid fa-trash"></i></button>
    </td>
</tr>
{{end}}
</tbody>`))

func HtmxSeasonsTable(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, start_date, COALESCE(end_date, ''), status FROM seasons ORDER BY id DESC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type seasonRow struct {
		models.Season
	}

	seasons := make([]seasonRow, 0)
	for rows.Next() {
		var s models.Season
		if err := rows.Scan(&s.ID, &s.Name, &s.StartDate, &s.EndDate, &s.Status); err != nil {
			continue
		}
		seasons = append(seasons, seasonRow{s})
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	seasonsTableTmpl.Execute(c.Writer, seasons)
}

func HtmxSeasonsCreate(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	_, err := app.DB.Exec("INSERT INTO seasons (name, start_date, status) VALUES (?, date('now'), 'active')", name)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	HtmxSeasonsTable(c)
}

func HtmxSeasonsArchive(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	_, err := app.DB.Exec("UPDATE seasons SET status = 'archived', end_date = date('now') WHERE id = ?", idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	HtmxSeasonsTable(c)
}

func HtmxSeasonsDelete(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	app.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id IN (SELECT id FROM round_snapshots WHERE season_id = ?)", idStr)
	app.DB.Exec("DELETE FROM round_snapshots WHERE season_id = ?", idStr)
	app.DB.Exec("DELETE FROM seasons WHERE id = ?", idStr)
	HtmxSeasonsTable(c)
}

// Teams

var teamsTableTmpl = template.Must(template.New("teams_table").Parse(`<tbody id="team-list">
{{range .}}
<tr>
    <td class="ps-4"><span class="color-dot" style="background:{{.Color}}"></span></td>
    <td class="fw-bold">{{.Name}}</td>
    <td class="text-end pe-4">
        <button class="btn btn-sm btn-outline-primary" hx-get="/api/html/teams/{{.ID}}/edit" hx-target="#teamModal .modal-body" hx-swap="innerHTML" data-bs-toggle="modal" data-bs-target="#teamModal"><i class="fa-solid fa-pen"></i></button>
        <button class="btn btn-sm btn-outline-danger" hx-delete="/api/html/teams/{{.ID}}" hx-target="#team-list" hx-swap="outerHTML" hx-confirm="Delete this team?"><i class="fa-solid fa-trash"></i></button>
    </td>
</tr>
{{end}}
</tbody>`))

var teamEditFormTmpl = template.Must(template.New("team_form").Parse(`<form id="team-form" hx-post="/api/html/teams" hx-target="#team-list" hx-swap="outerHTML" hx-trigger="submit">
    <input type="hidden" name="id" value="{{if .}}{{.ID}}{{end}}">
    <div class="mb-3">
        <label class="form-label small fw-bold">Team Name</label>
        <input type="text" class="form-control" name="name" value="{{if .}}{{.Name}}{{end}}" required>
    </div>
    <div class="mb-3">
        <label class="form-label small fw-bold">Team Color</label>
        <input type="color" class="form-control form-control-color" name="color" value="{{if .}}{{.Color}}{{else}}#d40000{{end}}" style="width:100%;height:38px;padding:2px">
    </div>
    <button type="submit" class="btn btn-primary w-100">Save Team</button>
</form>`))

func HtmxTeamsTable(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, color FROM teams ORDER BY name")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	teams := make([]models.Team, 0)
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Color); err != nil {
			continue
		}
		teams = append(teams, t)
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	teamsTableTmpl.Execute(c.Writer, teams)
}

func HtmxTeamsEditForm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var t models.Team
	err = app.DB.QueryRow("SELECT id, name, color FROM teams WHERE id=?", id).Scan(&t.ID, &t.Name, &t.Color)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	teamEditFormTmpl.Execute(c.Writer, &t)
}

func HtmxTeamsSave(c *gin.Context) {
	idStr := strings.TrimSpace(c.PostForm("id"))
	name := strings.TrimSpace(c.PostForm("name"))
	color := strings.TrimSpace(c.PostForm("color"))

	if name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	if color == "" {
		color = "#d40000"
	}

	id, _ := strconv.Atoi(idStr)
	if id == 0 {
		_, err := app.DB.Exec("INSERT INTO teams (name, color) VALUES (?, ?)", name, color)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := app.DB.Exec("UPDATE teams SET name=?, color=? WHERE id=?", name, color, id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.Header("HX-Trigger", `{"closeTeamModal":true}`)
	HtmxTeamsTable(c)
}

func HtmxTeamsDelete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	app.DB.Exec("UPDATE racers SET team_id = 0 WHERE team_id = ?", id)
	app.DB.Exec("DELETE FROM teams WHERE id = ?", id)
	HtmxTeamsTable(c)
}

var seasonNewFormTmpl = template.Must(template.New("season_form").Parse(`<form id="season-form" hx-post="/api/html/seasons" hx-target="#seasons-list" hx-swap="outerHTML" hx-trigger="submit">
    <div class="mb-3">
        <label class="form-label small fw-bold">Season Name</label>
        <input type="text" class="form-control" name="name" placeholder="e.g. 2024 Championship" required>
    </div>
    <button type="submit" class="btn btn-primary w-100">Create Season</button>
</form>`))

func HtmxSeasonsNewForm(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	seasonNewFormTmpl.Execute(c.Writer, nil)
}
