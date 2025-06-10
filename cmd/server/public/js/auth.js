// 인증 관련 기능
let currentUser = null;

// 페이지 로드 시 인증 상태 확인
document.addEventListener('DOMContentLoaded', function() {
    checkAuthStatus();
    initAuthEventListeners();
});

// 인증 이벤트 리스너 초기화
function initAuthEventListeners() {
    // 로그인 폼 이벤트
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    
    // 회원가입 폼 이벤트
    document.getElementById('signup-form').addEventListener('submit', handleSignup);
    
    // 모달 외부 클릭 시 닫기
    document.getElementById('auth-modal').addEventListener('click', function(e) {
        if (e.target === this) {
            closeAuthModal();
        }
    });
    
    // ESC 키로 모달 닫기
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && !document.getElementById('auth-modal').classList.contains('hidden')) {
            closeAuthModal();
        }
    });
}

// 인증 상태 확인
async function checkAuthStatus() {
    const token = getJWTToken();
    if (!token) {
        updateAuthUI(false);
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/users/me`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });
        
        if (response.ok) {
            const user = await response.json();
            currentUser = user;
            updateAuthUI(true, user);
        } else {
            // 토큰이 유효하지 않음
            setJWTToken(null);
            currentUser = null;
            updateAuthUI(false);
        }
    } catch (error) {
        console.error('Failed to check auth status:', error);
        setJWTToken(null);
        currentUser = null;
        updateAuthUI(false);
    }
}

// 인증 UI 업데이트
function updateAuthUI(isLoggedIn, user = null) {
    const loggedOutSection = document.getElementById('logged-out-section');
    const loggedInSection = document.getElementById('logged-in-section');
    const myArticlesLink = document.getElementById('my-articles-link');
    
    if (isLoggedIn && user) {
        loggedOutSection.classList.add('hidden');
        loggedInSection.classList.remove('hidden');
        
        document.getElementById('user-username').textContent = user.username || user.email;
        document.getElementById('user-email').textContent = user.email;
        
        // Show the "내 아티클" link when logged in
        if (myArticlesLink) {
            myArticlesLink.style.display = 'flex';
        }
    } else {
        loggedOutSection.classList.remove('hidden');
        loggedInSection.classList.add('hidden');
        
        // Hide the "내 아티클" link when logged out
        if (myArticlesLink) {
            myArticlesLink.style.display = 'none';
        }
    }
}

// 인증 모달 표시
function showAuthModal(mode = 'login') {
    const modal = document.getElementById('auth-modal');
    const title = document.getElementById('auth-modal-title');
    
    // 모달 초기화
    hideAuthMessages();
    clearAuthForms();
    
    // 모드에 따라 표시
    if (mode === 'login') {
        title.textContent = t('login');
        document.getElementById('login-form-container').classList.remove('hidden');
        document.getElementById('signup-form-container').classList.add('hidden');
    } else {
        title.textContent = t('signup');
        document.getElementById('login-form-container').classList.add('hidden');
        document.getElementById('signup-form-container').classList.remove('hidden');
    }
    
    modal.classList.remove('hidden');
    
    // 첫 번째 입력 필드에 포커스
    setTimeout(() => {
        const firstInput = modal.querySelector('input[type="email"], input[type="text"]');
        if (firstInput) firstInput.focus();
    }, 100);
}

// 인증 모달 닫기
function closeAuthModal() {
    const modal = document.getElementById('auth-modal');
    modal.classList.add('hidden');
    hideAuthMessages();
    clearAuthForms();
}

// 인증 모드 전환
function switchAuthMode(mode) {
    hideAuthMessages();
    clearAuthForms();
    
    const title = document.getElementById('auth-modal-title');
    const loginContainer = document.getElementById('login-form-container');
    const signupContainer = document.getElementById('signup-form-container');
    
    if (mode === 'login') {
        title.textContent = t('login');
        loginContainer.classList.remove('hidden');
        signupContainer.classList.add('hidden');
        
        setTimeout(() => {
            document.getElementById('login-email').focus();
        }, 100);
    } else {
        title.textContent = t('signup');
        loginContainer.classList.add('hidden');
        signupContainer.classList.remove('hidden');
        
        setTimeout(() => {
            document.getElementById('signup-username').focus();
        }, 100);
    }
}

// 로그인 처리
async function handleLogin(e) {
    e.preventDefault();
    
    const email = document.getElementById('login-email').value.trim();
    const password = document.getElementById('login-password').value;
    
    // 유효성 검사
    if (!email) {
        showAuthError(t('emailRequired'));
        return;
    }
    
    if (!password) {
        showAuthError(t('passwordRequired'));
        return;
    }
    
    if (!isValidEmail(email)) {
        showAuthError(t('invalidEmail'));
        return;
    }
    
    showAuthLoading(true);
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/users/auth`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                email: email,
                password: password
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // 로그인 성공
            setJWTToken(data.token);
            currentUser = data.user;
            
            showAuthSuccess(t('loginSuccess'));
            
            setTimeout(() => {
                closeAuthModal();
                updateAuthUI(true, data.user);
            }, 1000);
        } else {
            // 로그인 실패
            showAuthError(data.message || t('loginFailed'));
        }
    } catch (error) {
        console.error('Login error:', error);
        showAuthError(t('loginFailed'));
    } finally {
        showAuthLoading(false);
    }
}

// 회원가입 처리
async function handleSignup(e) {
    e.preventDefault();
    
    const username = document.getElementById('signup-username').value.trim();
    const email = document.getElementById('signup-email').value.trim();
    const password = document.getElementById('signup-password').value;
    const confirmPassword = document.getElementById('signup-confirm-password').value;
    
    // 유효성 검사
    if (!username) {
        showAuthError(t('usernameRequired'));
        return;
    }
    
    if (!email) {
        showAuthError(t('emailRequired'));
        return;
    }
    
    if (!password) {
        showAuthError(t('passwordRequired'));
        return;
    }
    
    if (!isValidEmail(email)) {
        showAuthError(t('invalidEmail'));
        return;
    }
    
    if (password !== confirmPassword) {
        showAuthError(t('passwordMismatch'));
        return;
    }
    
    if (password.length < 6) {
        showAuthError('비밀번호는 최소 6자 이상이어야 합니다.');
        return;
    }
    
    showAuthLoading(true);
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/users/`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                username: username,
                email: email,
                password: password
            })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // 회원가입 성공
            showAuthSuccess(t('signupSuccess'));
            
            setTimeout(() => {
                switchAuthMode('login');
                // 이메일 자동 입력
                document.getElementById('login-email').value = email;
                document.getElementById('login-password').focus();
            }, 1500);
        } else {
            // 회원가입 실패
            if (response.status === 409) {
                showAuthError('이미 존재하는 이메일 또는 사용자명입니다.');
            } else {
                showAuthError(data.message || t('signupFailed'));
            }
        }
    } catch (error) {
        console.error('Signup error:', error);
        showAuthError(t('signupFailed'));
    } finally {
        showAuthLoading(false);
    }
}

// 로그아웃 처리
async function logout() {
    setJWTToken(null);
    currentUser = null;
    updateAuthUI(false);
    
    // 성공 메시지 표시 (선택사항)
    if (typeof showNotification === 'function') {
        showNotification(t('logoutSuccess'), 'success');
    } else {
        console.log(t('logoutSuccess'));
    }
}

// 현재 사용자 정보 가져오기
function getCurrentUser() {
    return currentUser;
}

// 로그인 상태 확인
function isLoggedIn() {
    return currentUser !== null && getJWTToken() !== null;
}

// 이메일 유효성 검사
function isValidEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

// 인증 폼 초기화
function clearAuthForms() {
    document.getElementById('login-form').reset();
    document.getElementById('signup-form').reset();
}

// 로딩 상태 표시
function showAuthLoading(show) {
    const loading = document.getElementById('auth-loading');
    const loginForm = document.getElementById('login-form-container');
    const signupForm = document.getElementById('signup-form-container');
    
    if (show) {
        loading.classList.remove('hidden');
        loginForm.classList.add('hidden');
        signupForm.classList.add('hidden');
    } else {
        loading.classList.add('hidden');
        // 현재 모드에 따라 적절한 폼 표시
        const title = document.getElementById('auth-modal-title').textContent;
        if (title === t('login')) {
            loginForm.classList.remove('hidden');
        } else {
            signupForm.classList.remove('hidden');
        }
    }
}

// 에러 메시지 표시
function showAuthError(message) {
    const errorDiv = document.getElementById('auth-error');
    const errorMessage = document.getElementById('auth-error-message');
    
    errorMessage.textContent = message;
    errorDiv.classList.remove('hidden');
    
    // 성공 메시지 숨기기
    document.getElementById('auth-success').classList.add('hidden');
}

// 성공 메시지 표시
function showAuthSuccess(message) {
    const successDiv = document.getElementById('auth-success');
    const successMessage = document.getElementById('auth-success-message');
    
    successMessage.textContent = message;
    successDiv.classList.remove('hidden');
    
    // 에러 메시지 숨기기
    document.getElementById('auth-error').classList.add('hidden');
}

// 메시지 숨기기
function hideAuthMessages() {
    document.getElementById('auth-error').classList.add('hidden');
    document.getElementById('auth-success').classList.add('hidden');
}

// 전역 함수로 등록
window.showAuthModal = showAuthModal;
window.closeAuthModal = closeAuthModal;
window.switchAuthMode = switchAuthMode;
window.logout = logout;
window.getCurrentUser = getCurrentUser;
window.isLoggedIn = isLoggedIn;
window.checkAuthStatus = checkAuthStatus;
