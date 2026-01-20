// WebSocket을 사용한 스트리밍 검색 처리
// Global counter for generating unique IDs
let messageIdCounter = 0;

// 메인 검색 함수 (WebSocket 스트리밍)
async function handleStreamSearch() {
    const input = document.getElementById('search-input');
    const query = input.value.trim();
    if (!query) return;

    // 입력창 초기화
    input.value = '';

    const chatContainer = document.getElementById('chat-container');
    const welcomeMessage = document.getElementById('welcome-message');
    
    // 환영 메시지 숨기기
    if (welcomeMessage) {
        welcomeMessage.style.display = 'none';
    }

    // 1. 사용자 메시지 추가
    appendUserMessage(query);
    scrollToBottom();

    // 2. AI 응답 준비 (스피너 표시)
    const messageIds = appendAiMessage();
    const { contentId, statusId, sourcesId } = messageIds;
    scrollToBottom();

    // 검색 기록 저장 (비동기로 실행)
    saveSearchToHistory(query).catch(console.error);

    try {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/search/ws`;
        const ws = new WebSocket(wsUrl);
        
        // Agentic Search는 한 번에 답을 주거나 상태를 줌
        let fullAnswer = '';

        ws.onopen = function() {
            // 검색 요청 전송
            ws.send(JSON.stringify({
                query: query,
                size: 5 // 기본값
            }));
        };

        ws.onmessage = function(event) {
            try {
                const message = JSON.parse(event.data);
                
                switch(message.type) {
                    case 'status':
                        updateAiStatus(statusId, message.data);
                        scrollToBottom();
                        break;
                    case 'sources':
                        updateAiSources(sourcesId, message.data);
                        scrollToBottom();
                        break;
                    case 'answer':
                        // Agentic Search에서는 완성된 답변이 옴
                        fullAnswer = message.data;
                        updateAiContent(contentId, fullAnswer);
                        scrollToBottom();
                        break;
                    case 'error':
                        updateAiStatus(statusId, `<span class="text-red-500">Error: ${message.data}</span>`);
                        break;
                    case 'done':
                        updateAiStatus(statusId, ''); // 상태 메시지 제거 (완료)
                        ws.close();
                        break;
                    default:
                        console.log('Unknown message type:', message.type);
                }
            } catch (error) {
                console.error('Error processing WebSocket message:', error);
            }
        };

        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
            updateAiStatus(statusId, '<span class="text-red-500">Connection error</span>');
        };

    } catch (error) {
        console.error('Stream search error:', error);
        updateAiStatus(statusId, `<span class="text-red-500">${error.message}</span>`);
    }
}

// 사용자 메시지 추가
function appendUserMessage(text) {
    const container = document.getElementById('chat-container');
    const msgDiv = document.createElement('div');
    msgDiv.className = 'flex w-full mt-6 space-x-3 max-w-4xl mx-auto justify-end';
    
    msgDiv.innerHTML = `
        <div class="user-bubble text-lg">
            ${escapeHtml(text)}
        </div>
    `;
    container.appendChild(msgDiv);
}

// AI 메시지 컨테이너 추가
function appendAiMessage() {
    const container = document.getElementById('chat-container');
    const id = messageIdCounter++;
    const contentId = `ai-content-${id}`;
    const statusId = `ai-status-${id}`;
    const sourcesId = `ai-sources-${id}`;
    
    const msgDiv = document.createElement('div');
    msgDiv.className = 'flex w-full mt-6 space-x-3 max-w-4xl mx-auto ai-bubble-container';
    
    msgDiv.innerHTML = `
        <div class="ai-avatar">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>
        </div>
        <div class="min-w-0 flex-1 w-full">
            <div class="ai-bubble">
                <div id="${statusId}" class="mb-3 text-sm text-slate-500 flex items-center">
                    <div class="thinking-dots">
                        <div class="thinking-dot"></div>
                        <div class="thinking-dot"></div>
                        <div class="thinking-dot"></div>
                    </div>
                </div>
                <div id="${sourcesId}" class="flex flex-wrap gap-2 mb-3"></div>
                <div id="${contentId}" class="markdown-content text-slate-800"></div>
            </div>
        </div>
    `;
    
    container.appendChild(msgDiv);
    return { contentId, statusId, sourcesId };
}

// AI 콘텐츠 업데이트 (Markdown 렌더링)
function updateAiContent(elementId, text) {
    const element = document.getElementById(elementId);
    if (!element) return;
    
    // marked 라이브러리를 사용하여 Markdown 렌더링
    element.innerHTML = marked.parse(text);
}

// AI 상태 메시지 업데이트
function updateAiStatus(elementId, statusHtml) {
    const element = document.getElementById(elementId);
    if (!element) return;
    
    if (statusHtml === '' || statusHtml === null) {
        element.style.display = 'none';
    } else {
        element.style.display = 'block'; // Ensure visible
        // 텍스트만 왔을 경우 스피너 추가, HTML 태그가 있으면 그대로 사용
        if (!statusHtml.includes('<')) {
            element.innerHTML = `
                <div class="thinking-dots mr-2 inline-block">
                    <div class="thinking-dot"></div>
                    <div class="thinking-dot"></div>
                    <div class="thinking-dot"></div>
                </div>
                <span>${statusHtml}</span>
            `;
        } else {
            element.innerHTML = statusHtml;
        }
    }
}

// 소스(출처) 업데이트
function updateAiSources(elementId, sources) {
    const element = document.getElementById(elementId);
    if (!element || !sources || sources.length === 0) return;
    
    let html = '';
    sources.forEach((source, index) => {
        // 점수와 제목 표시 (상대 경로 문제 해결을 위해 encodeURIComponent 사용 가능)
        // const score = Math.round(source.score * 100);
        html += `
            <div class="source-chip" title="${escapeHtml(source.article.title)}">
                <span class="font-semibold mr-1">${index + 1}.</span>
                <span class="truncate max-w-[150px]">${escapeHtml(source.article.title)}</span>
            </div>
        `;
    });
    
    element.innerHTML = html;
}

// 자동 스크롤
function scrollToBottom() {
    const chatContainer = document.getElementById('chat-container');
    if (chatContainer) chatContainer.scrollTop = chatContainer.scrollHeight;
}

// 캐시 사용 안함 (Chat Interface에서는 보통 대화형이므로)
async function getCachedSearchResult(query) { return null; }
async function saveSearchToHistory(query) { /* Implement history saving */ }

// HTML 이스케이프 유틸리티
function escapeHtml(text) {
    if (!text) return '';
    return text
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// 검색 엔터키 처리
document.addEventListener('DOMContentLoaded', function() {
    const input = document.getElementById('search-input');
    if (input) {
        input.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                handleStreamSearch();
            }
        });
    }
});

// 전역 함수로 노출
window.handleSearch = handleStreamSearch;