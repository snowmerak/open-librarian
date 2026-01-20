// 아티클 추가 폼 이벤트 리스너 설정
function initArticleForm() {
    // 아티클 추가 폼 처리
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
            const metadata = {
                title: formData.get('title'),
                original_url: formData.get('original_url') || '',
                author: formData.get('author') || ''
            };
            
            // Handle created_date conversion from datetime-local to RFC3339
            const createdDateValue = formData.get('created_date');
            if (createdDateValue) {
                const localDate = new Date(createdDateValue);
                metadata.created_date = localDate.toISOString();
            }

            // Check if there are files in the queue
            if (window.uploadQueue && window.uploadQueue.length > 0) {
                 await handleUnifiedUpload(window.uploadQueue, metadata);
                 e.target.reset();
                 window.uploadQueue = []; // Clear queue
                 
                 // Show completion in the log
                 const log = document.getElementById('upload-log');
                 if(log) {
                     const successMsg = document.createElement('div');
                     successMsg.className = "text-green-600 font-bold mt-2 p-2 bg-green-50 rounded";
                     successMsg.innerText = "All operations completed successfully.";
                     log.appendChild(successMsg);
                     log.scrollTop = log.scrollHeight;
                 }
                 
                 // Optional: Delay and redirect? 
                 // For now, let user see the log.
            } else {
                // No files, try Single Text Upload
                const content = formData.get('content');
                if (!content || !content.trim()) {
                    alert(t('contentRequired') || "Content is required if no file is selected.");
                    throw new Error("Missing content"); // Stop execution
                }

                // Normal Single Article Logic
                const articleData = { ...metadata, content, title: metadata.title };
                
                // Try WebSocket first, fallback to regular HTTP if WebSocket fails
                try {
                    await handleWebSocketArticleAddition(articleData, button, originalText);
                    e.target.reset();
                    showView('search-view'); 
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
            }

        } catch (error) {
            console.error('Article submission error:', error);
            if(error.message !== "Missing content") alert(t('articleAddError'));
        } finally {
            button.disabled = false;
            button.innerHTML = originalText;
        }
    });

    // 파일 선택 처리
    const jsonlFileInput = document.getElementById('jsonl-file');
    if (jsonlFileInput) {
        jsonlFileInput.addEventListener('change', async function(e) {
            const files = Array.from(e.target.files);
            
            window.uploadQueue = [];
            
            if (files.length === 0) return;

            for (const file of files) {
                const fileName = file.name.toLowerCase();
                    // Just push to queue, parsing JSONL if needed
                if (fileName.endsWith('.jsonl') || fileName.endsWith('.json')) {
                     await new Promise((resolve) => {
                        const reader = new FileReader();
                        reader.onload = function(event) {
                            try {
                                const content = event.target.result;
                                const lines = content.trim().split('\n').filter(line => line.trim());
                                for (let i = 0; i < lines.length; i++) {
                                     try {
                                         const parsed = JSON.parse(lines[i]);
                                         if (parsed.title && parsed.content) {
                                             window.uploadQueue.push({ type: 'data', data: parsed, name: parsed.title });
                                         }
                                     } catch (parseError) {}
                                }
                            } catch (err) {}
                            resolve();
                        };
                        reader.readAsText(file);
                     });
                } else {
                     window.uploadQueue.push({ type: 'file', file: file, name: file.name });
                }
            }
            
             // Auto-populate title if single file
             const titleInput = document.getElementById('title');
             if(titleInput && !titleInput.value && window.uploadQueue.length === 1) {
                 titleInput.value = window.uploadQueue[0].name.replace(/\.[^/.]+$/, "");
             }
        });
    }
}

// Function to upload a single file (returns Promise)
async function uploadSingleFile(file, metadata = {}) {
    const formData = new FormData();
    formData.append('file', file);
    
    // Add metadata
    if (metadata.author) formData.append('author', metadata.author);
    if (metadata.original_url) formData.append('original_url', metadata.original_url);
    if (metadata.created_date) formData.append('created_date', metadata.created_date);

    const token = getJWTToken();
    if (!token) throw new Error('Login Required');

    const response = await fetch(`${API_BASE_URL}/api/v1/articles/upload`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
        body: formData
    });

    if (!response.ok) {
        let errMsg = `Status: ${response.status}`;
        try { errMsg += ` - ${await response.text()}`; } catch (e) {}
        throw new Error(errMsg);
    }
    return await response.json();
}

// 대량 업로드 처리
async function handleUnifiedUpload(queue, metadata) {
    if (!queue || queue.length === 0) return;
    
    const progressContainer = document.getElementById('upload-progress');
    const progressBar = document.getElementById('progress-bar');
    const currentProgress = document.getElementById('current-progress');
    const totalProgress = document.getElementById('total-progress');
    const currentItem = document.getElementById('current-item');
    const uploadLog = document.getElementById('upload-log');
    
    // UI Init
    if (progressContainer) progressContainer.classList.remove('hidden');
    if (totalProgress) totalProgress.textContent = queue.length;
    if (uploadLog) uploadLog.innerHTML = '';
    
    let successCount = 0;
    let errorCount = 0;
    
    for (let i = 0; i < queue.length; i++) {
        const item = queue[i];
        if (currentProgress) currentProgress.textContent = i + 1;
        
        const itemName = item.type === 'file' ? item.name : (item.name || 'Untitled');
        if (currentItem) {
            currentItem.innerHTML = `
                <div class="flex items-center">
                    <div class="spinner mr-2"></div>
                    <span>Processing: <strong>${escapeHtml(itemName.substring(0, 50))}...</strong></span>
                </div>
            `;
        }
        
        try {
            if (item.type === 'data') {
                const articleData = {
                    title: item.data.title || metadata.title || 'Untitled',
                    content: item.data.content || '',
                    original_url: item.data.original_url || metadata.original_url || '',
                    author: item.data.author || metadata.author || ''
                };
                const d = item.data.created_date || metadata.created_date;
                 if (d) {
                    const date = new Date(d);
                    if (!isNaN(date.getTime())) articleData.created_date = date.toISOString();
                }
                
                await processIndividualArticleWithWebSocket(articleData, articleData.title);
            } else if (item.type === 'file') {
                await uploadSingleFile(item.file, metadata);
            }

            successCount++;
            if (uploadLog) {
                const div = document.createElement('div');
                div.className = 'text-sm text-green-600 mb-1';
                div.innerHTML = `✓ ${escapeHtml(itemName.substring(0, 60))}`;
                uploadLog.appendChild(div);
            }
        } catch (error) {
            errorCount++;
            console.error(`Failed to upload ${itemName}`, error);
            if (uploadLog) {
                const div = document.createElement('div');
                div.className = 'text-sm text-red-600 mb-1';
                div.innerHTML = `✗ ${escapeHtml(itemName.substring(0, 60))} (Error: ${error.message})`;
                uploadLog.appendChild(div);
            }
        }
        
        if (progressBar) progressBar.style.width = `${((i + 1) / queue.length) * 100}%`;
        if (uploadLog) uploadLog.scrollTop = uploadLog.scrollHeight;
        if (item.type === 'data') await new Promise(r => setTimeout(r, 300));
    }
    
    if (currentItem) {
        currentItem.innerHTML = `<div class="text-green-600 font-medium">${t('uploadComplete', { success: successCount, failed: errorCount })}</div>`;
    }
    
    const fileInput = document.getElementById('jsonl-file');
    if (fileInput) fileInput.value = '';
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
