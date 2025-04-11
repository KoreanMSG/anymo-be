package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
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
var mlAPIURL = "https://anymo-ml.onrender.com"

func main() {
	// 로컬 개발 환경에서만 .env 파일 로드 (production이 아닌 경우)
	if os.Getenv("ENVIRONMENT") != "production" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	// ML API URL 환경변수 확인
	if envMLAPIURL := os.Getenv("ML_API_URL"); envMLAPIURL != "" {
		mlAPIURL = envMLAPIURL
	}
	log.Printf("Using ML API URL: %s", mlAPIURL)

	// 데이터베이스 연결 설정
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	log.Println("Connecting to database...")
	
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
	r.POST("/processChat", processChat)

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

// ML API와 연동하여 자살 위험도 분석
type SuicideRiskRequest struct {
	Text string `json:"text"`
}

type SuicideRiskResponse struct {
	Score int `json:"score"`
}

func analyzeSuicideRisk(text string) (int, error) {
	url := fmt.Sprintf("%s/suicide-risk", mlAPIURL)
	reqBody, err := json.Marshal(SuicideRiskRequest{Text: text})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal suicide risk request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("failed to make suicide risk API request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("suicide risk API returned error, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var riskResponse SuicideRiskResponse
	if err := json.NewDecoder(resp.Body).Decode(&riskResponse); err != nil {
		return 0, fmt.Errorf("failed to decode suicide risk response: %v", err)
	}

	return riskResponse.Score, nil
}

// ML API와 연동하여 감정 분석
type SentimentRequest struct {
	Text string `json:"text"`
}

type SentimentResponse struct {
	Sentiment string `json:"sentiment"`
}

func analyzeSentiment(text string) (string, error) {
	url := fmt.Sprintf("%s/sentiment", mlAPIURL)
	reqBody, err := json.Marshal(SentimentRequest{Text: text})
	if err != nil {
		return "", fmt.Errorf("failed to marshal sentiment request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to make sentiment API request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sentiment API returned error, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var sentimentResponse SentimentResponse
	if err := json.NewDecoder(resp.Body).Decode(&sentimentResponse); err != nil {
		return "", fmt.Errorf("failed to decode sentiment response: %v", err)
	}

	return sentimentResponse.Sentiment, nil
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

	// 필수 필드 확인
	if chat.Text == "" {
		c.JSON(400, gin.H{"error": "text field is required"})
		return
	}

	// ML API를 통해 자살 위험도 분석
	riskScore, err := analyzeSuicideRisk(chat.Text)
	if err != nil {
		log.Printf("Error analyzing suicide risk: %v", err)
		// API 호출 실패 시 수동 입력 값 또는 기본값(0) 사용
		if input.RiskScore != nil {
			chat.RiskScore = *input.RiskScore
		} else {
			chat.RiskScore = 0
		}
	} else {
		// API에서 받은 위험도 점수 사용 (이미 설정된 경우 재정의)
		chat.RiskScore = riskScore
	}

	// ML API를 통해 감정 분석
	sentiment, err := analyzeSentiment(chat.Text)
	if err != nil {
		log.Printf("Error analyzing sentiment: %v", err)
		// 감정 분석은 memo로 저장, 실패 시 입력 memo 사용
		if input.Memo != nil {
			chat.Memo = *input.Memo
		} else {
			chat.Memo = ""
		}
	} else {
		// 분석된 감정을 memo에 저장하되, 사용자 memo가 있으면 합쳐서 저장
		if input.Memo != nil && *input.Memo != "" {
			chat.Memo = fmt.Sprintf("Sentiment: %s | %s", sentiment, *input.Memo)
		} else {
			chat.Memo = fmt.Sprintf("Sentiment: %s", sentiment)
		}
	}

	err = db.QueryRow(
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

type ProcessChatRequest struct {
	CreatedAt string `json:"createdAt"` // You can parse this into time.Time later if needed.
	Text      string `json:"text"`
	Memo      string `json:"memo"`
}

// Response payload struct
type ProcessChatResponse struct {
	CreatedAt       string `json:"createdAt"`
	Text            string `json:"text"` // This will contain the updated dialogue with "@@" markers.
	Memo            string `json:"memo"`
	StartWithDoctor bool   `json:"startWithDoctor"` // Set based on LLM feedback.
}

// processChat accepts the original conversation data, calls Gemini via callLLMDirect, and returns structured output.
func processChat(c *gin.Context) {
	// Define the expected input structure.
	var req struct {
		CreatedAt string `json:"createdAt"`
		Text      string `json:"text"`
		Memo      string `json:"memo"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Call the Gemini API using the direct REST approach.
	updatedText, startWithDoctor, err := callLLMDirect(req.Text)
	if err != nil {
		c.JSON(500, gin.H{"error": "LLM processing error: " + err.Error()})
		return
	}

	// Build the response payload.
	resp := struct {
		CreatedAt       string `json:"createdAt"`
		Text            string `json:"text"`
		Memo            string `json:"memo"`
		StartWithDoctor bool   `json:"startWithDoctor"`
	}{
		CreatedAt:       req.CreatedAt,
		Text:            updatedText,
		Memo:            req.Memo,
		StartWithDoctor: startWithDoctor,
	}

	c.JSON(200, resp)
}

func callLLMDirect(originalText string) (string, bool, error) {
	// Create a context with timeout for the API call.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a Gemini client using your API key.
	// (Note: The new Gemini API client does not require a project or location.)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return "", false, fmt.Errorf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	// Instantiate the model with the desired version.
	// (You can change the model name if needed; for example "gemini-2.0-flash" is also available.)
	model := client.GenerativeModel("gemini-2.0-flash-lite-001")
	// Tell the model to output JSON.
	model.ResponseMIMEType = "application/json"
	// Provide a JSON schema so that the model always responds with our expected format.
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"updatedText": {
				Type: genai.TypeString,
			},
			"startWithDoctor": {
				Type: genai.TypeBoolean,
			},
		},
		Required: []string{"updatedText", "startWithDoctor"},
	}

	// Construct the prompt, instructing the model to process the dialogue.
	prompt := fmt.Sprintf(
		"Process the conversation below by inserting '@@' markers where the speaker changes. Also determine if the conversation starts with a doctor. Return a JSON object with the following fields:\n"+
			"  updatedText (string): the conversation with '@@' markers inserted,\n"+
			"  startWithDoctor (boolean): true if the first utterance is from the doctor, false otherwise.\n"+
			"Conversation: %s",
		originalText,
	)

	// Generate content using the Gemini model.
	respGen, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", false, fmt.Errorf("LLM API error: %v", err)
	}

	// Ensure that we have at least one candidate in the response.
	if len(respGen.Candidates) == 0 {
		return "", false, fmt.Errorf("no candidates returned from LLM")
	}

	// Extract the JSON response from the first candidate.
	var jsonResponse string
	for _, part := range respGen.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			jsonResponse = string(textPart)
			break
		}
	}
	if jsonResponse == "" {
		return "", false, fmt.Errorf("failed to retrieve JSON response from LLM")
	}

	// Decode the JSON response.
	var result struct {
		UpdatedText     string `json:"updatedText"`
		StartWithDoctor bool   `json:"startWithDoctor"`
	}
	if err := json.Unmarshal([]byte(jsonResponse), &result); err != nil {
		return "", false, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	return result.UpdatedText, result.StartWithDoctor, nil
}
