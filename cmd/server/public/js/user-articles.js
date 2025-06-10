// 사용자 아티클 관리 기능
class UserArticleManager {
    constructor() {
        this.currentArticles = [];
        this.currentPage = 0;
        this.pageSize = 20;
        this.totalArticles = 0;
        this.isLoading = false;
    }

    // 날짜 범위로 사용자 아티클 조회
    async getUserArticles(dateFrom = '', dateTo = '', from = 0, size = 20) {
        if (this.isLoading) return;
        
        this.isLoading = true;
        const loadingIndicator = document.getElementById('user-articles-loading');
        if (loadingIndicator) loadingIndicator.style.display = 'block';

        try {
            const token = localStorage.getItem('jwt_token');
            if (!token) {
                throw new Error('Authentication required');
            }

            const requestBody = {
                date_from: dateFrom,
                date_to: dateTo,
                from: from,
                size: size
            };

            const response = await fetch(`${API_BASE_URL}/api/v1/articles/user`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify(requestBody)
            });

            if (!response.ok) {
                if (response.status === 401) {
                    localStorage.removeItem('jwt_token');
                    window.location.reload();
                    return;
                }
                throw new Error(`Failed to fetch articles: ${response.status}`);
            }

            const data = await response.json();
            
            this.currentArticles = data.articles || [];
            this.totalArticles = data.total || 0;
            this.currentPage = Math.floor(from / size);
            
            this.displayArticles();
            this.updatePagination();
            
            return data;
        } catch (error) {
            console.error('Error fetching user articles:', error);
            this.showError(error.message);
        } finally {
            this.isLoading = false;
            if (loadingIndicator) loadingIndicator.style.display = 'none';
        }
    }

    // 아티클 목록 표시
    displayArticles() {
        const container = document.getElementById('user-articles-list');
        if (!container) return;

        if (this.currentArticles.length === 0) {
            container.innerHTML = `
                <div class="text-center py-12">
                    <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="mx-auto mb-4 text-slate-400">
                        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                        <polyline points="14,2 14,8 20,8"></polyline>
                        <line x1="16" y1="13" x2="8" y2="13"></line>
                        <line x1="16" y1="17" x2="8" y2="17"></line>
                        <polyline points="10,9 9,9 8,9"></polyline>
                    </svg>
                    <h3 class="text-lg font-semibold text-slate-600 mb-2">등록된 아티클이 없습니다</h3>
                    <p class="text-slate-500">선택한 기간에 등록된 아티클이 없습니다.</p>
                </div>
            `;
            return;
        }

        const articlesHtml = this.currentArticles.map(article => {
            const createdDate = formatCreatedDate(article.created_date);
            const summary = article.summary || article.content || '';
            const truncatedSummary = summary.length > 200 ? summary.substring(0, 200) + '...' : summary;
            
            return `
                <div class="article-item bg-white border border-slate-200 rounded-lg p-6 hover:border-indigo-300 hover:shadow-md transition-all cursor-pointer" data-article-id="${article.id}">
                    <div class="flex justify-between items-start mb-3">
                        <h3 class="text-lg font-semibold text-slate-800 flex-1 pr-4">${escapeHtml(article.title)}</h3>
                        <div class="flex items-center gap-2 flex-shrink-0">
                            <button onclick="userArticleManager.viewArticleDetail('${article.id}')" 
                                    class="px-3 py-1 text-sm bg-indigo-100 hover:bg-indigo-200 text-indigo-700 rounded transition-colors" 
                                    title="상세보기">
                                상세보기
                            </button>
                            <button onclick="userArticleManager.deleteArticle('${article.id}', '${escapeHtml(article.title).replace(/'/g, '\\\'')}')" 
                                    class="px-3 py-1 text-sm bg-red-100 hover:bg-red-200 text-red-700 rounded transition-colors" 
                                    title="삭제">
                                삭제
                            </button>
                        </div>
                    </div>
                    
                    <p class="text-slate-600 mb-3 text-sm leading-relaxed">${escapeHtml(truncatedSummary)}</p>
                    
                    <div class="flex justify-between items-center text-xs text-slate-500">
                        <span>작성자: ${escapeHtml(article.author || 'Unknown')}</span>
                        <span>등록일: ${createdDate}</span>
                    </div>
                    
                    ${article.tags && article.tags.length > 0 ? `
                        <div class="mt-3 flex flex-wrap gap-1">
                            ${article.tags.map(tag => `
                                <span class="px-2 py-1 text-xs bg-slate-100 text-slate-600 rounded">${escapeHtml(tag)}</span>
                            `).join('')}
                        </div>
                    ` : ''}
                </div>
            `;
        }).join('');

        container.innerHTML = articlesHtml;

        // 클릭 이벤트 추가
        container.querySelectorAll('.article-item').forEach(item => {
            item.addEventListener('click', (e) => {
                if (e.target.tagName === 'BUTTON') return; // 버튼 클릭 시 이벤트 전파 방지
                const articleId = item.dataset.articleId;
                this.viewArticleDetail(articleId);
            });
        });
    }

    // 페이지네이션 업데이트
    updatePagination() {
        const paginationContainer = document.getElementById('user-articles-pagination');
        if (!paginationContainer) return;

        const totalPages = Math.ceil(this.totalArticles / this.pageSize);
        
        if (totalPages <= 1) {
            paginationContainer.style.display = 'none';
            return;
        }

        paginationContainer.style.display = 'flex';

        const prevDisabled = this.currentPage === 0;
        const nextDisabled = this.currentPage >= totalPages - 1;

        paginationContainer.innerHTML = `
            <button onclick="userArticleManager.previousPage()" 
                    ${prevDisabled ? 'disabled' : ''} 
                    class="px-3 py-2 text-sm border border-slate-300 rounded-l-md hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed">
                이전
            </button>
            <span class="px-4 py-2 text-sm border-t border-b border-slate-300 bg-slate-50">
                ${this.currentPage + 1} / ${totalPages} (총 ${this.totalArticles}개)
            </span>
            <button onclick="userArticleManager.nextPage()" 
                    ${nextDisabled ? 'disabled' : ''} 
                    class="px-3 py-2 text-sm border border-slate-300 rounded-r-md hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed">
                다음
            </button>
        `;
    }

    // 아티클 상세보기
    async viewArticleDetail(articleId) {
        try {
            const token = localStorage.getItem('jwt_token');
            const headers = {
                'Content-Type': 'application/json'
            };
            
            if (token) {
                headers['Authorization'] = `Bearer ${token}`;
            }

            const response = await fetch(`${API_BASE_URL}/api/v1/articles/${articleId}`, {
                headers: headers
            });

            if (!response.ok) {
                throw new Error(`Failed to fetch article: ${response.status}`);
            }

            const article = await response.json();
            this.showArticleModal(article);
        } catch (error) {
            console.error('Error fetching article detail:', error);
            this.showError('아티클을 불러오는데 실패했습니다.');
        }
    }

    // 아티클 상세보기 모달
    showArticleModal(article) {
        const modal = document.createElement('div');
        modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50';
        modal.onclick = (e) => {
            if (e.target === modal) modal.remove();
        };

        const createdDate = formatCreatedDate(article.created_date);
        
        modal.innerHTML = `
            <div class="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
                <div class="p-6 border-b border-slate-200">
                    <div class="flex justify-between items-start">
                        <h2 class="text-xl font-bold text-slate-800 pr-4">${escapeHtml(article.title)}</h2>
                        <button onclick="this.closest('.fixed').remove()" 
                                class="text-slate-400 hover:text-slate-600 p-1">
                            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                <line x1="18" y1="6" x2="6" y2="18"></line>
                                <line x1="6" y1="6" x2="18" y2="18"></line>
                            </svg>
                        </button>
                    </div>
                    <div class="mt-2 text-sm text-slate-500">
                        <span>작성자: ${escapeHtml(article.author || 'Unknown')}</span>
                        <span class="mx-2">•</span>
                        <span>등록일: ${createdDate}</span>
                        ${article.original_url ? `
                            <span class="mx-2">•</span>
                            <a href="${escapeHtml(article.original_url)}" target="_blank" class="text-indigo-600 hover:text-indigo-800">원문 보기</a>
                        ` : ''}
                    </div>
                </div>
                
                <div class="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
                    ${article.summary ? `
                        <div class="mb-6">
                            <h3 class="text-lg font-semibold text-slate-700 mb-3">요약</h3>
                            <div class="bg-blue-50 border-l-4 border-blue-500 p-4 rounded-r-lg">
                                <p class="text-slate-700 leading-relaxed whitespace-pre-wrap">${escapeHtml(article.summary)}</p>
                            </div>
                        </div>
                    ` : ''}
                    
                    <div>
                        <h3 class="text-lg font-semibold text-slate-700 mb-3">내용</h3>
                        <div class="prose max-w-none text-slate-700 leading-relaxed whitespace-pre-wrap">${escapeHtml(article.content)}</div>
                    </div>
                    
                    ${article.tags && article.tags.length > 0 ? `
                        <div class="mt-6 pt-6 border-t border-slate-200">
                            <h3 class="text-sm font-semibold text-slate-700 mb-2">태그</h3>
                            <div class="flex flex-wrap gap-2">
                                ${article.tags.map(tag => `
                                    <span class="px-3 py-1 text-sm bg-slate-100 text-slate-600 rounded-full">${escapeHtml(tag)}</span>
                                `).join('')}
                            </div>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;

        document.body.appendChild(modal);
    }

    // 아티클 삭제
    async deleteArticle(articleId, articleTitle) {
        if (!confirm(`"${articleTitle}" 아티클을 삭제하시겠습니까?\n\n이 작업은 되돌릴 수 없습니다.`)) {
            return;
        }

        try {
            const token = localStorage.getItem('jwt_token');
            if (!token) {
                throw new Error('Authentication required');
            }

            const response = await fetch(`${API_BASE_URL}/api/v1/articles/${articleId}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });

            if (!response.ok) {
                if (response.status === 401) {
                    localStorage.removeItem('jwt_token');
                    window.location.reload();
                    return;
                }
                if (response.status === 403) {
                    throw new Error('권한이 없습니다. 본인이 등록한 아티클만 삭제할 수 있습니다.');
                }
                if (response.status === 404) {
                    throw new Error('아티클을 찾을 수 없습니다.');
                }
                throw new Error(`Failed to delete article: ${response.status}`);
            }

            // 성공 메시지 표시
            this.showSuccess('아티클이 성공적으로 삭제되었습니다.');
            
            // 목록 새로고침
            this.refreshCurrentList();
            
        } catch (error) {
            console.error('Error deleting article:', error);
            this.showError(error.message);
        }
    }

    // 이전 페이지
    previousPage() {
        if (this.currentPage > 0) {
            this.loadPage(this.currentPage - 1);
        }
    }

    // 다음 페이지
    nextPage() {
        const totalPages = Math.ceil(this.totalArticles / this.pageSize);
        if (this.currentPage < totalPages - 1) {
            this.loadPage(this.currentPage + 1);
        }
    }

    // 특정 페이지 로드
    loadPage(pageNumber) {
        const dateFrom = document.getElementById('date-from')?.value || '';
        const dateTo = document.getElementById('date-to')?.value || '';
        const from = pageNumber * this.pageSize;
        
        this.getUserArticles(dateFrom, dateTo, from, this.pageSize);
    }

    // 현재 목록 새로고침
    refreshCurrentList() {
        const dateFrom = document.getElementById('date-from')?.value || '';
        const dateTo = document.getElementById('date-to')?.value || '';
        const from = this.currentPage * this.pageSize;
        
        this.getUserArticles(dateFrom, dateTo, from, this.pageSize);
    }

    // 에러 메시지 표시
    showError(message) {
        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-red-500 text-white px-6 py-3 rounded-lg shadow-lg z-50';
        toast.textContent = message;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.remove();
        }, 5000);
    }

    // 성공 메시지 표시
    showSuccess(message) {
        const toast = document.createElement('div');
        toast.className = 'fixed top-4 right-4 bg-green-500 text-white px-6 py-3 rounded-lg shadow-lg z-50';
        toast.textContent = message;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.remove();
        }, 3000);
    }
}

// 전역 인스턴스 생성
const userArticleManager = new UserArticleManager();

// 날짜 범위 검색 함수
function searchUserArticles() {
    const dateFrom = document.getElementById('date-from')?.value || '';
    const dateTo = document.getElementById('date-to')?.value || '';
    
    // 날짜 유효성 검사
    if (dateFrom && dateTo && dateFrom > dateTo) {
        userArticleManager.showError('시작 날짜는 종료 날짜보다 이전이어야 합니다.');
        return;
    }
    
    userArticleManager.getUserArticles(dateFrom, dateTo, 0, 20);
}

// 오늘 날짜로 설정
function setToday() {
    const today = new Date().toISOString().split('T')[0];
    document.getElementById('date-to').value = today;
}

// 일주일 전 날짜로 설정
function setLastWeek() {
    const lastWeek = new Date();
    lastWeek.setDate(lastWeek.getDate() - 7);
    document.getElementById('date-from').value = lastWeek.toISOString().split('T')[0];
}

// 한달 전 날짜로 설정  
function setLastMonth() {
    const lastMonth = new Date();
    lastMonth.setMonth(lastMonth.getMonth() - 1);
    document.getElementById('date-from').value = lastMonth.toISOString().split('T')[0];
}

// 전체 기간으로 설정
function setAllTime() {
    document.getElementById('date-from').value = '';
    document.getElementById('date-to').value = '';
}
