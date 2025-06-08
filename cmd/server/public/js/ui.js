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

// 검색 히스토리 UI 업데이트
async function updateHistoryDisplay() {
    const historyContainer = document.getElementById('search-history');
    const history = await getSearchHistory();
    
    if (history.length === 0) {
        historyContainer.innerHTML = `<div class="text-xs text-slate-400 hidden md:block">${t('noSearchHistory')}</div>`;
        return;
    }
    
    historyContainer.innerHTML = history.map(item => {
        const timeAgo = getTimeAgo(item.timestamp);
        const sessionId = item.sessionId || '';
        const onclickFunc = sessionId ? 
            `searchFromHistory('${escapeHtml(item.query)}', '${sessionId}')` : 
            `searchFromHistory('${escapeHtml(item.query)}')`;
        
        return `
            <div class="history-item hidden md:block group" title="${escapeHtml(item.query)} (${timeAgo})">
                <div class="history-text" onclick="${onclickFunc}">
                    <div class="text-sm text-slate-700">${escapeHtml(item.query.length > 20 ? item.query.substring(0, 20) + '...' : item.query)}</div>
                    <div class="text-xs text-slate-400 mt-1">${timeAgo}</div>
                </div>
                <button class="delete-btn opacity-0 group-hover:opacity-100 transition-opacity" onclick="${sessionId ? `removeFromSearchHistory('${sessionId}')` : `deleteSearchFromHistory('${escapeHtml(item.query)}')`}" title="삭제">
                    ×
                </button>
            </div>
        `;
    }).join('');
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

async function loadSearchHistory() {
    await updateHistoryDisplay();
}

async function clearSearchHistory() {
    if (!confirm(t('clearAll') + '?')) {
        return;
    }
    
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
