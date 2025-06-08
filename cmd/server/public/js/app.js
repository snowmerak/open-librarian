const API_BASE_URL = window.location.hostname === 'localhost' ? 
    'http://localhost:8080' : 
    `http://${window.location.hostname}:8080`;

// 페이지 로드 시 초기화
document.addEventListener('DOMContentLoaded', async function() {
    try {
        await initDB();
        await initLanguage(); // 언어 초기화 추가
        await loadSearchHistory();
        initEventListeners();
        initArticleForm();
    } catch (error) {
        console.error('Failed to initialize database:', error);
        // Fallback to localStorage if IndexedDB fails
        await initLanguage();
        loadSearchHistoryFromLocalStorage();
        initEventListeners();
        initArticleForm();
    }
});

// 이벤트 리스너 초기화
function initEventListeners() {
    // 검색창에서 Enter 키 입력 시 검색 실행
    document.getElementById('search-input').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            handleSearch();
        }
    });
    
    // 모바일 언어 선택기 동기화
    const languageSelectMobile = document.getElementById('language-select-mobile');
    if (languageSelectMobile) {
        languageSelectMobile.addEventListener('change', function(e) {
            const languageSelect = document.getElementById('language-select');
            if (languageSelect) {
                languageSelect.value = e.target.value;
            }
        });
    }
}
