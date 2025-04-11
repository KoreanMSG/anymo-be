import os
import google.generativeai as genai
from dotenv import load_dotenv
import logging

# 환경 설정
load_dotenv()
logging.basicConfig(level=logging.INFO, 
                    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Gemini API 설정
try:
    GEMINI_API_KEY = os.getenv("GEMINI_API_KEY")
    if not GEMINI_API_KEY:
        raise ValueError("GEMINI_API_KEY not found in environment variables")
    genai.configure(api_key=GEMINI_API_KEY)
except Exception as e:
    logger.error(f"Failed to configure Gemini API: {e}")
    raise

class ConversationProcessor:
    def __init__(self):
        try:
            self.model = genai.GenerativeModel('gemini-1.5-pro')
            logger.info("Gemini model initialized successfully")
        except Exception as e:
            logger.error(f"Failed to initialize Gemini model: {e}")
            raise

    def process_conversation(self, text):
        """
        텍스트를 입력받아 Gemini를 이용해 의사와 환자의 대화로 분리합니다.
        
        Args:
            text (str): 분석할 대화 텍스트
            
        Returns:
            dict: 다음 형식의 결과를 반환합니다.
                {
                    'processed_text': '의사: 안녕하세요, 어떻게 지내세요?@@환자: 요즘 잠을 잘 못자요.',
                    'starts_with_doctor': True
                }
        """
        try:
            # Gemini 프롬프트 구성
            prompt = f"""
            아래 텍스트는 의사와 환자 사이의 대화입니다. 
            이 대화를 분석하여 각 발화자(의사/환자)를 식별하고, 다음 형식으로 출력해주세요:
            
            출력 형식:
            1. 대화가 의사로 시작하는지 여부(true/false)
            2. 대화 내용을 "@@" 구분자로 연결 (발화자 표시 없이 대화 내용만)
            
            입력 텍스트:
            {text}
            
            응답은 JSON 형식으로 해주세요:
            {{
                "starts_with_doctor": true/false,
                "conversation": "첫번째 발화@@두번째 발화@@세번째 발화"
            }}
            """
            
            response = self.model.generate_content(prompt)
            response_text = response.text
            
            # 응답에서 JSON 부분 추출
            import json
            import re
            
            # JSON 형식 추출
            json_match = re.search(r'```json\n(.*?)\n```', response_text, re.DOTALL)
            if json_match:
                json_str = json_match.group(1)
            else:
                json_str = response_text
                
            # 불필요한 문자 제거
            json_str = re.sub(r'```json|```', '', json_str).strip()
            
            # JSON 파싱
            try:
                result = json.loads(json_str)
                logger.info("Successfully processed conversation with Gemini")
                return {
                    'processed_text': result['conversation'],
                    'starts_with_doctor': result['starts_with_doctor']
                }
            except json.JSONDecodeError:
                # JSON 파싱 실패시 직접 분석
                logger.warning("Failed to parse JSON from Gemini response, attempting manual parsing")
                lines = response_text.strip().split('\n')
                starts_with_doctor = any('true' in line.lower() and 'starts_with_doctor' in line.lower() for line in lines)
                conversation_lines = []
                
                for line in lines:
                    if '@@' in line:
                        conversation_lines.append(line)
                
                processed_text = '@@'.join([l.strip() for l in conversation_lines]) if conversation_lines else text
                
                return {
                    'processed_text': processed_text,
                    'starts_with_doctor': starts_with_doctor
                }
                
        except Exception as e:
            logger.error(f"Error in processing conversation: {e}")
            # 오류 발생 시 기본값 반환
            return {
                'processed_text': text,
                'starts_with_doctor': True  # 기본값으로 True 설정
            }

# 테스트 코드
if __name__ == "__main__":
    processor = ConversationProcessor()
    test_text = "안녕하세요, 어떻게 지내세요? 요즘 잠을 못자고 있어요. 언제부터 그런 증상이 있었나요? 한 2주 정도 됐어요."
    result = processor.process_conversation(test_text)
    print(f"Processed text: {result['processed_text']}")
    print(f"Starts with doctor: {result['starts_with_doctor']}") 