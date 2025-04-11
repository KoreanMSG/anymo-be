# HSIL 한국어 메시지 API 문서

## 기본 정보

- **기본 URL**: `https://hsil-korean-msg-api.onrender.com`
- **콘텐츠 타입**: `application/json`

## 인증

현재 API는 별도의 인증 없이 사용 가능합니다.

## 채팅 API 엔드포인트

### 1. 모든 채팅 목록 조회

```
GET /chats
```

모든
채팅 기록을 최신순(생성일 기준 내림차순)으로 반환합니다.

**응답 예시**:
```json
[
  {
    "id": 1,
    "startWithDoctor": true,
    "text": "안녕하세요, 어떻게 도와드릴까요?@@안녕하세요, 최근에 복통이 있어서요.",
    "riskScore": 35,
    "memo": "첫 방문 환자",
    "createdAt": "2023-04-10T15:30:00Z"
  },
  {
    "id": 2,
    "startWithDoctor": false,
    "text": "어떤 증상이 있으신가요?@@머리가 아파요.@@언제부터 아프셨나요?",
    "riskScore": 65,
    "memo": "두통 환자",
    "createdAt": "2023-04-09T14:15:00Z"
  }
]
```

### 2. 특정 채팅 조회

```
GET /chats/{id}
```

**Path 파라미터**:
- `id` (필수): 조회할 채팅의 ID

**응답 예시**:
```json
{
  "id": 1,
  "startWithDoctor": true,
  "text": "안녕하세요, 어떻게 도와드릴까요?@@안녕하세요, 최근에 복통이 있어서요.",
  "riskScore": 35,
  "memo": "첫 방문 환자",
  "createdAt": "2023-04-10T15:30:00Z"
}
```

**오류 응답** (채팅을 찾을 수 없는 경우):
```json
{
  "error": "Chat not found"
}
```

### 3. 새 채팅 생성

```
POST /chats
```

**요청 본문**:
```json
{
  "startWithDoctor": true,
  "text": "안녕하세요, 어떻게 도와드릴까요?@@안녕하세요, 복통이 있어서요.",
  "riskScore": 45,
  "memo": "복통 환자"
}
```

**필수 필드**:
- `startWithDoctor`: 대화 시작자가 의사인지 여부 (boolean)
- `text`: 메시지 내용 (`@@`로 구분된 대화 내용)
- `riskScore`: 위험도 점수 (1-100)

**선택 필드**:
- `memo`: 메모 (기본값: 빈 문자열)

**응답 예시**:
```json
{
  "id": 3,
  "startWithDoctor": true,
  "text": "안녕하세요, 어떻게 도와드릴까요?@@안녕하세요, 복통이 있어서요.",
  "riskScore": 45,
  "memo": "복통 환자",
  "createdAt": "2023-04-11T10:25:30Z"
}
```

### 4. 채팅 업데이트

```
PUT /chats/{id}
```

**Path 파라미터**:
- `id` (필수): 업데이트할 채팅의 ID

**요청 본문**:
```json
{
  "startWithDoctor": true,
  "text": "안녕하세요, 어떻게 도와드릴까요?@@안녕하세요, 복통이 있어서요.@@언제부터 아프셨나요?@@어제부터요.",
  "riskScore": 50,
  "memo": "복통 환자 - 추가 증상 확인 필요"
}
```

**응답 예시**:
```json
{
  "id": 3,
  "startWithDoctor": true,
  "text": "안녕하세요, 어떻게 도와드릴까요?@@안녕하세요, 복통이 있어서요.@@언제부터 아프셨나요?@@어제부터요.",
  "riskScore": 50,
  "memo": "복통 환자 - 추가 증상 확인 필요",
  "createdAt": "2023-04-11T10:25:30Z"
}
```

**오류 응답** (채팅을 찾을 수 없는 경우):
```json
{
  "error": "Chat not found"
}
```

### 5. 채팅 삭제

```
DELETE /chats/{id}
```

**Path 파라미터**:
- `id` (필수): 삭제할 채팅의 ID

**응답 예시** (성공):
```json
{
  "message": "Chat deleted successfully"
}
```

**오류 응답** (채팅을 찾을 수 없는 경우):
```json
{
  "error": "Chat not found"
}
```

## 데이터 모델

### Chat 모델

| 필드 | 타입 | 설명 |
|------|------|------|
| id | integer | 채팅 ID (자동 생성) |
| startWithDoctor | boolean | 대화 시작자가 의사인지 여부 |
| text | string | `@@`로 구분된 대화 내용 |
| riskScore | integer | 위험도 점수 (1-100) |
| memo | string | 메모 (선택 사항) |
| createdAt | datetime | 생성 시간 (자동 생성) |

## Flutter 통합 예제

```dart
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:your_app/models/chat_model.dart';

class ChatService {
  final String baseUrl = 'https://hsil-korean-msg-api.onrender.com';
  
  // 모든 채팅 가져오기
  Future<List<Chat>> getAllChats() async {
    final response = await http.get(Uri.parse('$baseUrl/chats'));
    
    if (response.statusCode == 200) {
      List<dynamic> data = json.decode(response.body);
      return data.map((json) => Chat(
        id: json['id'],
        startWithDoctor: json['startWithDoctor'],
        text: json['text'],
        riskScore: json['riskScore'],
        memo: json['memo'] ?? '',
        createdAt: DateTime.parse(json['createdAt']),
      )).toList();
    } else {
      throw Exception('Failed to load chats');
    }
  }
  
  // 새 채팅 생성
  Future<Chat> createChat(Chat chat) async {
    final response = await http.post(
      Uri.parse('$baseUrl/chats'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'startWithDoctor': chat.startWithDoctor,
        'text': chat.text,
        'riskScore': chat.riskScore,
        'memo': chat.memo,
      }),
    );
    
    if (response.statusCode == 201) {
      Map<String, dynamic> data = json.decode(response.body);
      return Chat(
        id: data['id'],
        startWithDoctor: data['startWithDoctor'],
        text: data['text'],
        riskScore: data['riskScore'],
        memo: data['memo'] ?? '',
        createdAt: DateTime.parse(data['createdAt']),
      );
    } else {
      throw Exception('Failed to create chat');
    }
  }
}
```

## 에러 처리

모든 API 엔드포인트는 실패 시 적절한 HTTP 상태 코드와 함께 오류 메시지를 JSON 형식으로 반환합니다:

- `400 Bad Request`: 잘못된 요청 형식
- `404 Not Found`: 리소스를 찾을 수 없음
- `500 Internal Server Error`: 서버 내부 오류

## 주의사항

- `text` 필드의 메시지는 `@@` 구분자로 분리됩니다.
- `startWithDoctor` 값에 따라 메시지 순서가 결정됩니다:
  - `true`: 홀수 인덱스 메시지는 환자, 짝수 인덱스 메시지는 의사
  - `false`: 홀수 인덱스 메시지는 의사, 짝수 인덱스 메시지는 환자
- `riskScore`에 따른 위험도 레이블:
  - 1-24: "Low Risk"
  - 25-49: "Medium Risk"
  - 50-74: "High Risk"
  - 75-100: "Very High Risk"
