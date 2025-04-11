# HSIL Korean Message API Documentation

## Base Information

- **Base URL**: `https://hsil-korean-msg-api.onrender.com`
- **Content Type**: `application/json`

## Development Setup

### Prerequisites
- Go 1.16+
- PostgreSQL (원격 데이터베이스 사용)

### Local Setup
1. Clone the repository
2. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```
3. 필요시 .env 파일의 DATABASE_URL이나 PORT를 수정
4. Build and run the application:
   ```bash
   go build -o main
   ./main
   ```

### Environment Variables
- `DATABASE_URL`: 데이터베이스 접속 정보 (기본값은 Render에 배포된 PostgreSQL 데이터베이스)
- `ENVIRONMENT`: "development" 또는 "production" (로컬 개발 환경에서는 "development")
- `PORT`: 서버 포트 (기본값: 8080)

## Chat API Endpoints

### 1. Get All Chats

```
GET /chats
```

Returns all chat records sorted by creation date (descending order).

**Example Response**:
```json
[
  {
    "id": 1,
    "startWithDoctor": true,
    "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain recently.",
    "riskScore": 35,
    "memo": "First-time patient",
    "createdAt": "2023-04-10T15:30:00Z"
  },
  {
    "id": 2,
    "startWithDoctor": false,
    "text": "What symptoms are you experiencing?@@I have a headache.@@How long have you been having this pain?",
    "riskScore": 65,
    "memo": "Headache patient",
    "createdAt": "2023-04-09T14:15:00Z"
  }
]
```

**Example cURL**:
```bash
curl -X GET https://hsil-korean-msg-api.onrender.com/chats
```

### 2. Get Specific Chat

```
GET /chats/{id}
```

**Path Parameters**:
- `id` (required): ID of the chat to retrieve

**Example Response**:
```json
{
  "id": 1,
  "startWithDoctor": true,
  "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain recently.",
  "riskScore": 35,
  "memo": "First-time patient",
  "createdAt": "2023-04-10T15:30:00Z"
}
```

**Example cURL**:
```bash
curl -X GET https://hsil-korean-msg-api.onrender.com/chats/1
```

**Error Response** (chat not found):
```json
{
  "error": "Chat not found"
}
```

### 3. Create New Chat

```
POST /chats
```

**Request Body**:
```json
{
  "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain."
}
```

**Required Fields**:
- `text`: Message content (conversations separated by `@@`)

**Optional Fields**:
- `startWithDoctor`: Whether a doctor started the conversation (boolean, default: false)
- `riskScore`: Risk score (1-100, default: 0)
- `memo`: Additional notes (default: empty string)

**Example Response**:
```json
{
  "id": 3,
  "startWithDoctor": false,
  "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain.",
  "riskScore": 0,
  "memo": "",
  "createdAt": "2023-04-11T10:25:30Z"
}
```

**Example cURL**:
```bash
curl -X POST https://hsil-korean-msg-api.onrender.com/chats \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain."
  }'
```

### 4. Update Chat

```
PUT /chats/{id}
```

**Path Parameters**:
- `id` (required): ID of the chat to update

**Request Body** (all fields are optional):
```json
{
  "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain.@@How long have you been experiencing this pain?@@Since yesterday."
}
```

**Optional Fields**:
- `startWithDoctor`: Whether a doctor started the conversation (boolean)
- `text`: Message content (conversations separated by `@@`)
- `riskScore`: Risk score (1-100)
- `memo`: Additional notes

**Example Response**:
```json
{
  "id": 3,
  "startWithDoctor": false,
  "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain.@@How long have you been experiencing this pain?@@Since yesterday.",
  "riskScore": 0,
  "memo": "",
  "createdAt": "2023-04-11T10:25:30Z"
}
```

**Example cURL**:
```bash
curl -X PUT https://hsil-korean-msg-api.onrender.com/chats/3 \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, how can I help you?@@Hi, I've been having abdominal pain.@@How long have you been experiencing this pain?@@Since yesterday."
  }'
```

**Error Response** (chat not found):
```json
{
  "error": "Chat not found"
}
```

### 5. Delete Chat

```
DELETE /chats/{id}
```

**Path Parameters**:
- `id` (required): ID of the chat to delete

**Example Response** (success):
```json
{
  "message": "Chat deleted successfully"
}
```

**Example cURL**:
```bash
curl -X DELETE https://hsil-korean-msg-api.onrender.com/chats/3
```

**Error Response** (chat not found):
```json
{
  "error": "Chat not found"
}
```

## Data Model

### Chat Model

The Chat model follows the database schema:

| Field | Type | Description |
|-------|------|-------------|
| id | integer | Chat ID (auto-generated) |
| startWithDoctor | boolean | Whether a doctor started the conversation |
| text | string | Conversation content separated by `@@` |
| riskScore | integer | Risk score (1-100) |
| memo | string | Additional notes (optional) |
| createdAt | datetime | Creation time (auto-generated) |

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS chats (
    id SERIAL PRIMARY KEY,
    start_with_doctor BOOLEAN NOT NULL,
    text TEXT NOT NULL,
    risk_score INTEGER NOT NULL,
    memo TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## JavaScript Example

```javascript
// Example: Get all chats
async function getAllChats() {
  try {
    const response = await fetch('https://hsil-korean-msg-api.onrender.com/chats');
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }
    const data = await response.json();
    console.log(data);
    return data;
  } catch (error) {
    console.error('Error fetching chats:', error);
  }
}

// Example: Create a new chat
async function createChat(chatData) {
  try {
    const response = await fetch('https://hsil-korean-msg-api.onrender.com/chats', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        startWithDoctor: chatData.startWithDoctor,
        text: chatData.text,
        riskScore: chatData.riskScore,
        memo: chatData.memo
      }),
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }
    
    const data = await response.json();
    console.log('Created chat:', data);
    return data;
  } catch (error) {
    console.error('Error creating chat:', error);
  }
}

// Example usage
createChat({
  startWithDoctor: true,
  text: "Hello, what symptoms are you experiencing?@@I have a fever and cough.",
  riskScore: 40,
  memo: "Respiratory symptoms"
});
```

## Error Handling

All API endpoints return appropriate HTTP status codes along with error messages in JSON format in case of failure:

- `400 Bad Request`: Invalid request format
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server-side error

## Health Check

The API provides a health endpoint to verify the service status:

```
GET /health
```

**Example Response** (service healthy):
```json
{
  "status": "ok",
  "message": "Server is running and connected to the database"
}
```

**Example cURL**:
```bash
curl -X GET https://hsil-korean-msg-api.onrender.com/health
```
