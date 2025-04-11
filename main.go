package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

var db *sql.DB

func main() {
	// 개발 환경에서만 .env 파일 로드
	if os.Getenv("ENVIRONMENT") != "production" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	// 데이터베이스 연결 설정
	var dbURL string

	// 로컬 환경 또는 배포 환경 감지
	useLocalDB := os.Getenv("USE_LOCAL_DB")
	
	if useLocalDB == "true" {
		// 로컬 데이터베이스 연결 정보
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		
		portStr := os.Getenv("DB_PORT")
		port, err := strconv.Atoi(portStr)
		if err != nil || portStr == "" {
			port = 5432
		}
		
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "anymodb"
		}
		
		dbURL = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", 
			host, port, user, password, dbname)
		
		log.Printf("Using local database configuration: host=%s, port=%d, dbname=%s", host, port, dbname)
	} else {
		// Render 등의 배포 환경 URL 사용
		dbURL = os.Getenv("DATABASE_URL")
		if dbURL == "" {
			log.Fatal("DATABASE_URL environment variable is required when USE_LOCAL_DB is not set to true")
		}
		log.Println("Using DATABASE_URL configuration")
	}

	log.Printf("Connecting to database...")
	
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// 연결 테스트
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// 테이블 생성
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

	// Gin 라우터 초기화
	r := gin.Default()

	// CORS 미들웨어
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

	// 라우트 설정
	r.GET("/chats", getChats)
	r.GET("/chats/:id", getChat)
	r.POST("/chats", createChat)
	r.PUT("/chats/:id", updateChat)
	r.DELETE("/chats/:id", deleteChat)

	// 헬스 체크 엔드포인트
	r.GET("/health", func(c *gin.Context) {
		err := db.Ping()
		if err != nil {
			c.JSON(500, gin.H{"status": "error", "message": fmt.Sprintf("Database connection failed: %v", err)})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "message": "Server is running and connected to the database"})
	})

	// 서버 시작
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
	var input struct {
		StartWithDoctor *bool   `json:"startWithDoctor"`
		Text            string  `json:"text"`
		RiskScore       *int    `json:"riskScore"`
		Memo            *string `json:"memo"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 기본값 설정
	chat := Chat{
		CreatedAt: time.Now(),
		Text:      input.Text,
	}

	// StartWithDoctor 설정 (기본값: false)
	if input.StartWithDoctor != nil {
		chat.StartWithDoctor = *input.StartWithDoctor
	} else {
		chat.StartWithDoctor = false
	}

	// RiskScore 설정 (기본값: 0)
	if input.RiskScore != nil {
		chat.RiskScore = *input.RiskScore
	} else {
		chat.RiskScore = 0
	}

	// Memo 설정 (기본값: 빈 문자열)
	if input.Memo != nil {
		chat.Memo = *input.Memo
	} else {
		chat.Memo = ""
	}

	// 필수 필드 확인
	if chat.Text == "" {
		c.JSON(400, gin.H{"error": "text field is required"})
		return
	}

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
	
	// 기존 채팅 데이터 조회
	var existingChat Chat
	err := db.QueryRow("SELECT id, start_with_doctor, text, risk_score, memo, created_at FROM chats WHERE id = $1", id).
		Scan(&existingChat.ID, &existingChat.StartWithDoctor, &existingChat.Text, &existingChat.RiskScore, &existingChat.Memo, &existingChat.CreatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Chat not found"})
			return
		}
		log.Printf("Error retrieving existing chat with ID %s: %v", id, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	// 입력 구조체
	var input struct {
		StartWithDoctor *bool   `json:"startWithDoctor"`
		Text            *string `json:"text"`
		RiskScore       *int    `json:"riskScore"`
		Memo            *string `json:"memo"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 입력된 값이 있으면 업데이트
	if input.StartWithDoctor != nil {
		existingChat.StartWithDoctor = *input.StartWithDoctor
	}
	
	if input.Text != nil {
		existingChat.Text = *input.Text
	}
	
	if input.RiskScore != nil {
		existingChat.RiskScore = *input.RiskScore
	}
	
	if input.Memo != nil {
		existingChat.Memo = *input.Memo
	}

	// 데이터베이스 업데이트
	result, err := db.Exec(
		"UPDATE chats SET start_with_doctor = $1, text = $2, risk_score = $3, memo = $4 WHERE id = $5",
		existingChat.StartWithDoctor, existingChat.Text, existingChat.RiskScore, existingChat.Memo, id,
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

	c.JSON(200, existingChat)
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
