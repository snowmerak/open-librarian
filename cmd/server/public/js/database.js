// IndexedDB 설정
const DB_NAME = 'LibrarianDB';
const DB_VERSION = 2; // 버전 업데이트
const STORE_HISTORY = 'searchHistory';
const STORE_CACHE = 'searchResultsCache';
const STORE_SETTINGS = 'settings';

let db = null;

// IndexedDB 초기화
async function initDB() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open(DB_NAME, DB_VERSION);
        
        request.onerror = () => reject(request.error);
        request.onsuccess = () => {
            db = request.result;
            resolve(db);
        };
        
        request.onupgradeneeded = (event) => {
            const database = event.target.result;
            
            // 검색 히스토리 스토어
            if (!database.objectStoreNames.contains(STORE_HISTORY)) {
                const historyStore = database.createObjectStore(STORE_HISTORY, { keyPath: 'id', autoIncrement: true });
                historyStore.createIndex('query', 'query', { unique: false });
                historyStore.createIndex('timestamp', 'timestamp', { unique: false });
            }
            
            // 검색 결과 캐시 스토어
            if (!database.objectStoreNames.contains(STORE_CACHE)) {
                const cacheStore = database.createObjectStore(STORE_CACHE, { keyPath: 'query' });
                cacheStore.createIndex('timestamp', 'timestamp', { unique: false });
            }
            
            // 설정 스토어 (언어 설정 등)
            if (!database.objectStoreNames.contains(STORE_SETTINGS)) {
                const settingsStore = database.createObjectStore(STORE_SETTINGS, { keyPath: 'key' });
                settingsStore.createIndex('timestamp', 'timestamp', { unique: false });
            }
        };
    });
}

// 검색 히스토리 관리 (IndexedDB)
async function saveSearchToHistory(query, sessionId = null) {
    if (!db) return;
    
    try {
        const transaction = db.transaction([STORE_HISTORY], 'readwrite');
        const store = transaction.objectStore(STORE_HISTORY);
        
        // 새 검색 추가
        await new Promise((resolve, reject) => {
            const addRequest = store.add({
                query: query,
                sessionId: sessionId,
                timestamp: new Date().toISOString()
            });
            addRequest.onsuccess = () => resolve();
            addRequest.onerror = () => reject(addRequest.error);
        });
        
        await updateHistoryDisplay();
    } catch (error) {
        console.error('Failed to save search to history:', error);
    }
}

// 히스토리에 세션 ID 업데이트
async function updateSearchHistoryWithSession(query, sessionId) {
    if (!db) return;
    
    try {
        const transaction = db.transaction([STORE_HISTORY], 'readwrite');
        const store = transaction.objectStore(STORE_HISTORY);
        const index = store.index('query');
        
        // 가장 최근의 해당 쿼리 찾기
        const request = index.openCursor(IDBKeyRange.only(query), 'prev');
        
        request.onsuccess = (event) => {
            const cursor = event.target.result;
            if (cursor) {
                const record = cursor.value;
                if (!record.sessionId) { // 세션 ID가 없는 가장 최근 기록 업데이트
                    record.sessionId = sessionId;
                    cursor.update(record);
                    updateHistoryDisplay();
                }
            }
        };
    } catch (error) {
        console.error('Failed to update search history with session:', error);
    }
}

// 세션 ID로 히스토리에서 제거
async function removeFromSearchHistory(sessionId) {
    if (!db) return;
    
    try {
        const transaction = db.transaction([STORE_HISTORY], 'readwrite');
        const store = transaction.objectStore(STORE_HISTORY);
        
        const request = store.openCursor();
        request.onsuccess = (event) => {
            const cursor = event.target.result;
            if (cursor) {
                if (cursor.value.sessionId === sessionId) {
                    cursor.delete();
                    updateHistoryDisplay();
                }
                cursor.continue();
            }
        };
    } catch (error) {
        console.error('Failed to remove from search history:', error);
    }
}

async function getSearchHistory() {
    if (!db) return [];
    
    try {
        return new Promise((resolve, reject) => {
            const transaction = db.transaction([STORE_HISTORY], 'readonly');
            const store = transaction.objectStore(STORE_HISTORY);
            const index = store.index('timestamp');
            const request = index.openCursor(null, 'prev'); // 최신순 정렬
            
            const results = [];
            request.onsuccess = (event) => {
                const cursor = event.target.result;
                if (cursor) {
                    results.push(cursor.value);
                    cursor.continue();
                } else {
                    resolve(results);
                }
            };
            request.onerror = () => reject(request.error);
        });
    } catch (error) {
        console.error('Failed to get search history:', error);
        return [];
    }
}

async function deleteSearchFromHistory(query) {
    if (!db) {
        deleteSearchFromHistoryLocalStorage(query);
        return;
    }
    
    try {
        const transaction = db.transaction([STORE_HISTORY], 'readwrite');
        const store = transaction.objectStore(STORE_HISTORY);
        const index = store.index('query');
        const request = index.openCursor(IDBKeyRange.only(query));
        
        request.onsuccess = async (event) => {
            const cursor = event.target.result;
            if (cursor) {
                await cursor.delete();
                cursor.continue();
            }
        };
        
        // 검색 결과 캐시에서도 제거
        const cacheTransaction = db.transaction([STORE_CACHE], 'readwrite');
        const cacheStore = cacheTransaction.objectStore(STORE_CACHE);
        await new Promise((resolve, reject) => {
            const deleteRequest = cacheStore.delete(query);
            deleteRequest.onsuccess = () => resolve();
            deleteRequest.onerror = () => reject(deleteRequest.error);
        });
        
        await updateHistoryDisplay();
    } catch (error) {
        console.error('Failed to delete search from history:', error);
    }
}

// 검색 결과 캐시 관리 (IndexedDB)
async function saveSearchResultToCache(query, result) {
    if (!db) return;
    
    try {
        const transaction = db.transaction([STORE_CACHE], 'readwrite');
        const store = transaction.objectStore(STORE_CACHE);
        
        await new Promise((resolve, reject) => {
            const request = store.put({
                query: query,
                result: result,
                timestamp: new Date().toISOString()
            });
            request.onsuccess = () => resolve();
            request.onerror = () => reject(request.error);
        });
    } catch (error) {
        console.error('Failed to save search result to cache:', error);
    }
}

async function getCachedSearchResult(query) {
    if (!db) return null;
    
    try {
        return new Promise((resolve, reject) => {
            const transaction = db.transaction([STORE_CACHE], 'readonly');
            const store = transaction.objectStore(STORE_CACHE);
            const request = store.get(query);
            
            request.onsuccess = () => {
                const result = request.result;
                resolve(result ? result.result : null);
            };
            request.onerror = () => reject(request.error);
        });
    } catch (error) {
        console.error('Failed to get cached search result:', error);
        return null;
    }
}

// LocalStorage 폴백 함수들
function deleteSearchFromHistoryLocalStorage(query) {
    try {
        const history = localStorage.getItem('librarian_search_history');
        if (history) {
            const parsedHistory = JSON.parse(history);
            const filteredHistory = parsedHistory.filter(item => item.query !== query);
            localStorage.setItem('librarian_search_history', JSON.stringify(filteredHistory));
        }
        
        // 캐시에서도 제거
        const cache = localStorage.getItem('librarian_search_results_cache');
        if (cache) {
            const parsedCache = JSON.parse(cache);
            delete parsedCache[query];
            localStorage.setItem('librarian_search_results_cache', JSON.stringify(parsedCache));
        }
        
        loadSearchHistoryFromLocalStorage();
    } catch (error) {
        console.error('Failed to delete search from localStorage:', error);
    }
}

function loadSearchHistoryFromLocalStorage() {
    // LocalStorage fallback implementation
    console.log('Using localStorage fallback for search history');
}
