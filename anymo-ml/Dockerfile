FROM python:3.10-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 필요한 디렉토리 생성
RUN mkdir -p data models

# NLTK 리소스 다운로드
RUN python -c "import nltk; nltk.download('punkt'); nltk.download('stopwords'); nltk.download('wordnet')"

# 파일 복사 (CSV 파일 포함)
COPY . .

# CSV 파일이 data 디렉토리에 있는지 확인하고 없으면 자동으로 이동
RUN if [ -f "Suicide_Detection.csv" ] && [ ! -f "data/Suicide_Detection.csv" ]; then \
    mv Suicide_Detection.csv data/; \
    fi

# 환경 변수 설정
ENV PORT=8000
ENV CSV_PATH=data/Suicide_Detection.csv

# 포트 노출
EXPOSE 8000

# 서버 실행
CMD ["python", "api.py"] 