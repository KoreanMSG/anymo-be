package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

type Chat struct {
	ID             int       `json:"id"`
	StartWithDoctor bool     `json:"startWithDoctor"`
	Text           string    `json:"text"`
	RiskScore      int       `json:"riskScore"`
	Memo           string    `json:"memo"`
	CreatedAt      time.Time `json:"createdAt"`
}

var db *sql.DB

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://anymodb_user:lQ11owYzyp03O7tRibPL2gonpJYiZ3pB@dpg-cvsd7j95pdvs73bjq0gg-a.singapore-postgres.render.com/anymodb?sslmode=require"
	}
	
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Routes
	r.GET("/chats", getChats)
	r.GET("/chats/:id", getChat)
	r.POST("/chats", createChat)
	r.PUT("/chats/:id", updateChat)
	r.DELETE("/chats/:id", deleteChat)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func getChats(c *gin.Context) {
	rows, err := db.Query("SELECT id, start_with_doctor, text, risk_score, memo, created_at FROM chats ORDER BY created_at DESC")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		err := rows.Scan(&chat.ID, &chat.StartWithDoctor, &chat.Text, &chat.RiskScore, &chat.Memo, &chat.CreatedAt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		chats = append(chats, chat)
	}

	c.JSON(200, chats)
}

func getChat(c *gin.Context) {
	id := c.Param("id")
	var chat Chat
	err := db.QueryRow("SELECT id, start_with_doctor, text, risk_score, memo, created_at FROM chats WHERE id = $1", id).
		Scan(&chat.ID, &chat.StartWithDoctor, &chat.Text, &chat.RiskScore, &chat.Memo, &chat.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Chat not found"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, chat)
}

func createChat(c *gin.Context) {
	var chat Chat
	if err := c.ShouldBindJSON(&chat); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	chat.CreatedAt = time.Now()
	err := db.QueryRow(
		"INSERT INTO chats (start_with_doctor, text, risk_score, memo, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		chat.StartWithDoctor, chat.Text, chat.RiskScore, chat.Memo, chat.CreatedAt,
	).Scan(&chat.ID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, chat)
}

func updateChat(c *gin.Context) {
	id := c.Param("id")
	var chat Chat
	if err := c.ShouldBindJSON(&chat); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(
		"UPDATE chats SET start_with_doctor = $1, text = $2, risk_score = $3, memo = $4 WHERE id = $5",
		chat.StartWithDoctor, chat.Text, chat.RiskScore, chat.Memo, id,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(404, gin.H{"error": "Chat not found"})
		return
	}

	c.JSON(200, chat)
}

func deleteChat(c *gin.Context) {
	id := c.Param("id")
	result, err := db.Exec("DELETE FROM chats WHERE id = $1", id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(404, gin.H{"error": "Chat not found"})
		return
	}

	c.JSON(200, gin.H{"message": "Chat deleted successfully"})
}
