// 화면 전환 로직
function showView(viewId) {
    document.querySelectorAll('.view').forEach(view => {
        view.classList.add('hidden');
    });
    document.getElementById(viewId).classList.remove('hidden');

    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
    });
    document.querySelector(`[onclick="showView('${viewId}')"]`).classList.add('active');
}

let currentHistoryPage = 1;
const historyPageSize = 20;

// 검색 히스토리 UI 업데이트
async function updateHistoryDisplay(append = false) {
    const historyContainer = document.getElementById('search-history');
    
    if (!append) {
        currentHistoryPage = 1;
    }

    try {
        const response = await fetch(`/api/v1/chat/history?page=${currentHistoryPage}&size=${historyPageSize}`);
        if (!response.ok) throw new Error('Failed to fetch history');
        const history = await response.json();
        
        // Remove existing "Load More" button if it exists
        const loadMoreBtn = document.getElementById('history-load-more');
        if (loadMoreBtn) loadMoreBtn.remove();

        if ((!history || history.length === 0) && !append) {
            historyContainer.innerHTML = `<div class="text-xs text-slate-400 hidden md:block text-center p-4">${t('noSearchHistory') || '기록 없음'}</div>`;
            return;
        }

        if (history.length === 0 && append) {
            return;
        }

        const html = history.map(item => {
            const timeAgo = getTimeAgo(item.updated_at);
            const title = item.title || "New Chat";
            const onclickFunc = `loadChatSession('${item.id}')`;
            
            return `
                <div class="history-item hidden md:block group relative hover:bg-slate-100 rounded-md transition-colors" title="${escapeHtml(title)} (${timeAgo})">
                    <div class="history-text cursor-pointer p-2" onclick="${onclickFunc}">
                        <div class="text-sm text-slate-700 truncate font-medium">${escapeHtml(title)}</div>
                        <div class="text-xs text-slate-400 mt-0.5">${timeAgo}</div>
                    </div>
                    <button class="delete-btn opacity-0 group-hover:opacity-100 transition-opacity absolute right-2 top-2 text-slate-400 hover:text-red-500 p-1 hover:bg-slate-200 rounded" onclick="deleteChatSession(event, '${item.id}')" title="삭제">
                        ×
                    </button>
                </div>
            `;
        }).join('');

        if (append) {
            historyContainer.insertAdjacentHTML('beforeend', html);
        } else {
            historyContainer.innerHTML = html;
        }

        // Add Load More button if we received a full page
        if (history.length === historyPageSize) {
             const moreBtn = document.createElement('button');
             moreBtn.id = 'history-load-more';
             moreBtn.className = 'w-full text-xs text-slate-500 hover:text-slate-700 py-2 mt-2 border-t border-slate-200 hidden md:block transition-colors';
             moreBtn.innerText = t('loadMore') || '더 보기';
             moreBtn.onclick = () => {
                 currentHistoryPage++;
                 updateHistoryDisplay(true);
             };
             historyContainer.appendChild(moreBtn);
        }

    } catch (e) {
        console.error("History load error:", e);
        if (!append) historyContainer.innerHTML = `<div class="text-xs text-red-400 hidden md:block text-center p-4">Load Error</div>`;
    }
}

// 세션 로드 함수
async function loadChatSession(sessionId) {
    try {
        const response = await fetch(`/api/v1/chat/history/${sessionId}`);
        if (!response.ok) throw new Error('Failed to load session');
        const session = await response.json();
        
        // UI 초기화
        clearChatInterface();
        setCurrentSession(sessionId);
        
        // 환영 메시지 숨기기
        const welcomeMessage = document.getElementById('welcome-message');
        if (welcomeMessage) welcomeMessage.style.display = 'none';

        // 메시지 렌더링
        if (session.messages && session.messages.length > 0) {
            session.messages.forEach(msg => {
                if (msg.role === 'user') {
                    appendUserMessage(msg.content);
                } else if (msg.role === 'assistant') {
                    renderStaticAiMessage(msg.content, msg.sources);
                }
            });
            scrollToBottom();
        }
    } catch (e) {
        console.error("Session load error:", e);
        alert("Failed to load search session.");
    }
}

// 세션 삭제
async function deleteChatSession(event, sessionId) {
    event.stopPropagation(); // 부모 onclick 방지
    if (!confirm(t('deleteConfirm') || '삭제하시겠습니까?')) return;

    try {
        const response = await fetch(`/api/v1/chat/history/${sessionId}`, { method: 'DELETE' });
        if (response.ok) {
            if (typeof currentSessionId !== 'undefined' && currentSessionId === sessionId) {
                 clearChatInterface();
            }
            updateHistoryDisplay();
        } else {
            alert("삭제 실패");
        }
    } catch (e) {
        console.error(e);
    }
}

// 시간 차이 계산 함수
function getTimeAgo(timestamp) {
    const now = new Date();
    const past = new Date(timestamp);
    const diffMs = now - past;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);
    
    if (diffMins < 1) return t('justNow') || '방금 전';
    if (diffMins < 60) return `${diffMins}${t('minutesAgo') || '분 전'}`;
    if (diffHours < 24) return `${diffHours}${t('hoursAgo') || '시간 전'}`;
    if (diffDays < 7) return `${diffDays}${t('daysAgo') || '일 전'}`;
    
    return past.toLocaleDateString();
}

window.loadChatSession = loadChatSession; // Export
window.deleteChatSession = deleteChatSession; // Export

async function loadSearchHistory() {
    await updateHistoryDisplay();
}

async function clearSearchHistory() {
    if (!confirm(t('clearAll') + '?')) {
        return;
    }
    
    // Server clear not implemented yet? User requested delete support.
    // We implemented single delete. Bulk delete logic can be added later if needed.
    // For now, let's just alert.
    alert("Not supported yet");
    
    /*
    if (!db) {
        localStorage.removeItem('librarian_search_history');
        localStorage.removeItem('librarian_search_results_cache');
        await updateHistoryDisplay();
        return;
    }
    
    try {
        // 히스토리 삭제
        const historyTransaction = db.transaction([STORE_HISTORY], 'readwrite');
        const historyStore = historyTransaction.objectStore(STORE_HISTORY);
        await new Promise((resolve, reject) => {
            const request = historyStore.clear();
            request.onsuccess = () => resolve();
            request.onerror = () => reject(request.error);
        });
        
        // 캐시 삭제
        const cacheTransaction = db.transaction([STORE_CACHE], 'readwrite');
        const cacheStore = cacheTransaction.objectStore(STORE_CACHE);
        await new Promise((resolve, reject) => {
            const request = cacheStore.clear();
            request.onsuccess = () => resolve();
            request.onerror = () => reject(request.error);
        });
        
        await updateHistoryDisplay();
    } catch (error) {
        console.error('Failed to clear search history:', error);
    }
    */
    alert("Not supported yet");
}

// HTML 이스케이프 함수
function escapeHtml(text) {
    if (text === null || text === undefined) {
        return '';
    }
    if (typeof text !== 'string') {
        text = String(text);
    }
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, function(m) { return map[m]; });
}

// 모바일 언어 선택기 동기화
function updateLanguageSelector() {
    const languageSelect = document.getElementById('language-select');
    const languageSelectMobile = document.getElementById('language-select-mobile');
    
    if (languageSelect) {
        languageSelect.value = currentLanguage;
    }
    if (languageSelectMobile) {
        languageSelectMobile.value = currentLanguage;
    }
}
