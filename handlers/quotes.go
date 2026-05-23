package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/ent"
	"heat/ent/quote"
	"heat/models"
)

// @Summary Get all quotes
// @Description Get the list of all quotes
// @Tags Quotes
// @Produce json
// @Success 200 {array} models.Quote
// @Router /api/quotes [get]
func GetQuotes(c *gin.Context) {
	entQuotes, err := app.Ent.Quote.Query().Order(ent.Asc(quote.FieldID)).All(context.Background())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	quotes := make([]models.Quote, len(entQuotes))
	for i, q := range entQuotes {
		quotes[i] = models.Quote{ID: q.ID, Text: q.Text, Author: q.Author, CreatedAt: q.CreatedAt}
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
	q, err := app.Ent.Quote.Query().Order(ent.Asc(quote.FieldID)).First(context.Background())
	if err != nil {
		q = &ent.Quote{Text: "The engines roar as these legends battle for glory!", Author: "Commentator"}
	}
	c.JSON(http.StatusOK, models.Quote{ID: q.ID, Text: q.Text, Author: q.Author, CreatedAt: q.CreatedAt})
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
		created, err := app.Ent.Quote.Create().SetText(q.Text).SetAuthor(q.Author).Save(context.Background())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		q.ID = created.ID
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
		_, err := app.Ent.Quote.UpdateOneID(q.ID).SetText(q.Text).SetAuthor(q.Author).Save(context.Background())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, q)
	case "DELETE":
		idStr := c.Query("id")
		if idStr == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote ID is required"})
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid quote ID"})
			return
		}
		_, err = app.Ent.Quote.Delete().Where(quote.ID(id)).Exec(context.Background())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}
