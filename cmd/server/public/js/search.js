// WebSocket을 사용한 스트리밍 검색 처리 (Chat Style)

// Global counter for generating unique IDs
let messageIdCounter = 0;
let currentSessionId = null;

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

    try {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/search/ws`;
        const ws = new WebSocket(wsUrl);
        
        let fullAnswer = '';

        ws.onopen = function() {
            // 검색 요청 전송
            ws.send(JSON.stringify({
                query: query,
                size: 5, // 기본값
                session_id: currentSessionId || "" // 세션 ID 전송
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
                        fullAnswer = message.data;
                        updateAiContent(contentId, fullAnswer);
                        scrollToBottom();
                        break;
                    case 'error':
                        updateAiStatus(statusId, `<span class="text-red-500">Error: ${message.data}</span>`);
                        break;
                    case 'done':
                        updateAiStatus(statusId, ''); // 상태 메시지 제거 (완료)
                        
                        // 세션 ID 업데이트 check
                        if (message.data && message.data.session_id) {
                            currentSessionId = message.data.session_id;
                            // 히스토리 목록 갱신 요청
                            if (typeof updateHistoryDisplay === 'function') {
                                updateHistoryDisplay();
                            }
                        }
                        
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

// 세션 변경 처리
function setCurrentSession(sessionId) {
    currentSessionId = sessionId;
}

// 채팅 화면 초기화
function clearChatInterface() {
    const chatContainer = document.getElementById('chat-container');
    const welcomeMessage = document.getElementById('welcome-message');
    
    // 환영 메시지 제외하고 모두 제거
    if (chatContainer) {
        // chatContainer의 자식 중 welcome-message가 아닌 것들 제거
        Array.from(chatContainer.children).forEach(child => {
            if (child.id !== 'welcome-message') {
                chatContainer.removeChild(child);
            }
        });
        
        if (welcomeMessage) {
            welcomeMessage.style.display = 'block';
        }
    }
    
    currentSessionId = null;
}

// 정적 AI 메시지 렌더링 (히스토리 로드용)
function renderStaticAiMessage(content, sources) {
    const container = document.getElementById('chat-container');
    const id = messageIdCounter++;
    const contentId = `ai-content-${id}`;
    const sourcesId = `ai-sources-${id}`;
    
    const msgDiv = document.createElement('div');
    msgDiv.className = 'flex w-full mt-6 space-x-3 max-w-4xl mx-auto ai-bubble-container';
    
    // 소스 HTML 생성
    let sourcesHtml = '';
    if (sources && sources.length > 0) {
        sources.forEach((source, index) => {
             // 소스 구조가 유동적일 수 있으므로 체크 (DB 저장 구조 vs 실시간 구조)
             // DB Saved: source might be simpler or same
             const title = source.article ? source.article.title : (source.Title || "Source");
             sourcesHtml += `
                <div class="source-chip" title="${escapeHtml(title)}">
                    <span class="font-semibold mr-1">${index + 1}.</span>
                    <span class="truncate max-w-[150px]">${escapeHtml(title)}</span>
                </div>
            `;
        });
    }

    msgDiv.innerHTML = `
        <div class="ai-avatar">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>
        </div>
        <div class="min-w-0 flex-1 w-full">
            <div class="ai-bubble">
                <div id="${sourcesId}" class="flex flex-wrap gap-2 mb-3">${sourcesHtml}</div>
                <div id="${contentId}" class="markdown-content text-slate-800"></div>
            </div>
        </div>
    `;
    
    container.appendChild(msgDiv);
    
    // Markdown 렌더링
    updateAiContent(contentId, content);
}

// 사용자 메시지 추가
function appendUserMessage(text) {
    const container = document.getElementById('chat-container');
    const msgDiv = document.createElement('div');
    msgDiv.className = 'flex w-full mt-6 space-x-3 max-w-4xl mx-auto justify-end';
    
    // safe escape inside
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
    if (typeof marked !== 'undefined') {
        element.innerHTML = marked.parse(text);
    } else {
        element.textContent = text;
    }
}

// AI 상태 메시지 업데이트
function updateAiStatus(elementId, statusHtml) {
    const element = document.getElementById(elementId);
    if (!element) return;
    
    if (statusHtml === '' || statusHtml === null) {
        element.style.display = 'none';
    } else {
        element.style.display = 'block'; 
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

// 히스토리에서 검색 호출용
function searchFromHistory(query) {
    const input = document.getElementById('search-input');
    if (input) {
        input.value = query;
        handleStreamSearch();
    }
}

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
window.searchFromHistory = null; // Will be defined in ui.js properly or handled via loadChatSession
window.setCurrentSession = setCurrentSession;
window.clearChatInterface = clearChatInterface;
window.renderStaticAiMessage = renderStaticAiMessage;
window.appendUserMessage = appendUserMessage; // Export for external use