package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Chat struct {
	ID              int       `json:"id"`
	StartWithDoctor bool      `json:"startWithDoctor"`
	Text            string    `json:"text"`
	RiskScore       int       `json:"riskScore"`
	Memo            string    `json:"memo"`
	CreatedAt       time.Time `json:"createdAt"`
}

var host = "localhost"

// var host = os.Getenv("databaseURL")
var port = 5432
var user = os.Getenv("username")
var password = os.Getenv("password")
var dbname = "anymodb"

var db *sql.DB

func main() {
	pgConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	conn, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	db = conn
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	fmt.Println("Connected to the PostgreSQL database")
	db, err = sql.Open("postgres", host)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			id SERIAL PRIMARY KEY,
			start_with_doctor BOOLEAN NOT NULL,
			text TEXT NOT NULL,
			risk_score INTEGER NOT NULL,
			memo TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	log.Println("Database table checked/created")

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

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		err := db.Ping()
		if err != nil {
			c.JSON(500, gin.H{"status": "error", "message": fmt.Sprintf("Database connection failed: %v", err)})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "message": "Server is running and connected to the database"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}

func getChats(c *gin.Context) {
	rows, err := db.Query("SELECT id, start_with_doctor, text, risk_score, memo, created_at FROM chats ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Error querying chats: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		err := rows.Scan(&chat.ID, &chat.StartWithDoctor, &chat.Text, &chat.RiskScore, &chat.Memo, &chat.CreatedAt)
		if err != nil {
			log.Printf("Error scanning chat row: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after scanning rows: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
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
		log.Printf("Error retrieving chat with ID %s: %v", id, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, chat)
}

func createChat(c *gin.Context) {
	var chat Chat
	if err := c.ShouldBindJSON(&chat); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	chat.CreatedAt = time.Now()
	err := db.QueryRow(
		"INSERT INTO chats (start_with_doctor, text, risk_score, memo, created_at) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		chat.StartWithDoctor, chat.Text, chat.RiskScore, chat.Memo, chat.CreatedAt,
	).Scan(&chat.ID)

	if err != nil {
		log.Printf("Error creating chat: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, chat)
}

func updateChat(c *gin.Context) {
	id := c.Param("id")
	var chat Chat
	if err := c.ShouldBindJSON(&chat); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(
		"UPDATE chats SET start_with_doctor = $1, text = $2, risk_score = $3, memo = $4 WHERE id = $5",
		chat.StartWithDoctor, chat.Text, chat.RiskScore, chat.Memo, id,
	)
	if err != nil {
		log.Printf("Error updating chat with ID %s: %v", id, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
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
		log.Printf("Error deleting chat with ID %s: %v", id, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(404, gin.H{"error": "Chat not found"})
		return
	}

	c.JSON(200, gin.H{"message": "Chat deleted successfully"})
}
