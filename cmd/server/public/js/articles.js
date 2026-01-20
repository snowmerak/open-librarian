// 탭 전환 함수
function showAddTab(tab) {
    const singleTab = document.getElementById('single-tab');
    const bulkTab = document.getElementById('bulk-tab');
    const singleAdd = document.getElementById('single-add');
    const bulkAdd = document.getElementById('bulk-add');
    
    if (tab === 'single') {
        singleTab.className = 'flex-1 px-6 py-3 text-sm font-medium text-indigo-600 border-b-2 border-indigo-600';
        bulkTab.className = 'flex-1 px-6 py-3 text-sm font-medium text-slate-500 hover:text-slate-700';
        singleAdd.classList.remove('hidden');
        bulkAdd.classList.add('hidden');
    } else {
        singleTab.className = 'flex-1 px-6 py-3 text-sm font-medium text-slate-500 hover:text-slate-700';
        bulkTab.className = 'flex-1 px-6 py-3 text-sm font-medium text-indigo-600 border-b-2 border-indigo-600';
        singleAdd.classList.add('hidden');
        bulkAdd.classList.remove('hidden');
    }
}

// 아티클 추가 폼 이벤트 리스너 설정
function initArticleForm() {
    // 아티클 추가 폼 처리 - WebSocket을 이용한 실시간 진행률 표시
    document.getElementById('add-article-form').addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // 로그인 상태 확인
        if (!isLoggedIn()) {
            alert('로그인이 필요합니다.');
            showAuthModal('login');
            return;
        }
        
        const button = e.target.querySelector('button[type="submit"]');
        const originalText = button.innerHTML;
        
        button.disabled = true;
        button.innerHTML = `<div class="flex items-center justify-center"><div class="spinner mr-2"></div> ${t('processing')}</div>`;

        try {
            const formData = new FormData(e.target);
            const articleData = {
                title: formData.get('title'),
                content: formData.get('content'),
                original_url: formData.get('original_url') || '',
                author: formData.get('author') || ''
            };

            // Handle created_date conversion from datetime-local to RFC3339
            const createdDateValue = formData.get('created_date');
            if (createdDateValue) {
                // Convert from datetime-local format to RFC3339
                const localDate = new Date(createdDateValue);
                articleData.created_date = localDate.toISOString();
            }

            // Try WebSocket first, fallback to regular HTTP if WebSocket fails
            try {
                await handleWebSocketArticleAddition(articleData, button, originalText);
                e.target.reset();
                showView('search-view'); // 추가 후 검색 화면으로 이동
            } catch (wsError) {
                console.warn('WebSocket article addition failed, falling back to HTTP:', wsError);
                
                // Fallback to regular HTTP request with JWT token
                const token = getJWTToken();
                const response = await fetch(`${API_BASE_URL}/api/v1/articles`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify(articleData)
                });

                if (response.status === 401) {
                    alert('인증이 만료되었습니다. 다시 로그인해주세요.');
                    setJWTToken(null);
                    showAuthModal('login');
                    return;
                }

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                alert(t('articleAddedSuccess'));
                e.target.reset();
                showView('search-view'); // 추가 후 검색 화면으로 이동
            }

        } catch (error) {
            console.error('Article submission error:', error);
            alert(t('articleAddError'));
        } finally {
            button.disabled = false;
            button.innerHTML = originalText;
        }
    });

    // 파일 선택 처리 (JSONL 및 일반 문서)
    document.getElementById('jsonl-file').addEventListener('change', function(e) {
        const file = e.target.files[0];
        const uploadBtn = document.getElementById('bulk-upload-btn');
        const preview = document.getElementById('file-preview');
        const previewContent = document.getElementById('preview-content');
        const totalLines = document.getElementById('total-lines');
        
        if (!file) {
            uploadBtn.disabled = true;
            preview.classList.add('hidden');
            window.uploadFile = null;
            window.jsonlData = null;
            return;
        }

        const fileName = file.name.toLowerCase();
        
        // JSONL 파일 처리
        if (fileName.endsWith('.jsonl') || fileName.endsWith('.json')) {
            window.uploadFile = null; // Clear binary file
            
            const reader = new FileReader();
            reader.onload = function(event) {
                try {
                    const content = event.target.result;
                    const lines = content.trim().split('\n').filter(line => line.trim());
                    
                    // 각 줄이 유효한 JSON인지 확인
                    const validLines = [];
                    for (let i = 0; i < lines.length; i++) {
                        try {
                            const parsed = JSON.parse(lines[i]);
                            if (parsed.title && parsed.content) {
                                validLines.push(parsed);
                            }
                        } catch (parseError) {
                            console.warn(`Line ${i + 1} is not valid JSON:`, lines[i]);
                        }
                    }
                    
                    if (validLines.length === 0) {
                        alert(t('noValidArticles'));
                        uploadBtn.disabled = true;
                        return;
                    }
                    
                    // 미리보기 표시
                    previewContent.textContent = validLines.slice(0, 3).map(item => 
                        JSON.stringify(item, null, 2)
                    ).join('\n\n') + (validLines.length > 3 ? '\n\n...' : '');
                    
                    totalLines.textContent = validLines.length;
                    preview.classList.remove('hidden');
                    uploadBtn.disabled = false;
                    
                    // 파일 데이터를 전역 변수에 저장
                    window.jsonlData = validLines;
                    
                } catch (error) {
                    alert(t('fileReadError'));
                    uploadBtn.disabled = true;
                }
            };
            reader.readAsText(file);
        } 
        // 일반 문서 파일 (PDF, DOCX, XLSX 등) 처리
        else {
            window.jsonlData = null; // Clear JSONL data
            window.uploadFile = file;
            
            previewContent.textContent = `File: ${file.name}\nSize: ${(file.size / 1024).toFixed(2)} KB\nType: ${file.type || 'Unknown'}`;
            totalLines.textContent = "1 (Document Upload)";
            preview.classList.remove('hidden');
            uploadBtn.disabled = false;
        }
    });
}


// Function to handle single file upload (PDF, XLSX, DOCX, etc.)
async function handleSingleFileUpload(file) {
    const uploadBtn = document.getElementById('bulk-upload-btn');
    const uploadLog = document.getElementById('upload-log');
    const progressContainer = document.getElementById('upload-progress');
    const currentItem = document.getElementById('current-item');
    const totalProgress = document.getElementById('total-progress');
    const currentProgress = document.getElementById('current-progress');
    
    // UI Init
    const originalText = uploadBtn.innerHTML;
    uploadBtn.disabled = true;
    uploadBtn.innerHTML = `<div class="flex items-center justify-center"><div class="spinner mr-2"></div> ${t('uploading')}</div>`;
    progressContainer.classList.remove('hidden');
    uploadLog.innerHTML = '';
    
    // Reset progress counters for single file
    totalProgress.textContent = "1";
    currentProgress.textContent = "1";

    currentItem.innerHTML = `
        <div class="flex items-center">
            <div class="spinner mr-2"></div>
            <span>Uploading: <strong>${escapeHtml(file.name)}</strong></span>
        </div>
    `;

    const formData = new FormData();
    formData.append('file', file);

    const token = getJWTToken();
    if (!token) {
        alert(t('loginRequired'));
        return;
    }

    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/articles/upload`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            },
            body: formData
        });

        if (!response.ok) {
            let errMsg = `Status: ${response.status}`;
            try {
                const errData = await response.text(); // Get text first in case it isn't JSON
                errMsg += ` - ${errData}`;
            } catch (e) {}
            throw new Error(errMsg);
        }

        const result = await response.json();
        
        // Success
        currentItem.innerHTML = `<span class="text-green-600 font-bold">Upload Complete!</span>`;
        
        const logEntry = document.createElement('div');
        logEntry.className = 'text-sm text-green-600 mb-1';
        logEntry.textContent = `✓ Successfully uploaded ${file.name}`;
        uploadLog.appendChild(logEntry);
        
        alert(t('articleAddedSuccess'));

    } catch (error) {
        console.error('Upload Error:', error);
        currentItem.innerHTML = `<span class="text-red-600 font-bold">Upload Failed</span>`;
        
        const logEntry = document.createElement('div');
        logEntry.className = 'text-sm text-red-600 mb-1';
        logEntry.textContent = `✗ Error: ${error.message}`;
        uploadLog.appendChild(logEntry);
        
        alert(t('articleAddError'));
    } finally {
        uploadBtn.disabled = false;
        uploadBtn.innerHTML = originalText;
    }
}

// 대량 업로드 처리 - 각 아티클을 개별 WebSocket으로 처리
async function handleBulkUpload() {
    // 로그인 상태 확인
    if (!isLoggedIn()) {
        alert('로그인이 필요합니다.');
        showAuthModal('login');
        return;
    }
    
    // Check if we have a single binary file to upload
    if (window.uploadFile) {
        await handleSingleFileUpload(window.uploadFile);
        return;
    }

    if (!window.jsonlData || window.jsonlData.length === 0) {
        alert(t('noUploadData'));
        return;
    }
    
    const uploadBtn = document.getElementById('bulk-upload-btn');
    const progressContainer = document.getElementById('upload-progress');
    const progressBar = document.getElementById('progress-bar');
    const currentProgress = document.getElementById('current-progress');
    const totalProgress = document.getElementById('total-progress');
    const currentItem = document.getElementById('current-item');
    const uploadLog = document.getElementById('upload-log');
    
    // UI 초기화
    const originalText = uploadBtn.innerHTML;
    uploadBtn.disabled = true;
    uploadBtn.innerHTML = `<div class="flex items-center justify-center"><div class="spinner mr-2"></div> ${t('uploading')}</div>`;
    progressContainer.classList.remove('hidden');
    
    const totalItems = window.jsonlData.length;
    totalProgress.textContent = totalItems;
    uploadLog.innerHTML = '';
    
    let successCount = 0;
    let errorCount = 0;
    
    // 각 아티클을 개별 WebSocket으로 순차 처리
    for (let i = 0; i < totalItems; i++) {
        const article = window.jsonlData[i];
        currentProgress.textContent = i + 1;
        
        // 아티클 데이터 준비
        const articleData = {
            title: article.title,
            content: article.content,
            original_url: article.original_url || '',
            author: article.author || ''
        };
        
        // Handle created_date if provided
        if (article.created_date) {
            try {
                const date = new Date(article.created_date);
                if (!isNaN(date.getTime())) {
                    articleData.created_date = date.toISOString();
                }
            } catch (dateError) {
                console.warn(`Invalid date format for article "${article.title}": ${article.created_date}`);
            }
        }
        
        // 현재 처리 중인 아티클 표시
        currentItem.innerHTML = `
            <div class="flex items-center">
                <div class="spinner mr-2"></div>
                <span>처리 중: <strong>${escapeHtml(article.title.substring(0, 50))}...</strong></span>
            </div>
        `;
        
        try {
            // 개별 아티클에 대해 WebSocket 요청 (폴백 포함)
            await processIndividualArticleWithWebSocket(articleData, article.title);
            
            successCount++;
            
            // 성공 로그 추가
            const logEntry = document.createElement('div');
            logEntry.className = 'text-sm text-green-600 mb-1';
            logEntry.innerHTML = `✓ ${escapeHtml(article.title.substring(0, 60))}${article.title.length > 60 ? '...' : ''}`;
            uploadLog.appendChild(logEntry);
            
        } catch (error) {
            errorCount++;
            console.error(`Failed to upload article: ${article.title}`, error);
            
            // 실패 로그 추가
            const logEntry = document.createElement('div');
            logEntry.className = 'text-sm text-red-600 mb-1';
            logEntry.innerHTML = `✗ ${escapeHtml(article.title.substring(0, 60))}${article.title.length > 60 ? '...' : ''} (오류: ${error.message})`;
            uploadLog.appendChild(logEntry);
        }
        
        // 전체 진행률 업데이트
        const progress = ((i + 1) / totalItems) * 100;
        progressBar.style.width = `${progress}%`;
        
        // 로그 스크롤을 하단으로
        uploadLog.scrollTop = uploadLog.scrollHeight;
        
        // 각 요청 사이에 짧은 지연 추가 (서버 부하 방지)
        await new Promise(resolve => setTimeout(resolve, 500));
    }
    
    // 완료 처리
    currentItem.innerHTML = `
        <div class="text-green-600 font-medium">
            ${t('uploadComplete', { success: successCount, failed: errorCount })}
        </div>
    `;
    
    uploadBtn.disabled = false;
    uploadBtn.innerHTML = originalText;
    
    if (successCount > 0) {
        // 성공적으로 업로드된 아티클이 있으면 검색 화면으로 이동 제안
        if (confirm(`${successCount}${t('moveToSearch')}`)) {
            showView('search-view');
        }
    }
    
    // 파일 입력 초기화
    document.getElementById('jsonl-file').value = '';
    document.getElementById('file-preview').classList.add('hidden');
    window.jsonlData = null;
}

// 개별 아티클에 대한 WebSocket 처리 (실시간 진행률 표시 포함)
async function processIndividualArticleWithWebSocket(articleData, articleTitle) {
    return new Promise((resolve, reject) => {
        // JWT 토큰 확인
        const token = getJWTToken();
        if (!token) {
            reject(new Error('Authentication required'));
            return;
        }
        
        // WebSocket 연결 설정 - 토큰을 쿼리 파라미터로 전달
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${window.location.host}/api/v1/articles/ws?token=${encodeURIComponent(token)}`;
        const ws = new WebSocket(wsUrl);
        
        let hasCompleted = false;
        
        // WebSocket 연결 열림
        ws.onopen = function() {
            console.log('WebSocket connection opened for article:', articleTitle);
            // 아티클 추가 요청 전송
            ws.send(JSON.stringify(articleData));
        };
        
        // WebSocket 메시지 수신
        ws.onmessage = function(event) {
            const message = JSON.parse(event.data);
            console.log('Received message for article:', articleTitle, message);
            
            switch (message.type) {
                case 'status':
                    // 현재 상태 업데이트
                    const currentItem = document.getElementById('current-item');
                    if (currentItem) {
                        currentItem.innerHTML = `
                            <div class="flex items-center">
                                <div class="spinner mr-2"></div>
                                <span><strong>${escapeHtml(articleTitle.substring(0, 30))}...</strong> - ${message.data}</span>
                            </div>
                        `;
                    }
                    break;
                    
                case 'progress':
                    // 진행률 업데이트
                    const data = message.data;
                    const currentItem2 = document.getElementById('current-item');
                    if (currentItem2) {
                        currentItem2.innerHTML = `
                            <div class="flex items-center">
                                <div class="spinner mr-2"></div>
                                <span><strong>${escapeHtml(articleTitle.substring(0, 30))}...</strong> - ${data.step} (${Math.round(data.percent)}%)</span>
                            </div>
                        `;
                    }
                    break;
                    
                case 'success':
                    console.log('Article processed successfully:', articleTitle);
                    break;
                    
                case 'done':
                    if (!hasCompleted) {
                        hasCompleted = true;
                        ws.close();
                        resolve();
                    }
                    break;
                    
                case 'error':
                    if (!hasCompleted) {
                        hasCompleted = true;
                        ws.close();
                        reject(new Error(message.data));
                    }
                    break;
            }
        };
        
        // WebSocket 오류 처리
        ws.onerror = function(error) {
            console.error('WebSocket error for article:', articleTitle, error);
            console.error('WebSocket URL was:', wsUrl);
            console.error('WebSocket readyState:', ws.readyState);
            if (!hasCompleted) {
                hasCompleted = true;
                // WebSocket 실패 시 HTTP로 폴백
                fallbackToHttpUpload(articleData)
                    .then(resolve)
                    .catch(reject);
            }
        };
        
        // WebSocket 연결 종료
        ws.onclose = function(event) {
            console.log('WebSocket connection closed for article:', articleTitle);
            console.log('Close code:', event.code, 'Close reason:', event.reason);
            console.log('WebSocket URL was:', wsUrl);
            if (!hasCompleted) {
                hasCompleted = true;
                console.log('WebSocket closed unexpectedly, falling back to HTTP');
                // 정상적인 완료가 아닌 경우 HTTP로 폴백
                fallbackToHttpUpload(articleData)
                    .then(resolve)
                    .catch(reject);
            }
        };
        
        // 타임아웃 설정 (5분)
        setTimeout(() => {
            if (!hasCompleted) {
                hasCompleted = true;
                ws.close();
                reject(new Error('WebSocket timeout'));
            }
        }, 5 * 60 * 1000);
    });
}

// HTTP 폴백 함수
async function fallbackToHttpUpload(articleData) {
    console.log('Falling back to HTTP upload for article:', articleData.title);
    
    // JWT 토큰 확인
    const token = getJWTToken();
    if (!token) {
        throw new Error('Authentication required');
    }
    
    const response = await fetch(`${API_BASE_URL}/api/v1/articles`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(articleData)
    });
    
    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    return await response.json();
}

// WebSocket을 이용한 아티클 추가 - 실시간 진행률 표시
async function handleWebSocketArticleAddition(articleData, button, originalText) {
    return new Promise((resolve, reject) => {
        // Create progress UI elements
        const progressContainer = createArticleProgressUI(articleData.title, button);
        
        try {
            // JWT 토큰 확인
            const token = getJWTToken();
            if (!token) {
                reject(new Error('Authentication required'));
                return;
            }
            
            // WebSocket 연결 생성 - 토큰을 쿼리 파라미터로 전달
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/api/v1/articles/ws?token=${encodeURIComponent(token)}`;
            const ws = new WebSocket(wsUrl);
            
            let completed = false;

            ws.onopen = function() {
                console.log('WebSocket connected for article addition');
                // 아티클 추가 요청 전송
                ws.send(JSON.stringify(articleData));
            };

            ws.onmessage = function(event) {
                try {
                    const message = JSON.parse(event.data);
                    
                    switch(message.type) {
                        case 'status':
                            updateArticleProgressStatus(message.data);
                            break;
                        case 'progress':
                            updateArticleProgress(message.data);
                            break;
                        case 'success':
                            completed = true;
                            showArticleAdditionSuccess(message.data);
                            ws.close();
                            setTimeout(() => {
                                removeArticleProgressUI();
                                resolve(message.data);
                            }, 1500);
                            break;
                        case 'error':
                            completed = true;
                            showArticleAdditionError(message.data);
                            ws.close();
                            setTimeout(() => {
                                removeArticleProgressUI();
                                reject(new Error(message.data));
                            }, 1500);
                            break;
                        case 'done':
                            if (!completed) {
                                completed = true;
                                ws.close();
                                setTimeout(() => {
                                    removeArticleProgressUI();
                                    resolve();
                                }, 1500);
                            }
                            break;
                        default:
                            console.log('Unknown message type:', message.type);
                    }
                } catch (error) {
                    console.error('Error processing WebSocket message:', error);
                    reject(error);
                }
            };

            ws.onerror = function(error) {
                console.error('WebSocket error:', error);
                console.error('WebSocket URL was:', wsUrl);
                console.error('WebSocket readyState:', ws.readyState);
                removeArticleProgressUI();
                reject(new Error('WebSocket connection failed'));
            };

            ws.onclose = function(event) {
                console.log('WebSocket connection closed');
                console.log('Close code:', event.code, 'Close reason:', event.reason);
                console.log('WebSocket URL was:', wsUrl);
                if (!completed) {
                    removeArticleProgressUI();
                    reject(new Error('WebSocket connection closed unexpectedly'));
                }
            };

        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            removeArticleProgressUI();
            reject(error);
        }
    });
}

// 아티클 추가 진행률 UI 생성
function createArticleProgressUI(title, button) {
    // 버튼 아래에 진행률 표시 컨테이너 생성
    const form = document.getElementById('add-article-form');
    const progressContainer = document.createElement('div');
    progressContainer.id = 'article-progress-container';
    progressContainer.className = 'mt-4 bg-blue-50 border border-blue-200 rounded-lg p-4';
    
    progressContainer.innerHTML = `
        <div class="mb-3">
            <div class="flex justify-between items-center mb-2">
                <h4 class="font-medium text-blue-900">Processing: "${escapeHtml(title.substring(0, 50))}${title.length > 50 ? '...' : ''}"</h4>
                <span id="article-progress-percent" class="text-sm text-blue-600">0%</span>
            </div>
            <div class="w-full bg-blue-200 rounded-full h-2">
                <div id="article-progress-bar" class="bg-blue-600 h-2 rounded-full transition-all duration-300" style="width: 0%"></div>
            </div>
        </div>
        <div id="article-progress-status" class="text-sm text-blue-700">
            Initializing...
        </div>
        <div id="article-progress-steps" class="mt-3 space-y-1 text-xs text-blue-600">
            <!-- Progress steps will be added here -->
        </div>
    `;
    
    form.appendChild(progressContainer);
    return progressContainer;
}

// 아티클 추가 진행률 업데이트
function updateArticleProgress(progressData) {
    const progressBar = document.getElementById('article-progress-bar');
    const progressPercent = document.getElementById('article-progress-percent');
    const progressSteps = document.getElementById('article-progress-steps');
    
    if (progressBar && progressPercent) {
        const percent = Math.round(progressData.percent);
        progressBar.style.width = `${percent}%`;
        progressPercent.textContent = `${percent}%`;
    }
    
    if (progressSteps) {
        const stepElement = document.createElement('div');
        stepElement.className = 'flex items-center';
        stepElement.innerHTML = `
            <div class="w-2 h-2 bg-blue-500 rounded-full mr-2"></div>
            <span>${escapeHtml(progressData.step)} (${progressData.progress}/${progressData.total})</span>
        `;
        progressSteps.appendChild(stepElement);
        
        // 스크롤을 맨 아래로
        progressSteps.scrollTop = progressSteps.scrollHeight;
    }
}

// 아티클 추가 상태 업데이트
function updateArticleProgressStatus(status) {
    const statusElement = document.getElementById('article-progress-status');
    if (statusElement) {
        statusElement.textContent = status;
    }
}

// 아티클 추가 성공 표시
function showArticleAdditionSuccess(data) {
    const statusElement = document.getElementById('article-progress-status');
    const progressBar = document.getElementById('article-progress-bar');
    const progressPercent = document.getElementById('article-progress-percent');
    
    if (statusElement) {
        statusElement.innerHTML = `
            <div class="flex items-center text-green-700">
                <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                </svg>
                Article added successfully!
            </div>
        `;
    }
    
    if (progressBar && progressPercent) {
        progressBar.style.width = '100%';
        progressBar.className = 'bg-green-600 h-2 rounded-full transition-all duration-300';
        progressPercent.textContent = '100%';
        progressPercent.className = 'text-sm text-green-600';
    }
}

// 아티클 추가 오류 표시
function showArticleAdditionError(error) {
    const statusElement = document.getElementById('article-progress-status');
    const progressBar = document.getElementById('article-progress-bar');
    
    if (statusElement) {
        statusElement.innerHTML = `
            <div class="flex items-center text-red-700">
                <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                </svg>
                Error: ${escapeHtml(error)}
            </div>
        `;
    }
    
    if (progressBar) {
        progressBar.className = 'bg-red-600 h-2 rounded-full transition-all duration-300';
    }
}

// 아티클 추가 진행률 UI 제거
function removeArticleProgressUI() {
    const progressContainer = document.getElementById('article-progress-container');
    if (progressContainer) {
        progressContainer.remove();
    }
}
