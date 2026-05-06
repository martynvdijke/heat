package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

// @Summary Get all quotes
// @Description Get the list of all quotes
// @Tags Quotes
// @Produce json
// @Success 200 {array} models.Quote
// @Router /api/quotes [get]
func GetQuotes(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, text, author, created_at FROM quotes ORDER BY id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	quotes := make([]models.Quote, 0)
	for rows.Next() {
		var q models.Quote
		if err := rows.Scan(&q.ID, &q.Text, &q.Author, &q.CreatedAt); err != nil {
			continue
		}
		quotes = append(quotes, q)
	}
	c.JSON(http.StatusOK, quotes)
}

// @Summary Get a random quote
// @Description Get a random racing quote
// @Tags Quotes
// @Produce json
// @Success 200 {object} models.Quote
// @Router /api/quote/random [get]
func GetRandomQuote(c *gin.Context) {
	var q models.Quote
	err := app.DB.QueryRow("SELECT id, text, author, created_at FROM quotes ORDER BY RANDOM() LIMIT 1").Scan(&q.ID, &q.Text, &q.Author, &q.CreatedAt)
	if err != nil {
		q = models.Quote{Text: "The engines roar as these legends battle for glory!", Author: "Commentator"}
	}
	c.JSON(http.StatusOK, q)
}

// @Summary Create a quote
// @Description Create a new quote
// @Tags Quotes
// @Accept json
// @Produce json
// @Param quote body models.Quote true "Quote data"
// @Success 201 {object} models.Quote
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/quotes [post]
func HandleQuotes(c *gin.Context) {
	switch c.Request.Method {
	case "GET":
		GetQuotes(c)
	case "POST":
		var q models.Quote
		if err := c.ShouldBindJSON(&q); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if q.Text == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote text is required"})
			return
		}
		if q.Author == "" {
			q.Author = "Commentator"
		}
		result, err := app.DB.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", q.Text, q.Author)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := result.LastInsertId()
		q.ID = int(id)
		c.JSON(http.StatusCreated, q)
	case "PUT":
		var q models.Quote
		if err := c.ShouldBindJSON(&q); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if q.ID == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote ID is required"})
			return
		}
		_, err := app.DB.Exec("UPDATE quotes SET text = ?, author = ? WHERE id = ?", q.Text, q.Author, q.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, q)
	case "DELETE":
		id := c.Query("id")
		if id == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote ID is required"})
			return
		}
		_, err := app.DB.Exec("DELETE FROM quotes WHERE id = ?", id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}
