// WebSocket을 사용한 스트리밍 검색 처리
async function handleStreamSearch() {
    const input = document.getElementById('search-input');
    const query = input.value.trim();
    if (!query) return;

    // 먼저 캐시된 결과 확인
    const cachedResult = await getCachedSearchResult(query);
    if (cachedResult) {
        displayCachedResult(query, cachedResult);
        input.value = '';
        return;
    }

    const searchResults = document.getElementById('search-results');
    const welcomeMessage = document.getElementById('welcome-message');
    
    // 환영 메시지 숨기기
    if (welcomeMessage) {
        welcomeMessage.style.display = 'none';
    }

    // 검색 기록에 저장
    await saveSearchToHistory(query);

    // 결과 컨테이너 생성
    const resultDiv = document.createElement('div');
    resultDiv.id = 'streaming-result';
    resultDiv.className = 'search-result';
    
    // 초기 UI 설정
    resultDiv.innerHTML = `
        <div class="mb-4">
            <h3 class="text-lg font-semibold text-slate-800 mb-3">"${escapeHtml(query)}" ${t('answerFor')}</h3>
            <div id="status-indicator" class="flex items-center mb-3">
                <div class="spinner mr-2"></div>
                <span class="text-sm text-slate-600">${t('searchingInProgress')}</span>
            </div>
            <div id="answer-content" class="markdown-content text-slate-700 min-h-[50px] border-l-4 border-blue-500 pl-4 bg-blue-50 rounded-r-lg"></div>
        </div>
        <div id="sources-container" class="mt-6 pt-6 border-t border-slate-200" style="display: none;">
            <h4 class="font-semibold text-slate-700 mb-4">${t('references')}</h4>
            <div id="sources-list" class="grid gap-3"></div>
        </div>
    `;
    
    searchResults.querySelector('.max-w-4xl').appendChild(resultDiv);
    resultDiv.scrollIntoView({ behavior: 'smooth', block: 'start' });

    try {
        // WebSocket 연결 생성
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/search/ws`;
        const ws = new WebSocket(wsUrl);
        
        let fullAnswer = '';
        let sources = [];

        ws.onopen = function() {
            console.log('WebSocket connected');
            // 검색 요청 전송
            ws.send(JSON.stringify({
                query: query,
                size: 5
            }));
        };

        ws.onmessage = function(event) {
            try {
                const message = JSON.parse(event.data);
                
                switch(message.type) {
                    case 'status':
                        updateStatus(message.data);
                        break;
                    case 'sources':
                        sources = message.data;
                        displayStreamingSources(sources);
                        break;
                    case 'answer':
                        fullAnswer += message.data;
                        updateStreamingAnswer(fullAnswer);
                        break;
                    case 'error':
                        throw new Error(message.data);
                    case 'done':
                        completeSearch(query, fullAnswer, sources);
                        ws.close();
                        break;
                    default:
                        console.log('Unknown message type:', message.type);
                }
            } catch (error) {
                console.error('Error processing WebSocket message:', error);
                throw error;
            }
        };

        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
            throw new Error('WebSocket connection failed');
        };

        ws.onclose = function() {
            console.log('WebSocket connection closed');
        };

    } catch (error) {
        console.error('Stream search error:', error);
        
        const statusIndicator = document.getElementById('status-indicator');
        if (statusIndicator) {
            statusIndicator.innerHTML = `
                <div class="text-red-600">
                    <span class="text-sm">${t('errorOccurred')}: ${error.message}</span>
                </div>
            `;
        }
        
        throw error; // 폴백을 위해 에러를 다시 던짐
    }
}

// 상태 업데이트 함수
function updateStatus(status) {
    const statusIndicator = document.getElementById('status-indicator');
    if (statusIndicator) {
        statusIndicator.innerHTML = `
            <div class="spinner mr-2"></div>
            <span class="text-sm text-slate-600">${status}</span>
        `;
    }
}

// 검색 완료 처리
async function completeSearch(query, fullAnswer, sources) {
    const statusIndicator = document.getElementById('status-indicator');
    if (statusIndicator) {
        statusIndicator.style.display = 'none';
    }
    
    // 최종 결과를 캐시에 저장
    const finalResult = {
        answer: fullAnswer,
        sources: sources,
        took: 0
    };
    await saveSearchResultToCache(query, finalResult);
    
    // 입력창 초기화
    const input = document.getElementById('search-input');
    if (input) {
        input.value = '';
    }
}
