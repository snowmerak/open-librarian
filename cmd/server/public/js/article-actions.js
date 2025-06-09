// Article deletion functionality

// JWT 토큰 저장/관리
let currentJWTToken = null;

// JWT 토큰 설정
function setJWTToken(token) {
    currentJWTToken = token;
    if (token) {
        localStorage.setItem('jwt_token', token);
    } else {
        localStorage.removeItem('jwt_token');
    }
}

// JWT 토큰 가져오기
function getJWTToken() {
    if (!currentJWTToken) {
        currentJWTToken = localStorage.getItem('jwt_token');
    }
    return currentJWTToken;
}

// Article 삭제 함수
async function deleteArticle(articleId, articleTitle) {
    const token = getJWTToken();
    
    if (!token) {
        alert('삭제하려면 로그인이 필요합니다.');
        return false;
    }
    
    if (!confirm(t('deleteConfirm'))) {
        return false;
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/articles/${articleId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });
        
        if (response.ok) {
            alert(t('deleteSuccess'));
            // 검색 결과에서 해당 article 제거
            removeArticleFromSearchResults(articleId);
            return true;
        } else if (response.status === 403) {
            alert(t('deletePermissionDenied'));
            return false;
        } else if (response.status === 404) {
            alert('Article not found');
            return false;
        } else {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
    } catch (error) {
        console.error('Delete article error:', error);
        alert(t('deleteError'));
        return false;
    }
}

// 검색 결과에서 article 제거
function removeArticleFromSearchResults(articleId) {
    // 모든 source-card에서 해당 article 제거
    const sourceCards = document.querySelectorAll('.source-card');
    sourceCards.forEach(card => {
        const deleteBtn = card.querySelector(`[data-article-id="${articleId}"]`);
        if (deleteBtn) {
            card.remove();
        }
    });
    
    // 만약 모든 source가 제거되었다면 references 섹션도 숨기기
    document.querySelectorAll('.search-result').forEach(result => {
        const sourcesContainer = result.querySelector('.grid.gap-3');
        if (sourcesContainer && sourcesContainer.children.length === 0) {
            const referencesSection = result.querySelector('.mt-6.pt-6.border-t');
            if (referencesSection) {
                referencesSection.style.display = 'none';
            }
        }
    });
}

// 임시 로그인 함수 (테스트용)
async function tempLogin(username, password) {
    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                username: username,
                password: password
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            setJWTToken(data.token);
            console.log('Login successful');
            return true;
        } else {
            console.error('Login failed');
            return false;
        }
    } catch (error) {
        console.error('Login error:', error);
        return false;
    }
}

// 페이지 로드시 JWT 토큰 복원
document.addEventListener('DOMContentLoaded', function() {
    getJWTToken();
});

// Global functions for HTML onclick events
window.deleteArticle = deleteArticle;
window.setJWTToken = setJWTToken;
window.tempLogin = tempLogin;
