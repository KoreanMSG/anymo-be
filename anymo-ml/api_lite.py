import os
import logging
import uvicorn
import re
from fastapi import FastAPI, Request, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
from typing import List, Dict, Any, Optional
from enum import Enum
from datetime import datetime

# 기본 로깅 설정
logging.basicConfig(level=logging.INFO, 
                    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# FastAPI 앱 초기화
app = FastAPI(
    title="자살 위험도 분석 API - Lite 버전",
    description="텍스트 대화에서 자살 위험도를 분석하는 API (경량화 버전)",
    version="1.0.0-lite"
)

# CORS 설정
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 데이터 모델 정의
class TextInput(BaseModel):
    text: str = Field(..., description="분석할 텍스트")

class RiskLevel(str, Enum):
    NO_RISK = "No Risk"
    LOW_RISK = "Low Risk"
    MEDIUM_RISK = "Medium Risk"
    HIGH_RISK = "High Risk"
    SEVERE_RISK = "Severe Risk"

class AnalysisResult(BaseModel):
    risk_score: int = Field(..., description="위험도 점수 (1-100)")
    risk_level: RiskLevel = Field(..., description="위험도 레벨")
    keywords_found: List[str] = Field(default=[], description="발견된 위험 키워드")
    analysis_time: str = Field(..., description="분석 시간")

# 자살 관련 키워드 (한국어와 영어)
SUICIDE_KEYWORDS = {
    "high": [
        "자살", "죽고 싶", "목숨", "목맬", "목을 매", "죽어버리", "없어지고 싶", "사라지고 싶", 
        "suicide", "kill myself", "end my life", "take my life", "don't want to live",
        "better off dead", "cannot go on", "방법"
    ],
    "medium": [
        "우울", "희망이 없", "삶이 의미", "살아갈 이유", "혼자", "외롭", "고통", "아파", 
        "hopeless", "worthless", "alone", "lonely", "pain", "suffering", "depressed",
        "depression", "pills", "overdose", "goodbye", "farewell"
    ],
    "low": [
        "슬프", "힘들", "지쳤", "피곤", "실패", "포기", "도움", "아무도 몰라", 
        "sad", "tired", "exhausted", "failed", "give up", "help me", "nobody understands",
        "crying", "tears", "hate myself"
    ]
}

# 라이트 버전의 텍스트 분석 함수
def analyze_text_lite(text: str) -> Dict[str, Any]:
    """간단한 키워드 기반 자살 위험도 분석"""
    start_time = datetime.now()
    
    # 모든 텍스트를 소문자로 변환
    text_lower = text.lower()
    
    # 키워드 검색
    found_keywords = {
        "high": [kw for kw in SUICIDE_KEYWORDS["high"] if kw.lower() in text_lower],
        "medium": [kw for kw in SUICIDE_KEYWORDS["medium"] if kw.lower() in text_lower],
        "low": [kw for kw in SUICIDE_KEYWORDS["low"] if kw.lower() in text_lower]
    }
    
    # 위험도 점수 계산
    high_count = len(found_keywords["high"]) * 15
    medium_count = len(found_keywords["medium"]) * 7
    low_count = len(found_keywords["low"]) * 3
    
    # 반복 사용 가중치
    repetition_multiplier = 1.0
    for level in ["high", "medium", "low"]:
        for kw in found_keywords[level]:
            count = len(re.findall(re.escape(kw.lower()), text_lower))
            if count > 1:
                repetition_multiplier += 0.1 * (count - 1)
    
    # 총 점수 계산 (최대 100점)
    score = min(100, int((high_count + medium_count + low_count) * repetition_multiplier))
    
    # 위험도 레벨 결정
    if score >= 80:
        risk_level = RiskLevel.SEVERE_RISK
    elif score >= 60:
        risk_level = RiskLevel.HIGH_RISK
    elif score >= 40:
        risk_level = RiskLevel.MEDIUM_RISK
    elif score >= 20:
        risk_level = RiskLevel.LOW_RISK
    else:
        risk_level = RiskLevel.NO_RISK
    
    # 모든 키워드를 하나의 리스트로 합침
    all_keywords = found_keywords["high"] + found_keywords["medium"] + found_keywords["low"]
    
    return {
        "risk_score": score,
        "risk_level": risk_level,
        "keywords_found": all_keywords,
        "analysis_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    }

# 루트 엔드포인트 
@app.get("/")
def read_root():
    return {"message": "자살 위험도 분석 API - Lite 버전", "status": "online"}

# 상태 확인 엔드포인트
@app.get("/status")
def check_status():
    port = os.getenv("PORT", 8000)
    return {
        "status": "running",
        "port": port,
        "version": "lite",
        "environment": os.getenv("ENVIRONMENT", "development"),
        "memory_usage": "low",
        "features": ["basic_text_analysis", "keyword_detection"]
    }

# 간단한 텍스트 분석 엔드포인트
@app.post("/analyze", response_model=AnalysisResult)
def analyze_text(text_input: TextInput):
    try:
        if not text_input.text or len(text_input.text.strip()) < 3:
            raise HTTPException(status_code=400, detail="Text too short for analysis")
        
        # 간단한 키워드 기반 분석
        result = analyze_text_lite(text_input.text)
        return result
    except Exception as e:
        logger.error(f"Error in analysis: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# 에러 처리 미들웨어
@app.middleware("http")
async def errors_handling(request: Request, call_next):
    try:
        return await call_next(request)
    except Exception as exc:
        logger.error(f"Error in request: {exc}")
        return JSONResponse(
            status_code=500,
            content={"detail": str(exc)},
        )

# 서버 시작 함수
def start_server():
    """API 서버 실행"""
    port = int(os.getenv("PORT", 8000))
    logger.info(f"Starting server on port {port}...")
    
    # host를 "0.0.0.0"으로 설정하는 것이 중요함
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=port,
        log_level="info"
    )

if __name__ == "__main__":
    logger.info("API Lite script started")
    start_server() 