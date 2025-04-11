# 자살 위험도 분석 ML 서비스

HSIL-Korean-MSG 프로젝트의 머신러닝 부분 구현입니다. 환자-의사 대화에서 자살 위험도를 분석하는 API를 제공합니다.

## 디렉토리 구조

```
ml/
├── api.py                   # FastAPI 기반 API 서버
├── conversation_processor.py # Gemini를 사용한 대화 처리
├── suicide_predictor.py     # 자살 위험도 예측 모델
├── sentiment_analyzer.py    # 감정 분석 및 키워드 기반 분석
├── data/                    # 데이터 디렉토리
│   └── Suicide_Detection.csv # 자살 위험도 학습 데이터
├── models/                  # 학습된 모델 저장 디렉토리
├── requirements.txt         # 필요한 패키지 목록
├── .env.example             # 환경 변수 예시 파일
├── Dockerfile               # Docker 이미지 설정
└── render.yaml              # Render 배포 설정
```

## 데이터 파일 안내

본 레포지토리에는 크기 제한(100MB)으로 인해 전체 Suicide_Detection.csv 파일이 포함되어 있지 않습니다. 대신:

1. **샘플 데이터**: `Suicide_Detection_sample.csv` 파일에 1만 라인의 샘플 데이터가 포함되어 있어 기본 테스트가 가능합니다.

2. **압축 데이터**: `Suicide_Detection.csv.gz`는 전체 데이터셋을 gzip으로 압축한 파일입니다. 코드는 자동으로 이 파일을 감지하고 압축을 풀어 사용할 수 있습니다.

3. **전체 데이터 사용**: 전체 데이터셋(압축되지 않은)이 필요한 경우, 다음과 같이 준비할 수 있습니다:
   - 팀 구성원에게 전체 데이터셋 요청하기
   - 압축 파일 풀기: `gzip -d Suicide_Detection.csv.gz`

실행 시 코드는 다음 우선순위로 데이터 파일을 찾습니다:
1. Suicide_Detection_sample.csv (샘플 데이터)
2. Suicide_Detection.csv.gz (압축된 전체 데이터)
3. Suicide_Detection.csv (압축되지 않은 전체 데이터)

## 기능

1. **대화 처리**: Gemini AI를 사용하여 의사-환자 대화 식별 및 처리
2. **자살 위험도 예측**: CSV 데이터 기반 ML 모델로 자살 위험도 예측
3. **감정 분석**: 감정 분석 및 키워드 기반 자살 경향성 추가 분석
4. **API 통합**: 위 기능들을 제공하는 REST API

## 설치 및 실행

### 환경 설정

1. 먼저 setup.sh 스크립트를 실행하여 필요한 디렉토리와 파일을 설정합니다:
   ```bash
   chmod +x setup.sh
   ./setup.sh
   ```

2. 필요한 패키지 설치:
   ```bash
   pip install -r requirements.txt
   ```

3. `.env` 파일에 Gemini API 키를 설정:
   - [Google AI Studio](https://makersuite.google.com/app/apikey)에서 API 키를 발급받아 `.env` 파일에 설정

### 모델 학습

자살 위험도 예측 모델을 학습하려면:

```bash
python suicide_predictor.py
```

### API 서버 실행

```bash
python api.py
```

서버는 기본적으로 `http://localhost:8000`에서 실행됩니다.

## API 엔드포인트

### 1. 통합 분석

```
POST /analyze
```

단일 API 호출로 모든 분석을 수행합니다.

**요청 본문**:
```json
{
  "text": "의사-환자 대화 텍스트"
}
```

**응답 예시**:
```json
{
  "conversation": {
    "processed_text": "안녕하세요, 어떻게 지내세요?@@잠을 잘 못자고 있어요",
    "starts_with_doctor": true
  },
  "ml_prediction": {
    "risk_score": 35,
    "is_suicide_risk": false,
    "confidence": 0.35
  },
  "sentiment_analysis": {
    "risk_score": 45,
    "risk_level": "Medium Risk",
    "keyword_matches": [],
    "negative_sentiment": 0.45
  },
  "final_risk_score": 39
}
```

### 2. 대화 처리

```
POST /process-conversation
```

텍스트를 대화로 분리합니다.

### 3. 자살 위험도 예측

```
POST /predict-suicide
```

ML 모델을 통해 자살 위험도를 예측합니다.

### 4. 감정 분석

```
POST /analyze-sentiment
```

텍스트에서 감정 분석을 통해 자살 경향성을 평가합니다.

### 5. 모델 학습

```
POST /train
```

CSV 파일에서 모델을 학습/재학습합니다.

## Docker 실행

```bash
docker build -t suicide-analysis-api .
docker run -p 8000:8000 --env-file .env suicide-analysis-api
```

## Render 배포

1. GitHub에 코드를 푸시합니다
2. Render 대시보드에서 "Blueprint"를 선택하여 `render.yaml`을 사용한 배포를 시작합니다
3. 배포 후 Gemini API 키를 환경 변수로 설정합니다

## Go 백엔드와 Flutter 앱 연동

### Flutter 앱 연동

```dart
import 'dart:convert';
import 'package:http/http.dart' as http;

Future<Map<String, dynamic>> analyzeText(String text) async {
  final response = await http.post(
    Uri.parse('http://your-api-url/analyze'),
    headers: {'Content-Type': 'application/json'},
    body: json.encode({'text': text}),
  );
  
  if (response.statusCode == 200) {
    return json.decode(response.body);
  } else {
    throw Exception('Failed to analyze text');
  }
}
```

### Go 백엔드 연동

```go
import (
	"bytes"
	"encoding/json"
	"net/http"
)

type TextInput struct {
	Text string `json:"text"`
}

func analyzeText(text string) (map[string]interface{}, error) {
	input := TextInput{Text: text}
	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	
	resp, err := http.Post(
		"http://your-ml-api-url/analyze",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}
```