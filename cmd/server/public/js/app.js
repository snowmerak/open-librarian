const API_BASE_URL = window.location.hostname === 'localhost' ? 
    'http://localhost:8080' : 
    `http://${window.location.hostname}:8080`;

// 페이지 로드 시 초기화
document.addEventListener('DOMContentLoaded', async function() {
    try {
        await initLanguage(); // 언어 초기화
        
        // IndexedDB 로직 제거: 서버 사이드 관리로 전환됨
        // await initDB();
        
        await loadSearchHistory();
        initEventListeners();
        initArticleForm();
    } catch (error) {
        console.error('Failed to initialize app:', error);
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
