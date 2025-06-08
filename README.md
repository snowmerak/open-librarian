# Open Librarian

Open Librarian은 AI 기반 하이브리드 검색 시스템으로, 벡터 검색과 키워드 검색을 결합하여 정확하고 의미론적으로 관련성 높은 검색 결과를 제공하는 지능형 문서 검색 서비스입니다.

## 시스템 아키텍처

### 핵심 구성 요소

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Go API Server │    │   OpenSearch    │    │     Qdrant      │
│                 │    │ (Keyword Search)│    │ (Vector Search) │
│ - HTTP Handlers │◄──►│ - Full-text     │    │ - Embeddings    │
│ - Business Logic│    │ - Aggregations  │    │ - Similarity    │
│ - Data Pipeline │    │ - Analytics     │    │ - Collections   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │
         ▼
┌─────────────────┐
│   Ollama LLM    │
│                 │
│ - Text Gen.     │
│ - Embeddings    │
│ - Summarization │
└─────────────────┘
```

### 기술 스택

- **Backend**: Go 1.24+ with Chi router
- **Vector Database**: Qdrant (768-dimension embeddings)
- **Search Engine**: OpenSearch 2.13.0
- **AI/LLM**: Ollama (로컬 LLM 서버)
- **Language Detection**: Lingua-Go
- **Containerization**: Docker & Docker Compose

## 핵심 기능 및 원리

### 1. 하이브리드 검색 시스템

Open Librarian은 두 가지 검색 방식을 결합합니다:

#### 벡터 검색 (Semantic Search)
- **원리**: 문서와 쿼리를 고차원 벡터로 변환하여 의미적 유사성 측정
- **구현**: Ollama를 통해 768차원 임베딩 생성 → Qdrant에서 벡터 유사도 검색
- **장점**: 동의어, 유사 개념, 문맥적 의미 이해

#### 키워드 검색 (Lexical Search)
- **원리**: 전통적인 텍스트 매칭 기반 검색
- **구현**: OpenSearch의 BM25 알고리즘 활용
- **장점**: 정확한 용어 매칭, 빠른 검색 속도

#### 점수 결합 알고리즘
```go
// 하이브리드 점수 계산 (벡터 60%, 키워드 40% 가중평균)
if 벡터_결과 && 키워드_결과 {
    최종점수 = (0.6 * 벡터점수) + (0.4 * 키워드점수)
    출처 = "hybrid"
} else {
    최종점수 = 개별점수
    출처 = "vector" 또는 "keyword"
}
```

### 2. AI 기반 문서 처리 파이프라인

#### 문서 인덱싱 과정
```
원본 문서 입력
    ↓
언어 감지 (Lingua-Go)
    ↓
AI 요약 생성 (Ollama)
    ↓
키워드 추출 (Ollama)
    ↓
임베딩 생성 (Ollama)
    ↓
병렬 저장: OpenSearch (텍스트) + Qdrant (벡터)
```

#### 검색 및 답변 생성 과정
```
사용자 쿼리
    ↓
언어 감지 & 쿼리 임베딩 생성
    ↓
병렬 검색: 벡터 검색 + 키워드 검색
    ↓
결과 결합 & 중복 제거 & 점수 정렬
    ↓
상위 결과를 컨텍스트로 AI 답변 생성
```

### 3. 다국어 지원

- **지원 언어**: 한국어, 영어, 일본어, 중국어 등
- **언어별 최적화**: 검색 시 언어 필터링으로 정확도 향상
- **자동 언어 감지**: 문서와 쿼리의 언어를 자동으로 식별

## API 엔드포인트

### 문서 관리
- `POST /api/v1/articles` - 새 문서 추가
- `GET /api/v1/articles/{id}` - 특정 문서 조회

### 검색
- `POST /api/v1/search` - AI 답변이 포함된 하이브리드 검색
- `GET /api/v1/search/keyword?q={query}` - 키워드 전용 검색

### 유틸리티
- `GET /health` - 시스템 상태 확인
- `GET /api/v1/languages` - 지원 언어 목록

## 로컬 개발 환경 설정

### 필수 요구사항

- Docker 및 Docker Compose
- Ollama (로컬 LLM 서버)
- Go 1.24+

### 1. 인프라 서비스 시작

```bash
# OpenSearch, Qdrant, API 서버 실행
./scripts/setup-services.sh

# Rebuild API 서버 (코드 변경 시)
./scripts/rebuild-server.sh

# 서비스 상태 확인
curl http://localhost:8080/health
```

### 2. Ollama 설정

```bash
# Ollama 설치 (macOS)
brew install ollama

# Ollama 서버 시작
ollama serve

# 필요한 모델 다운로드 (별도 터미널)
ollama pull gemma3:12b  # 또는 원하는 모델
```

### 3. 테스트

#### 문서 추가
```bash
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Content-Type: application/json" \
  -d '{
    "title": "AI와 머신러닝의 차이점",
    "content": "인공지능(AI)는 인간의 지능을 모방하는 기술이고, 머신러닝은 AI의 하위 분야로...",
    "author": "홍길동"
  }'
```

#### 검색 테스트
```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "AI와 머신러닝의 관계는?",
    "size": 5
  }'
```

### 4. 접근 정보

- **API 서버**: http://localhost:8080
- **OpenSearch**: http://localhost:9200
- **OpenSearch Dashboards**: http://localhost:5601
- **Qdrant Dashboard**: http://localhost:6333/dashboard

## 시스템 특징

### 성능 최적화
- **병렬 검색**: 벡터 검색과 키워드 검색을 동시 실행
- **결과 캐싱**: 중복 계산 방지를 위한 결과 캐싱
- **배치 처리**: 대량 문서 처리 시 배치 인덱싱

### 확장성
- **수평 확장**: 각 컴포넌트 독립적 스케일링 가능
- **언어 확장**: 새로운 언어 지원 용이
- **모델 교체**: Ollama를 통한 다양한 LLM 모델 사용 가능

### 신뢰성
- **헬스 체크**: 모든 서비스의 상태 모니터링
- **장애 복구**: 일부 서비스 장애 시에도 제한적 기능 제공
- **데이터 일관성**: OpenSearch와 Qdrant 간 데이터 동기화

## 개발 및 배포

### 로컬 개발
```bash
# 의존성 설치
go mod download

# 개발 서버 실행
go run cmd/server/main.go
```

### Docker 빌드
```bash
# 이미지 빌드
docker build -t open-librarian .

# 컨테이너 실행
docker run -p 8080:8080 open-librarian
```

### 환경 변수

| 변수명 | 기본값 | 설명 |
|--------|--------|------|
| PORT | 8080 | API 서버 포트 |
| OPENSEARCH_URL | http://localhost:9200 | OpenSearch 연결 URL |
| OLLAMA_URL | http://localhost:11434 | Ollama 서버 URL |
| QDRANT_HOST | localhost | Qdrant 호스트 |
| QDRANT_PORT | 6334 | Qdrant gRPC 포트 |
