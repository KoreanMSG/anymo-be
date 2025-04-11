import logging
import os
import torch
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import numpy as np

# 환경 설정
logging.basicConfig(level=logging.INFO, 
                    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class SentimentAnalyzer:
    def __init__(self):
        """감정 분석을 위한 클래스 초기화"""
        try:
            # 기본 감정 분석 모델 로드 (Hugging Face transformers)
            model_name = "distilbert-base-uncased-finetuned-sst-2-english"  # 기본 감정 분석 모델
            
            logger.info(f"Loading sentiment analysis model: {model_name}")
            self.tokenizer = AutoTokenizer.from_pretrained(model_name)
            self.model = AutoModelForSequenceClassification.from_pretrained(model_name)
            
            # 추가 부정적 감정 키워드 - 자살 생각과 관련된 단어들
            self.negative_keywords = [
                'suicide', 'kill myself', 'end my life', 'want to die', 'better off dead',
                'no reason to live', 'can\'t go on', 'hopeless', 'worthless', 'burden',
                'painful', 'miserable', 'trapped', 'can\'t take it anymore', 'no future',
                'alone', 'lonely', 'abandoned', 'depressed', 'despair',
                'overdose', 'self-harm', 'cut myself', 'pills', 'gun',
                'plan to', 'goodbye', 'note', 'last wish', 'never wake up'
            ]
            
            logger.info("Sentiment analyzer initialized successfully")
        except Exception as e:
            logger.error(f"Failed to initialize sentiment analyzer: {e}")
            raise
    
    def predict_sentiment(self, text):
        """텍스트 감정 분석 예측 (positive/negative)"""
        try:
            # 모델 입력 준비
            inputs = self.tokenizer(text, return_tensors="pt", truncation=True, padding=True)
            
            # 예측
            with torch.no_grad():
                outputs = self.model(**inputs)
                
            # 결과 처리
            scores = outputs.logits.softmax(dim=1).numpy()[0]
            
            # 모델 결과는 [negative, positive] 형태
            negative_score = float(scores[0])
            positive_score = float(scores[1])
            
            return {
                'negative_score': negative_score,
                'positive_score': positive_score
            }
        
        except Exception as e:
            logger.error(f"Error in sentiment prediction: {e}")
            return {
                'negative_score': 0.5,
                'positive_score': 0.5
            }
    
    def analyze_suicide_sentiment(self, text):
        """텍스트에서 자살 경향 관련 감정 분석"""
        try:
            # 기본 감정 분석
            sentiment = self.predict_sentiment(text)
            negative_score = sentiment['negative_score']
            
            # 자살 관련 키워드 검출
            keyword_count = 0
            keyword_matches = []
            
            for keyword in self.negative_keywords:
                if keyword.lower() in text.lower():
                    keyword_count += 1
                    keyword_matches.append(keyword)
            
            # 자살 관련 키워드 가중치 (최대 0.3)
            keyword_weight = min(0.3, keyword_count * 0.02)
            
            # 최종 위험도 점수 계산
            # 기본 감정 분석 결과(negative_score)에 키워드 가중치 추가
            risk_score_raw = negative_score + keyword_weight
            
            # 점수를 1-100 범위로 조정
            risk_score = min(100, int(risk_score_raw * 100))
            
            # 위험 단계 분류
            risk_level = "Low Risk"
            if risk_score >= 75:
                risk_level = "Very High Risk"
            elif risk_score >= 50:
                risk_level = "High Risk"
            elif risk_score >= 25:
                risk_level = "Medium Risk"
                
            return {
                'risk_score': risk_score,
                'risk_level': risk_level,
                'negative_sentiment': negative_score,
                'keyword_matches': keyword_matches,
                'keyword_count': keyword_count
            }
            
        except Exception as e:
            logger.error(f"Error in suicide sentiment analysis: {e}")
            return {
                'risk_score': 50,  # 기본값
                'risk_level': "Medium Risk",
                'negative_sentiment': 0.5,
                'keyword_matches': [],
                'keyword_count': 0
            }

# 테스트 코드
if __name__ == "__main__":
    analyzer = SentimentAnalyzer()
    
    # 테스트 샘플
    test_samples = [
        "I'm feeling good today, everything is working out well.",
        "I don't know if I can continue living like this anymore. I feel hopeless.",
        "Sometimes I think about ending it all. I have the pills ready.",
        "I had a great day at work and enjoyed time with my family."
    ]
    
    for sample in test_samples:
        result = analyzer.analyze_suicide_sentiment(sample)
        print(f"\nText: {sample}")
        print(f"Risk score: {result['risk_score']}")
        print(f"Risk level: {result['risk_level']}")
        print(f"Negative sentiment: {result['negative_sentiment']:.4f}")
        print(f"Keywords: {result['keyword_matches']}") 