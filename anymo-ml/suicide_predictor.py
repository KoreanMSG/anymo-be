import os
import pandas as pd
import numpy as np
import logging
import joblib
import gzip
from sklearn.model_selection import train_test_split
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.pipeline import Pipeline
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import classification_report, accuracy_score
import nltk
from nltk.corpus import stopwords
from nltk.tokenize import word_tokenize
from nltk.stem import WordNetLemmatizer
import re

# 환경 설정
logging.basicConfig(level=logging.INFO, 
                    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# NLTK 리소스 다운로드
try:
    # NLTK 데이터 디렉토리 설정 (Render 환경에서 쓰기 가능한 경로로)
    nltk_data_dir = os.path.join(os.path.dirname(__file__), 'nltk_data')
    os.makedirs(nltk_data_dir, exist_ok=True)
    
    # NLTK 데이터 경로 설정
    nltk.data.path.append(nltk_data_dir)
    
    # 필요한 데이터 다운로드
    nltk.download('punkt', download_dir=nltk_data_dir, quiet=True)
    nltk.download('stopwords', download_dir=nltk_data_dir, quiet=True)
    nltk.download('wordnet', download_dir=nltk_data_dir, quiet=True)
    
    logger.info(f"NLTK resources downloaded to {nltk_data_dir}")
except Exception as e:
    logger.warning(f"NLTK download warning: {e}")
    # 에러가 있더라도 기본적인 처리 가능하도록 준비
    try:
        from nltk.tokenize import word_tokenize as _word_tokenize
        def word_tokenize(text):
            try:
                return _word_tokenize(text)
            except:
                return text.split()
        
        if not hasattr(nltk, 'corpus') or not hasattr(nltk.corpus, 'stopwords') or not hasattr(nltk.corpus.stopwords, 'words'):
            nltk.corpus.stopwords.words = lambda lang: []
        
        if not hasattr(nltk, 'stem') or not hasattr(nltk.stem, 'WordNetLemmatizer'):
            class DummyLemmatizer:
                def lemmatize(self, word):
                    return word
            nltk.stem.WordNetLemmatizer = DummyLemmatizer
            
        logger.info("Set up fallback NLTK functionality")
    except Exception as inner_e:
        logger.error(f"Failed to set up fallback NLTK functionality: {inner_e}")

class SuicidePredictor:
    def __init__(self, model_path='models/suicide_model.joblib'):
        self.model_path = model_path
        self.model = None
        self.vectorizer = None
        self.keywords = {
            "high": [],
            "medium": [],
            "low": []
        }
        
        try:
            self.lemmatizer = WordNetLemmatizer()
            try:
                self.stop_words = set(stopwords.words('english'))
            except Exception as e:
                logger.warning(f"Could not load stopwords: {e}")
                self.stop_words = set()
        except Exception as e:
            logger.warning(f"Could not initialize lemmatizer: {e}")
            # 폴백 처리
            class DummyLemmatizer:
                def lemmatize(self, word):
                    return word
            self.lemmatizer = DummyLemmatizer()
            self.stop_words = set()
        
        # 모델 로드 또는 생성
        if os.path.exists(model_path):
            try:
                self.model = joblib.load(model_path)
                logger.info(f"Model loaded from {model_path}")
            except Exception as e:
                logger.error(f"Failed to load model: {e}")
                self.model = None
    
    def preprocess_text(self, text):
        """
        텍스트 전처리 함수
        
        Parameters:
        -----------
        text : str
            전처리할 원본 텍스트
        
        Returns:
        --------
        str
            전처리된 텍스트
        """
        if not text:
            return ""
        
        # 소문자 변환
        text = text.lower()
        
        # 한글과 영어, 숫자, 공백만 남기고 나머지는 제거
        text = re.sub(r'[^\w\s가-힣]', ' ', text)
        
        # 연속된 공백 제거
        text = re.sub(r'\s+', ' ', text)
        
        # 앞뒤 공백 제거
        text = text.strip()
        
        return text
    
    def extract_keywords_from_csv(self, csv_path):
        """
        CSV 파일에서 자살 관련 키워드를 추출합니다.
        
        Args:
            csv_path (str): CSV 파일 경로
        
        Returns:
            bool: 추출 성공 여부
        """
        try:
            # CSV 파일 로드
            df = pd.read_csv(csv_path)
            
            # 최소 2개 이상의 열이 있는지 확인 (텍스트, 라벨)
            if len(df.columns) < 2:
                logger.warning("CSV 파일의 열이 부족합니다. 기본 키워드를 사용합니다.")
                self._set_default_keywords()
                return False
            
            # 텍스트 열과 라벨 열 추출 (첫 번째와 두 번째 열 가정)
            text_col = df.columns[0]
            label_col = df.columns[1]
            
            # 라벨이 1인 텍스트만 선택 (자살 관련 텍스트)
            suicide_texts = df[df[label_col] == 1][text_col].astype(str).tolist()
            non_suicide_texts = df[df[label_col] == 0][text_col].astype(str).tolist()
            
            if len(suicide_texts) == 0:
                logger.warning("자살 관련 텍스트가 없습니다. 기본 키워드를 사용합니다.")
                self._set_default_keywords()
                return False
            
            # TF-IDF 벡터라이저 적용
            tfidf = TfidfVectorizer(max_features=200, stop_words='english', ngram_range=(1, 2))
            tfidf_matrix = tfidf.fit_transform(suicide_texts)
            feature_names = tfidf.get_feature_names_out()
            
            # 각 단어의 TF-IDF 점수 평균 계산
            tfidf_means = tfidf_matrix.mean(axis=0).A1
            
            # 단어와 점수를 튜플 리스트로 변환
            word_scores = [(word, score) for word, score in zip(feature_names, tfidf_means)]
            
            # 점수 기준으로 정렬
            word_scores.sort(key=lambda x: x[1], reverse=True)
            
            # 위험도별 키워드 수 계산
            total_keywords = len(word_scores)
            high_risk_count = min(int(total_keywords * 0.2), 50)  # 상위 20%, 최대 50개
            medium_risk_count = min(int(total_keywords * 0.3), 100)  # 상위 30%, 최대 100개
            low_risk_count = min(total_keywords - high_risk_count - medium_risk_count, 150)  # 나머지, 최대 150개
            
            # 키워드 분류
            self.keywords = {
                "high": [word for word, _ in word_scores[:high_risk_count]],
                "medium": [word for word, _ in word_scores[high_risk_count:high_risk_count + medium_risk_count]],
                "low": [word for word, _ in word_scores[high_risk_count + medium_risk_count:high_risk_count + medium_risk_count + low_risk_count]]
            }
            
            # 추출된 키워드 검증
            if not all(len(self.keywords[level]) > 0 for level in ["high", "medium", "low"]):
                logger.warning("일부 위험도 레벨에 키워드가 없습니다. 기본 키워드를 추가합니다.")
                self._add_default_keywords()
            
            logger.info(f"키워드 추출 완료: 고위험 {len(self.keywords['high'])}개, 중위험 {len(self.keywords['medium'])}개, 저위험 {len(self.keywords['low'])}개")
            print(f"고위험 키워드 예시: {self.keywords['high'][:5]}")
            print(f"중위험 키워드 예시: {self.keywords['medium'][:5]}")
            print(f"저위험 키워드 예시: {self.keywords['low'][:5]}")
            
            return True
            
        except Exception as e:
            logger.error(f"키워드 추출 중 오류 발생: {e}")
            self._set_default_keywords()
            return False
    
    def _set_default_keywords(self):
        """기본 키워드 설정"""
        self.keywords = {
            "high": [
                "suicide", "kill myself", "end my life", "want to die", "rather be dead",
                "take my own life", "ending it all", "ending my life", "killing myself", "commit suicide"
            ],
            "medium": [
                "hopeless", "worthless", "no reason to live", "can't go on", "tired of living",
                "what's the point", "better off dead", "no future", "no hope", "give up"
            ],
            "low": [
                "depressed", "depression", "sad", "lonely", "alone", "pain", "suffering",
                "hurt", "failure", "miserable", "lost", "empty"
            ]
        }
        logger.info("기본 키워드를 사용합니다.")

    def _add_default_keywords(self):
        """기존 키워드에 기본 키워드 추가"""
        default_keywords = {
            "high": [
                "suicide", "kill myself", "end my life", "want to die", "rather be dead",
                "take my own life", "ending it all", "ending my life", "killing myself", "commit suicide"
            ],
            "medium": [
                "hopeless", "worthless", "no reason to live", "can't go on", "tired of living",
                "what's the point", "better off dead", "no future", "no hope", "give up"
            ],
            "low": [
                "depressed", "depression", "sad", "lonely", "alone", "pain", "suffering",
                "hurt", "failure", "miserable", "lost", "empty"
            ]
        }
        
        # 각 위험도 레벨에 기본 키워드 추가 (중복 제거)
        for level in ["high", "medium", "low"]:
            existing_keywords_set = set(self.keywords[level])
            for keyword in default_keywords[level]:
                if keyword not in existing_keywords_set:
                    self.keywords[level].append(keyword)

    def train_from_csv(self, csv_path):
        """CSV 데이터로부터 모델 학습"""
        try:
            # 키워드 먼저 추출
            self.extract_keywords_from_csv(csv_path)
            
            # 데이터 로드
            df = pd.read_csv(csv_path)
            
            # 데이터 구조 확인
            if len(df.columns) >= 2:
                text_column = df.columns[0] if df.columns[0] != '' else df.columns[1]
                label_column = df.columns[1]
                
                # 데이터 준비
                X = df[text_column].fillna('')
                
                # 라벨 변환 (필요한 경우)
                if df[label_column].dtype == 'object':
                    suicide_keywords = ['suicide', 'suicidal', 'yes', 'positive', '1', 'true']
                    y = df[label_column].apply(
                        lambda x: 1 if str(x).lower() in suicide_keywords else 0
                    )
                else:
                    y = df[label_column]
                
                # 데이터 전처리 및 학습
                self.vectorizer = TfidfVectorizer(max_features=5000)
                X_transformed = self.vectorizer.fit_transform(X)
                
                # 모델 학습
                self.model = LogisticRegression(C=1.0, max_iter=200)
                self.model.fit(X_transformed, y)
                
                print(f"모델 학습 완료")
                return True
            else:
                print(f"CSV 파일 형식이 올바르지 않음: {csv_path}")
                return False
        except Exception as e:
            print(f"학습 실패: {e}")
            return False
    
    def train(self, csv_path, test_size=0.2, random_state=42):
        """CSV 파일로부터 자살 예측 모델 학습"""
        try:
            # 데이터 로드
            logger.info(f"Loading data from {csv_path}")
            df = pd.read_csv(csv_path)
            
            # 필요한 열만 선택 (텍스트와 라벨)
            # 주의: CSV 파일의 실제 열 이름에 맞게 조정 필요
            text_column = 'text'  # 실제 텍스트 열 이름으로 변경
            label_column = 'class'  # 실제 라벨 열 이름으로 변경
            
            if text_column not in df.columns or label_column not in df.columns:
                # 열 이름 추측
                if len(df.columns) >= 2:
                    text_column = df.columns[0]
                    label_column = df.columns[1]
                    logger.warning(f"Column names not found, using {text_column} and {label_column} instead")
                else:
                    raise ValueError("CSV file does not have enough columns")
            
            logger.info("Preprocessing data...")
            
            # 텍스트 전처리
            df['processed_text'] = df[text_column].fillna('').apply(self.preprocess_text)
            
            # 라벨 변환 (필요한 경우)
            # 예: 'suicide' -> 1, 'non-suicide' -> 0
            if df[label_column].dtype == 'object':
                unique_labels = df[label_column].unique()
                if len(unique_labels) == 2:
                    # 라벨이 이미 바이너리인 경우
                    if all(label in [0, 1] for label in unique_labels):
                        pass
                    else:
                        # 문자열 라벨을 숫자로 변환
                        # 'suicide'와 관련된 라벨을 1로, 나머지를 0으로
                        suicide_keywords = ['suicide', 'suicidal', 'yes', 'positive', '1', 'true']
                        df['label'] = df[label_column].apply(
                            lambda x: 1 if str(x).lower() in suicide_keywords else 0
                        )
                        label_column = 'label'
                        logger.info(f"Converted labels: {df[label_column].value_counts().to_dict()}")
                else:
                    logger.warning(f"Found {len(unique_labels)} unique labels, expected 2")
            
            # 데이터 분할
            X_train, X_test, y_train, y_test = train_test_split(
                df['processed_text'], 
                df[label_column],
                test_size=test_size,
                random_state=random_state,
                stratify=df[label_column]
            )
            
            # 파이프라인 생성
            logger.info("Building and training model...")
            pipeline = Pipeline([
                ('tfidf', TfidfVectorizer(max_features=10000)),
                ('clf', LogisticRegression(C=1.0, class_weight='balanced', max_iter=1000, random_state=random_state))
            ])
            
            # 모델 학습
            pipeline.fit(X_train, y_train)
            
            # 성능 평가
            y_pred = pipeline.predict(X_test)
            accuracy = accuracy_score(y_test, y_pred)
            report = classification_report(y_test, y_pred)
            
            logger.info(f"Model accuracy: {accuracy:.4f}")
            logger.info(f"Classification report:\n{report}")
            
            # 모델 저장
            os.makedirs(os.path.dirname(self.model_path), exist_ok=True)
            joblib.dump(pipeline, self.model_path)
            logger.info(f"Model saved to {self.model_path}")
            
            self.model = pipeline
            return {
                'accuracy': accuracy,
                'report': report
            }
            
        except Exception as e:
            logger.error(f"Error in training model: {e}")
            raise
    
    def predict(self, text, use_model=True):
        """
        텍스트의 자살 위험도 예측
        
        Parameters:
        -----------
        text : str
            분석할 텍스트
        use_model : bool, default=True
            학습된 모델을 사용할지 여부. False인 경우 키워드 기반으로만 예측
            
        Returns:
        --------
        dict
            - risk_level: 위험도 수준 ('none', 'low', 'medium', 'high')
            - risk_score: 위험도 점수 (0-100)
            - keywords_found: 텍스트에서 발견된 키워드 목록
            - model_used: 모델 사용 여부
        """
        if not text or len(text.strip()) == 0:
            return {
                "risk_level": "none",
                "risk_score": 0,
                "keywords_found": [],
                "model_used": False
            }
        
        # 키워드 기반 위험도 계산
        risk_score = self._calculate_keyword_risk(text)
        keywords_found = self._find_keywords_in_text(text)
        
        # 학습된 모델이 있고 모델 사용이 지정된 경우
        model_used = False
        if use_model and self.model is not None:
            try:
                # 모델 예측
                prediction = self.model.predict_proba([text])[0]
                # 모델 예측 확률 (0: 비자살, 1: 자살)
                suicide_prob = prediction[1]
                
                # 모델 점수를 100점 만점으로 변환 (키워드와 결합)
                model_score = suicide_prob * 100
                logger.debug(f"Model prediction: {suicide_prob:.4f}, Model score: {model_score:.2f}")
                
                # 키워드 점수와 모델 점수 가중 결합 (모델 70%, 키워드 30%)
                risk_score = model_score * 0.7 + risk_score * 0.3
                model_used = True
                
                logger.debug(f"Combined risk score: {risk_score:.2f}")
            except Exception as e:
                logger.error(f"Model prediction failed: {e}")
                # 모델 예측 실패 시 키워드 점수만 사용
        
        # 최종 위험도 레벨 결정
        risk_level = self._interpret_risk_score(risk_score)
        
        return {
            "risk_level": risk_level,
            "risk_score": round(risk_score, 2),
            "keywords_found": keywords_found,
            "model_used": model_used
        }
    
    def _calculate_keyword_risk(self, text):
        """키워드 기반 위험도 계산"""
        text_lower = text.lower()
        words = set(re.findall(r'\b\w+\b', text_lower))  # Extract individual words for exact matching
        
        # 결과 저장할 변수들
        high_matches = []
        medium_matches = []
        low_matches = []
        
        # 카테고리별 키워드 검사
        for keyword in self.keywords["high"]:
            if keyword.lower() in text_lower or keyword.lower() in words:
                high_matches.append(keyword)
        
        for keyword in self.keywords["medium"]:
            if keyword.lower() in text_lower or keyword.lower() in words:
                medium_matches.append(keyword)
        
        for keyword in self.keywords["low"]:
            if keyword.lower() in text_lower or keyword.lower() in words:
                low_matches.append(keyword)
        
        # 위험도 점수 계산 (100점 만점)
        high_score = min(len(high_matches) * 25, 70)  # 각 고위험 키워드당 25점, 최대 70점
        medium_score = min(len(medium_matches) * 10, 20)  # 각 중위험 키워드당 10점, 최대 20점
        low_score = min(len(low_matches) * 5, 10)  # 각 저위험 키워드당 5점, 최대 10점
        
        # 복합 키워드 패턴 검사 (n-gram 패턴)
        phrase_patterns = [
            (r"(want|wish) to (die|end|kill myself)", 30),  # 특정 표현 추가 점수
            (r"plan to (suicide|kill myself|end my life)", 40),
            (r"no (reason|point) (in|to) (living|life)", 25),
            (r"can'?t (take|bear|handle) (it|this) anymore", 20)
        ]
        
        pattern_score = 0
        for pattern, score in phrase_patterns:
            if re.search(pattern, text_lower):
                pattern_score += score
        
        # 최종 점수 계산 (최대 100점)
        total_score = min(high_score + medium_score + low_score + pattern_score, 100)
        
        logger.debug(f"Risk score calculation - High: {high_score}, Medium: {medium_score}, Low: {low_score}, Phrases: {pattern_score}")
        
        return total_score
    
    def _find_keywords_in_text(self, text):
        """텍스트에서 발견된 키워드 목록 반환"""
        text_lower = text.lower()
        words = set(re.findall(r'\b\w+\b', text_lower))  # 개별 단어 추출
        found_keywords = []
        
        # 모든 카테고리의 키워드 검사
        for category, weight in [("high", "High"), ("medium", "Medium"), ("low", "Low")]:
            for keyword in self.keywords[category]:
                keyword_lower = keyword.lower()
                # 개별 단어 또는 구문으로 검사
                if keyword_lower in text_lower or (len(keyword_lower.split()) == 1 and keyword_lower in words):
                    found_keywords.append({
                        "word": keyword,
                        "category": weight
                    })
        
        # 복합 키워드 패턴 검사
        phrase_patterns = [
            (r"(want|wish) to (die|end|kill myself)", "High", "Want to die"),
            (r"plan to (suicide|kill myself|end my life)", "High", "Suicide plan"),
            (r"no (reason|point) (in|to) (living|life)", "Medium", "No reason to live"),
            (r"can'?t (take|bear|handle) (it|this) anymore", "Medium", "Can't take it anymore")
        ]
        
        for pattern, category, description in phrase_patterns:
            if re.search(pattern, text_lower):
                found_keywords.append({
                    "word": description,
                    "category": category
                })
        
        return found_keywords
    
    def _interpret_risk_score(self, score):
        """위험도 점수 해석"""
        if score >= 65:
            return "high"
        elif score >= 35:
            return "medium"
        elif score >= 15:
            return "low"
        else:
            return "none"

# 테스트 코드
if __name__ == "__main__":
    predictor = SuicidePredictor()
    
    # 모델이 없으면 학습
    if predictor.model is None:
        # 파일 우선순위: 1. 샘플 파일 2. 압축 파일 3. 전체 파일
        sample_csv_path = 'Suicide_Detection_sample.csv'
        compressed_csv_path = 'Suicide_Detection.csv.gz'
        full_csv_path = 'Suicide_Detection.csv'
        
        if os.path.exists(sample_csv_path):
            logger.info(f"Using sample CSV file: {sample_csv_path}")
            csv_path = sample_csv_path
            df = pd.read_csv(csv_path)
        elif os.path.exists(compressed_csv_path):
            logger.info(f"Using compressed CSV file: {compressed_csv_path}")
            # 압축 파일 읽기
            with gzip.open(compressed_csv_path, 'rt') as f:
                df = pd.read_csv(f)
            csv_path = None  # df를 직접 전달하기 위해 경로는 None으로 설정
        elif os.path.exists(full_csv_path):
            logger.info(f"Using full CSV file: {full_csv_path}")
            csv_path = full_csv_path
            df = pd.read_csv(csv_path)
        else:
            raise FileNotFoundError("No suicide detection CSV file found. Please provide either Suicide_Detection.csv, Suicide_Detection.csv.gz, or Suicide_Detection_sample.csv")
        
        # csv_path가 있으면 경로로 학습, 없으면 df로 학습
        if csv_path:
            predictor.train_from_csv(csv_path)
        else:
            # train 메소드를 DataFrame을 직접 받을 수 있도록 수정 필요
            # 임시로 DataFrame을 CSV로 저장하고 사용
            temp_csv_path = 'temp_data.csv'
            df.to_csv(temp_csv_path, index=False)
            predictor.train_from_csv(temp_csv_path)
            # 임시 파일 삭제
            if os.path.exists(temp_csv_path):
                os.remove(temp_csv_path)
        
    # 테스트 텍스트로 예측
    test_text = "I don't want to live anymore. Everything feels so hopeless."
    result = predictor.predict(test_text)
    print(f"Risk level: {result['risk_level']}")
    print(f"Risk score: {result['risk_score']}")
    print(f"Keywords found: {result['keywords_found']}") 