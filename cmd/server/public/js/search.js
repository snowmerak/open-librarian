// 검색 처리 - 기본적으로 스트리밍을 사용하고 실패 시 일반 검색으로 폴백
async function handleSearch() {
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

    // 검색 시작 시 입력 필드 비우기
    input.value = '';

    // SSE 스트리밍 검색 시도
    try {
        await handleStreamSearch(query);
        return;
    } catch (error) {
        console.warn('Stream search failed, falling back to regular search:', error);
        
        // 스트리밍 실패 시 기존 UI 정리
        const existingResult = document.getElementById('streaming-result');
        if (existingResult) {
            existingResult.remove();
        }
    }

    // 폴백: 일반 검색 (query는 이미 가져왔고 input은 이미 비워짐)
    const searchResults = document.getElementById('search-results');
    const welcomeMessage = document.getElementById('welcome-message');
    const container = searchResults.querySelector('.max-w-4xl');
    
    // 환영 메시지 숨기기
    if (welcomeMessage) {
        welcomeMessage.style.display = 'none';
    }

    // 기존 검색 결과들을 모두 숨기기
    const existingResults = container.querySelectorAll('.search-result');
    existingResults.forEach(result => {
        result.style.display = 'none';
    });

    // 검색 기록에 저장
    await saveSearchToHistory(query);

    // 세션 ID 생성
    const sessionId = `search-${Date.now()}`;

    // 로딩 표시
    const loadingDiv = document.createElement('div');
    loadingDiv.id = sessionId;
    loadingDiv.className = 'search-result';
    
    // 현재 시간 표시
    const now = new Date();
    const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    
    loadingDiv.innerHTML = `
        <div class="flex justify-between items-start mb-4">
            <h3 class="text-lg font-semibold text-slate-800">"${escapeHtml(query)}" ${t('answerFor')}</h3>
            <div class="flex items-center gap-2">
                <span class="text-xs text-slate-500">${timeStr}</span>
                <button onclick="goBackToSearch()" class="px-3 py-1 text-xs bg-slate-100 hover:bg-slate-200 text-slate-600 rounded transition-colors" title="새 검색">
                    새 검색
                </button>
            </div>
        </div>
        <div class="flex items-center">
            <div class="spinner mr-3"></div>
            <span>${t('generating')}</span>
        </div>
    `;
    container.appendChild(loadingDiv);

    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/search`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ query: query })
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        
        // 결과를 캐시에 저장
        await saveSearchResultToCache(query, data);
        
        // 로딩 제거하고 결과 표시
        loadingDiv.remove();

        // 검색 결과 표시 (세션 ID 재사용)
        const resultDiv = document.createElement('div');
        resultDiv.className = 'search-result';
        resultDiv.id = sessionId;
        resultDiv.setAttribute('data-query', query);
        
        let sourcesHtml = '';
        if (data.sources && data.sources.length > 0) {
            sourcesHtml = `
                <div class="mt-6 pt-6 border-t border-slate-200">
                    <h4 class="font-semibold text-slate-700 mb-4">${t('references')}</h4>
                    <div class="grid gap-3">
                        ${data.sources.map(source => {
                            const article = source.article || source;
                            const title = article.title || 'Untitled';
                            const summary = article.summary || '';
                            const content = article.content || '';
                            const displayText = summary || content;
                            const author = article.author || 'Unknown';
                            const url = article.original_url || '#';
                            const score = source.score ? (source.score * 100).toFixed(2) : '0.00';
                            
                            const createdDate = article.created_date ? formatCreatedDate(article.created_date) : '';
                            const authorWithDate = createdDate ? `${escapeHtml(author)} • ${t('createdAt')}: ${createdDate}` : escapeHtml(author);
                            
                            return `
                                <div class="source-card" onclick="window.open('${escapeHtml(url)}', '_blank')">
                                    <div class="flex justify-between items-start mb-2">
                                        <div class="font-semibold text-indigo-700 flex-1">${escapeHtml(title)}</div>
                                        <div class="text-xs text-slate-500 ml-2 bg-slate-100 px-2 py-1 rounded">${score}%</div>
                                    </div>
                                    <p class="text-sm text-slate-600 mb-2">${escapeHtml(displayText.substring(0, 150))}...</p>
                                    <div class="text-xs text-slate-400">${authorWithDate}</div>
                                </div>
                            `;
                        }).join('')}
                    </div>
                </div>
            `;
        }

        resultDiv.innerHTML = `
            <div class="flex justify-between items-start mb-4">
                <h3 class="text-lg font-semibold text-slate-800">"${escapeHtml(query)}" ${t('answerFor')}</h3>
                <div class="flex items-center gap-2">
                    <span class="text-xs text-slate-500">${timeStr}</span>
                    <button onclick="goBackToSearch()" class="px-3 py-1 text-xs bg-slate-100 hover:bg-slate-200 text-slate-600 rounded transition-colors" title="새 검색">
                        새 검색
                    </button>
                </div>
            </div>
            <div class="markdown-content text-slate-700 mb-4">${marked.parse(data.answer || '답변을 생성할 수 없습니다.')}</div>
            ${sourcesHtml}
        `;

        container.appendChild(resultDiv);
        
        // 검색 기록에 세션 ID 저장
        updateSearchHistoryWithSession(query, sessionId);

    } catch (error) {
        console.error('Search error:', error);
        
        // 로딩 제거
        if (loadingDiv && loadingDiv.parentNode) {
            loadingDiv.remove();
        }

        // 에러 메시지 표시
        const errorDiv = document.createElement('div');
        errorDiv.id = sessionId;
        errorDiv.className = 'search-result border-red-200 bg-red-50';
        errorDiv.innerHTML = `
            <div class="flex justify-between items-start mb-4">
                <h3 class="text-lg font-semibold text-slate-800">"${escapeHtml(query)}" ${t('answerFor')}</h3>
                <div class="flex items-center gap-2">
                    <span class="text-xs text-slate-500">${timeStr}</span>
                    <button onclick="goBackToSearch()" class="px-3 py-1 text-xs bg-slate-100 hover:bg-slate-200 text-slate-600 rounded transition-colors" title="새 검색">
                        새 검색
                    </button>
                </div>
            </div>
            <div class="text-red-600">
                <h4 class="font-semibold mb-2">${t('errorOccurred')}</h4>
                <p>${t('errorMessage')}</p>
            </div>
        `;
        container.appendChild(errorDiv);
    }
}

function displaySearchResult(query, data) {
    const searchResults = document.getElementById('search-results');
    const container = searchResults.querySelector('.max-w-4xl');
    
    // 기존 검색 결과들을 모두 숨기기 (삭제하지 않고 숨김)
    const existingResults = container.querySelectorAll('.search-result');
    existingResults.forEach(result => {
        result.style.display = 'none';
    });
    
    // 검색 세션 ID 생성 (timestamp 기반)
    const sessionId = `search-${Date.now()}`;
    
    // 검색 결과 표시
    const resultDiv = document.createElement('div');
    resultDiv.className = 'search-result';
    resultDiv.id = sessionId;
    resultDiv.setAttribute('data-query', query);
    
    let sourcesHtml = '';
    if (data.sources && data.sources.length > 0) {
        sourcesHtml = `
            <div class="mt-6 pt-6 border-t border-slate-200">
                <h4 class="font-semibold text-slate-700 mb-4">${t('references')}</h4>
                <div class="grid gap-3">
                    ${data.sources.map(source => {
                        const article = source.article || source;
                        const title = article.title || 'Untitled';
                        const summary = article.summary || '';
                        const content = article.content || '';
                        const displayText = summary || content;
                        const author = article.author || 'Unknown';
                        const url = article.original_url || '#';
                        const score = source.score ? (source.score * 100).toFixed(2) : '0.00';
                        
                        const createdDate = article.created_date ? formatCreatedDate(article.created_date) : '';
                        const authorWithDate = createdDate ? `${escapeHtml(author)} • ${t('createdAt')}: ${createdDate}` : escapeHtml(author);
                        
                        return `
                            <div class="source-card" onclick="window.open('${escapeHtml(url)}', '_blank')">
                                <div class="flex justify-between items-start mb-2">
                                    <div class="font-semibold text-indigo-700 flex-1">${escapeHtml(title)}</div>
                                    <div class="text-xs text-slate-500 ml-2 bg-slate-100 px-2 py-1 rounded">${score}%</div>
                                </div>
                                <p class="text-sm text-slate-600 mb-2">${escapeHtml(displayText.substring(0, 150))}...</p>
                                <div class="text-xs text-slate-400">${authorWithDate}</div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        `;
    }

    // 현재 시간 표시
    const now = new Date();
    const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

    resultDiv.innerHTML = `
        <div class="flex justify-between items-start mb-4">
            <h3 class="text-lg font-semibold text-slate-800">"${escapeHtml(query)}" ${t('answerFor')}</h3>
            <div class="flex items-center gap-2">
                <span class="text-xs text-slate-500">${timeStr}</span>
                <button onclick="goBackToSearch()" class="px-3 py-1 text-xs bg-slate-100 hover:bg-slate-200 text-slate-600 rounded transition-colors" title="새 검색">
                    새 검색
                </button>
            </div>
        </div>
        <div class="markdown-content text-slate-700 mb-4">${marked.parse(data.answer || '답변을 생성할 수 없습니다.')}</div>
        ${sourcesHtml}
    `;

    container.appendChild(resultDiv);
    
    // 상단으로 스크롤
    searchResults.scrollIntoView({ behavior: 'smooth', block: 'start' });
    
    // 검색 기록에 세션 ID 저장
    updateSearchHistoryWithSession(query, sessionId);
}

async function searchFromHistory(query, sessionId = null) {
    // 세션 ID가 있으면 해당 결과를 표시 (다른 결과들은 숨김)
    if (sessionId) {
        const searchResults = document.getElementById('search-results');
        const container = searchResults.querySelector('.max-w-4xl');
        const welcomeMessage = document.getElementById('welcome-message');
        
        // 환영 메시지 숨기기
        if (welcomeMessage) {
            welcomeMessage.style.display = 'none';
        }
        
        // 모든 결과 숨기기
        const allResults = container.querySelectorAll('.search-result');
        allResults.forEach(result => {
            result.style.display = 'none';
        });
        
        // 해당 세션의 결과만 표시
        const targetResult = document.getElementById(sessionId);
        if (targetResult) {
            targetResult.style.display = 'block';
            searchResults.scrollIntoView({ behavior: 'smooth', block: 'start' });
            
            // 잠시 하이라이트 효과
            targetResult.style.boxShadow = '0 0 20px rgba(79, 70, 229, 0.3)';
            setTimeout(() => {
                targetResult.style.boxShadow = '';
            }, 2000);
            return;
        }
    }
    
    const cachedResult = await getCachedSearchResult(query);
    
    if (cachedResult) {
        // 캐시된 결과가 있으면 표시
        displayCachedResult(query, cachedResult);
    } else {
        // 캐시된 결과가 없으면 새로 검색
        document.getElementById('search-input').value = query;
        handleSearch();
    }
}

// 새 검색으로 돌아가기
function goBackToSearch() {
    const searchResults = document.getElementById('search-results');
    const container = searchResults.querySelector('.max-w-4xl');
    const welcomeMessage = document.getElementById('welcome-message');
    
    // 모든 검색 결과 숨기기
    const allResults = container.querySelectorAll('.search-result');
    allResults.forEach(result => {
        result.style.display = 'none';
    });
    
    // 환영 메시지 표시
    if (welcomeMessage) {
        welcomeMessage.style.display = 'block';
    }
    
    // 검색창 초기화 및 포커스
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
        searchInput.value = '';
        searchInput.focus();
    }
    
    // 상단으로 스크롤
    searchResults.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function scrollToExistingResult(query) {
    // 현재 페이지에서 해당 검색어의 결과를 찾기
    const resultDivs = document.querySelectorAll('.search-result');
    for (let div of resultDivs) {
        const titleElement = div.querySelector('h3');
        if (titleElement && titleElement.textContent.includes(`"${query}"`)) {
            div.scrollIntoView({ behavior: 'smooth', block: 'start' });
            // 잠시 하이라이트 효과
            div.style.boxShadow = '0 0 20px rgba(79, 70, 229, 0.3)';
            setTimeout(() => {
                div.style.boxShadow = '';
            }, 2000);
            return true;
        }
    }
    return false;
}

// 검색 결과 제거 함수
function removeSearchResult(sessionId) {
    const resultElement = document.getElementById(sessionId);
    if (resultElement) {
        resultElement.remove();
        
        // 히스토리에서도 제거
        removeFromSearchHistory(sessionId);
        
        // 결과가 하나도 없으면 환영 메시지 표시
        const remainingResults = document.querySelectorAll('.search-result');
        if (remainingResults.length === 0) {
            const welcomeMessage = document.getElementById('welcome-message');
            if (welcomeMessage) {
                welcomeMessage.style.display = 'block';
            }
        }
    }
}

function displayCachedResult(query, cachedResult) {
    const searchResults = document.getElementById('search-results');
    const welcomeMessage = document.getElementById('welcome-message');
    
    // 환영 메시지 숨기기
    if (welcomeMessage) {
        welcomeMessage.style.display = 'none';
    }

    // 캐시된 결과 표시
    displaySearchResult(query, cachedResult);
}

// WebSocket을 사용한 스트리밍 검색 처리
async function handleStreamSearch(query) {
    // 먼저 캐시된 결과 확인
    const cachedResult = await getCachedSearchResult(query);
    if (cachedResult) {
        displayCachedResult(query, cachedResult);
        return;
    }

    const searchResults = document.getElementById('search-results');
    const welcomeMessage = document.getElementById('welcome-message');
    const container = searchResults.querySelector('.max-w-4xl');
    
    // 환영 메시지 숨기기
    if (welcomeMessage) {
        welcomeMessage.style.display = 'none';
    }

    // 기존 검색 결과들을 모두 숨기기
    const existingResults = container.querySelectorAll('.search-result');
    existingResults.forEach(result => {
        result.style.display = 'none';
    });

    // 검색 기록에 저장
    await saveSearchToHistory(query);

    // 세션 ID 생성
    const sessionId = `search-${Date.now()}`;

    // 결과 컨테이너 생성
    const resultDiv = document.createElement('div');
    resultDiv.id = sessionId;
    resultDiv.className = 'search-result';
    resultDiv.setAttribute('data-query', query);
    
    // 현재 시간 표시
    const now = new Date();
    const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    
    // 초기 UI 설정
    resultDiv.innerHTML = `
        <div class="flex justify-between items-start mb-4">
            <h3 class="text-lg font-semibold text-slate-800">"${escapeHtml(query)}" ${t('answerFor')}</h3>
            <div class="flex items-center gap-2">
                <span class="text-xs text-slate-500">${timeStr}</span>
                <button onclick="goBackToSearch()" class="px-3 py-1 text-xs bg-slate-100 hover:bg-slate-200 text-slate-600 rounded transition-colors" title="새 검색">
                    새 검색
                </button>
            </div>
        </div>
        <div class="mb-4">
            <div id="status-indicator-${sessionId}" class="flex items-center mb-3">
                <div class="spinner mr-2"></div>
                <span class="text-sm text-slate-600">${t('searchingInProgress')}</span>
            </div>
            <div id="answer-content-${sessionId}" class="markdown-content text-slate-700 min-h-[50px] border-l-4 border-blue-500 pl-4 bg-blue-50 rounded-r-lg"></div>
        </div>
        <div id="sources-container-${sessionId}" class="mt-6 pt-6 border-t border-slate-200" style="display: none;">
            <h4 class="font-semibold text-slate-700 mb-4">${t('references')}</h4>
            <div id="sources-list-${sessionId}" class="grid gap-3"></div>
        </div>
    `;
    
    container.appendChild(resultDiv);
    searchResults.scrollIntoView({ behavior: 'smooth', block: 'start' });

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
                        updateStatus(message.data, sessionId);
                        break;
                    case 'sources':
                        sources = message.data;
                        displayStreamingSources(sources, sessionId);
                        break;
                    case 'answer':
                        fullAnswer += message.data;
                        updateStreamingAnswer(fullAnswer, sessionId);
                        break;
                    case 'error':
                        throw new Error(message.data);
                    case 'done':
                        completeSearch(query, fullAnswer, sources, sessionId);
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
        
        const statusIndicator = document.getElementById(`status-indicator-${sessionId}`);
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
function updateStatus(status, sessionId) {
    const statusIndicator = document.getElementById(`status-indicator-${sessionId}`);
    if (statusIndicator) {
        statusIndicator.innerHTML = `
            <div class="spinner mr-2"></div>
            <span class="text-sm text-slate-600">${status}</span>
        `;
    }
}

// 검색 완료 처리
async function completeSearch(query, fullAnswer, sources, sessionId) {
    const statusIndicator = document.getElementById(`status-indicator-${sessionId}`);
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
    
    // 히스토리에 세션 ID 업데이트
    updateSearchHistoryWithSession(query, sessionId);
}

// 스트리밍 답변 업데이트
function updateStreamingAnswer(answer, sessionId) {
    const answerContent = document.getElementById(`answer-content-${sessionId}`);
    if (answerContent) {
        answerContent.innerHTML = marked.parse(answer);
    }
}

// 스트리밍 소스 표시
function displayStreamingSources(sources, sessionId) {
    if (!sources || sources.length === 0) return;
    
    const sourcesContainer = document.getElementById(`sources-container-${sessionId}`);
    const sourcesList = document.getElementById(`sources-list-${sessionId}`);
    
    if (!sourcesContainer || !sourcesList) return;
    
    let sourcesHtml = '';
    sources.forEach(source => {
        const article = source.article || source;
        const title = article.title || 'Untitled';
        const summary = article.summary || '';
        const content = article.content || '';
        const displayText = summary || content;
        const author = article.author || 'Unknown';
        const url = article.original_url || '#';
        const score = source.score ? (source.score * 100).toFixed(2) : '0.00';
        
        const createdDate = article.created_date ? formatCreatedDate(article.created_date) : '';
        const authorWithDate = createdDate ? `${escapeHtml(author)} • ${t('createdAt')}: ${createdDate}` : escapeHtml(author);
        
        sourcesHtml += `
            <div class="source-card" onclick="window.open('${escapeHtml(url)}', '_blank')">
                <div class="flex justify-between items-start mb-2">
                    <div class="font-semibold text-indigo-700 flex-1">${escapeHtml(title)}</div>
                    <div class="text-xs text-slate-500 ml-2 bg-slate-100 px-2 py-1 rounded">${score}%</div>
                </div>
                <p class="text-sm text-slate-600 mb-2">${escapeHtml(displayText.substring(0, 150))}...</p>
                <div class="text-xs text-slate-400">${authorWithDate}</div>
            </div>
        `;
    });
    
    sourcesList.innerHTML = sourcesHtml;
    sourcesContainer.style.display = 'block';
}

// Global function registration for HTML onclick events
window.removeSearchResult = removeSearchResult;
window.searchFromHistory = searchFromHistory;
window.goBackToSearch = goBackToSearch;
