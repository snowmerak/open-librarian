# Open Librarian 🔍

An intelligent search service that provides AI-powered answers based on stored articles with real-time communication and user authentication.

## 🚀 Key Features

| **Component** | **Technology** | **Version** | **Description** |
|---------------|----------------|-------------|-----------------|
| **Language Detection** | Lingua-Go | v1.4+ | 8개 언어 감지 |
| **Frontend** | Vanilla JS + Tailwind | Latest | 현대적 웹 UI + 다국어 지원 |
| **Real-time** | WebSocket + SSE | Native | 실시간 통신 및 진행률 추적 |
| **Container** | Docker + Compose | Latest | 컨테이너화 배포 |
| **Authentication** | JWT + Argon2 | Latest | 보안 인증 시스템 |
| **Database** | MongoDB + IndexedDB | Latest | 사용자 데이터 및 클라이언트 저장소 |

## 🌟 New Features

### 🔐 User Authentication & Authorization
- **Secure Registration/Login**: JWT-based authentication with Argon2 password hashing
- **User Management**: Profile management, password changes, account deletion
- **Role-based Access**: Users can only manage their own uploaded articles
- **Session Management**: Token refresh and automatic logout on expiration

### 🌐 Real-time Communication
- **WebSocket Integration**: Real-time search results and upload progress
- **Live Progress Tracking**: Visual progress indicators for article processing
- **Streaming Responses**: Real-time AI answer generation
- **Connection Resilience**: Automatic fallback to HTTP on WebSocket failure

### 🌍 Multilingual Support
- **i18n System**: Complete internationalization framework
- **Supported Languages**: English, Korean, Chinese, Japanese, Spanish
- **Dynamic Language Switching**: Runtime language changes without page reload
- **Persistent Preferences**: Language settings saved locally

### 📊 Enhanced Article Management
- **Bulk Upload**: JSONL file processing with progress tracking
- **Article Ownership**: Users can delete their own articles
- **Metadata Support**: Author, creation date, original URL tracking
- **Real-time Validation**: Live feedback during upload process

### 🔗 External API Access
- **Agent-friendly Endpoints**: Read-only APIs for external systems
- **Rate Limiting**: Protection against abuse
- **Simplified Responses**: Optimized data formats for agents
- **Public Access**: No authentication required for read operations

## 🏗 Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Backend       │    │   Databases     │
│                 │    │                 │    │                 │
│ • Vanilla JS    │◄──►│ • Go (Chi)      │◄──►│ • MongoDB       │
│ • Tailwind CSS  │    │ • WebSocket     │    │ • OpenSearch    │
│ • IndexedDB     │    │ • JWT Auth      │    │ • Qdrant        │
│ • i18n Support  │    │ • Rate Limiting │    │ • Vector Store  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- Node.js (for development)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/snowmerak/open-librarian.git
cd open-librarian
```

2. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. **Start services with Docker**
```bash
docker-compose up -d
```

4. **Run the application**
```bash
go run cmd/server/main.go
```

5. **Access the application**
- Web UI: http://localhost:8080
- API Documentation: http://localhost:8080/swagger/
- Health Check: http://localhost:8080/health

## 🔧 Configuration

### Environment Variables

```bash
# Server Configuration
PORT=8080
JWT_SECRET=your-super-secret-jwt-key

# Database Connections
MONGODB_URI=mongodb://localhost:27017/open_librarian
OPENSEARCH_URL=http://localhost:9200
QDRANT_HOST=localhost
QDRANT_PORT=6334

# AI Service
OLLAMA_URL=http://localhost:11434
```

### MongoDB Setup

```bash
# Create user and database
mongosh
use open_librarian
db.createUser({
  user: "librarian",
  pwd: "your-password",
  roles: ["readWrite"]
})
```

## 📖 API Documentation

### Authentication Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/users/` | User registration |
| `POST` | `/api/v1/users/auth` | User login |
| `POST` | `/api/v1/users/refresh` | Token refresh |
| `GET` | `/api/v1/users/me` | Get current user |
| `PUT` | `/api/v1/users/{id}` | Update user profile |
| `DELETE` | `/api/v1/users/{id}` | Delete user account |

### Article Management

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `POST` | `/api/v1/articles` | Add single article | ✅ |
| `POST` | `/api/v1/articles/bulk` | Bulk article upload | ✅ |
| `DELETE` | `/api/v1/articles/{id}` | Delete article | ✅ (Owner only) |
| `GET` | `/api/v1/articles/{id}` | Get article details | ❌ |

### Search Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/search` | Hybrid search with AI answers |
| `GET` | `/api/v1/search/keyword` | Keyword-only search |
| `WS` | `/api/v1/search/ws` | WebSocket real-time search |

### External Agent APIs

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/external/articles` | List articles (public) |
| `GET` | `/api/v1/external/articles/{id}` | Get article (public) |
| `GET` | `/api/v1/external/search` | Search articles (public) |

## 🔌 WebSocket Usage

### Real-time Search
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/search/ws');

ws.onopen = () => {
    ws.send(JSON.stringify({
        query: "Your search query",
        size: 10
    }));
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    switch(message.type) {
        case 'status':
            console.log('Status:', message.data);
            break;
        case 'sources':
            console.log('Sources:', message.data);
            break;
        case 'answer':
            console.log('AI Answer chunk:', message.data);
            break;
        case 'done':
            console.log('Search completed');
            break;
    }
};
```

### Article Upload with Progress
```javascript
const token = 'your-jwt-token';
const ws = new WebSocket(`ws://localhost:8080/api/v1/articles/ws?token=${token}`);

ws.onopen = () => {
    ws.send(JSON.stringify({
        title: "Article Title",
        content: "Article content...",
        author: "Author Name"
    }));
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    switch(message.type) {
        case 'progress':
            updateProgressBar(message.data.percent);
            break;
        case 'success':
            console.log('Upload successful');
            break;
    }
};
```

## 🛡 Security Features

- **Password Security**: Argon2 hashing with salt
- **JWT Authentication**: Secure token-based auth
- **CORS Protection**: Configurable cross-origin policies
- **Rate Limiting**: API abuse prevention
- **Input Validation**: Comprehensive request validation
- **SQL Injection Prevention**: MongoDB query sanitization

## 🌐 Internationalization

The system supports multiple languages with automatic detection and manual selection:

- **English** (en) - Default
- **Korean** (ko) - 한국어
- **Chinese** (zh) - 中文
- **Japanese** (ja) - 日本語
- **Spanish** (es) - Español

Language preferences are automatically saved and restored across sessions.

## 📊 Monitoring & Logging

- **Health Checks**: Comprehensive service health monitoring
- **Request Logging**: Detailed HTTP request/response logging
- **Error Tracking**: Structured error reporting
- **Performance Metrics**: Response time and throughput monitoring

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [OpenSearch](https://opensearch.org/) for full-text search capabilities
- [Qdrant](https://qdrant.tech/) for vector similarity search
- [Ollama](https://ollama.ai/) for local LLM inference
- [MongoDB](https://www.mongodb.com/) for user data storage
- [Go Chi](https://go-chi.io/) for HTTP routing framework
