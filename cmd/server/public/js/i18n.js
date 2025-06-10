// 다국어 지원 시스템
const TRANSLATIONS = {
    en: {
        appTitle: 'Open Librarian',
        searchPlaceholder: 'Ask anything you\'re curious about...',
        searchWelcomeTitle: 'Search for knowledge',
        searchWelcomeDesc: 'We provide accurate answers based on stored articles.',
        searchRecent: 'Recent searches',
        clearAll: 'Clear all',
        noSearchHistory: 'No search history',
        searchButton: 'Search',
        addArticle: 'Add Article',
        integratedSearch: 'Integrated Search',
        myArticles: 'My Articles',
        
        // Authentication
        login: 'Login',
        logout: 'Logout',
        signup: 'Sign Up',
        email: 'Email',
        username: 'Username',
        password: 'Password',
        confirmPassword: 'Confirm Password',
        loginButton: 'Login',
        signupButton: 'Sign Up',
        emailPlaceholder: 'your@email.com',
        usernamePlaceholder: 'Enter your username',
        passwordPlaceholder: 'Enter your password',
        confirmPasswordPlaceholder: 'Confirm your password',
        noAccount: 'Don\'t have an account?',
        hasAccount: 'Already have an account?',
        signupLink: 'Sign up',
        loginLink: 'Login',
        loginSuccess: 'Login successful!',
        signupSuccess: 'Account created successfully!',
        logoutSuccess: 'Logged out successfully!',
        loginFailed: 'Login failed. Please check your credentials.',
        signupFailed: 'Sign up failed. Please try again.',
        emailRequired: 'Email is required',
        usernameRequired: 'Username is required',
        passwordRequired: 'Password is required',
        passwordMismatch: 'Passwords do not match',
        invalidEmail: 'Please enter a valid email address',
        
        // Common
        user: 'User',
        unknown: 'Unknown',
        home: 'Home',
        
        // Article Management
        articleAdd: 'Add Article',
        singleArticle: 'Single Article',
        bulkUpload: 'Bulk Upload (JSONL)',
        title: 'Title',
        content: 'Content',
        originalUrl: 'Original URL (Optional)',
        author: 'Author (Optional)',
        createdDate: 'Created Date (Optional)',
        createdDateHelp: 'If not provided, current time will be used',
        titlePlaceholder: 'Enter the article title',
        contentPlaceholder: 'Paste the full content of the article',
        urlPlaceholder: 'https://example.com/article',
        authorPlaceholder: 'John Doe',
        addArticleButton: 'Add Article',
        
        // JSONL Upload
        selectJsonlFile: 'Select JSONL File',
        jsonlFormat: 'Each line must be a JSON object in the following format:',
        filePreview: 'File Preview',
        articlesFound: 'articles found',
        uploadProgress: 'Upload Progress',
        uploadJsonlButton: 'Upload JSONL File',
        
        // Messages
        answerFor: 'Answer for',
        references: 'References',
        generating: 'Generating answer...',
        errorOccurred: 'An error occurred',
        errorMessage: 'Sorry, an error occurred while generating the answer. Please try again later.',
        processing: 'Processing...',
        uploading: 'Uploading...',
        uploadComplete: 'Upload complete! Success: {success}, Failed: {failed}',
        articleAddedSuccess: 'Article added successfully!',
        articleAddError: 'An error occurred while adding the article. Please try again.',
        noUploadData: 'No data to upload.',
        invalidFileType: 'Only JSONL or JSON files can be uploaded.',
        noValidArticles: 'No valid articles found. Please ensure each line contains a JSON object with title and content.',
        fileReadError: 'An error occurred while reading the file.',
        moveToSearch: 'articles have been successfully added. Would you like to go to the search screen?',
        
        // Date formatting
        createdAt: 'Created',
        today: 'Today',
        yesterday: 'Yesterday',
        daysAgo: 'days ago',
        weeksAgo: 'weeks ago',
        monthsAgo: 'months ago',
        yearsAgo: 'years ago',
        
        // Search History
        justNow: 'just now',
        minutesAgo: 'min ago',
        hoursAgo: 'h ago',
        removeResult: 'Remove result',
        searchingInProgress: 'Searching...',
        
        // Article Actions
        deleteArticle: 'Delete Article',
        deleteConfirm: 'Are you sure you want to delete this article?',
        deleteSuccess: 'Article deleted successfully',
        deleteError: 'Failed to delete article',
        deletePermissionDenied: 'Only the registrar can delete this article',
        
        // My Articles Page
        myArticlesPage: 'My Articles Management',
        myArticlesDesc: 'You can view and manage your registered articles',
        searchConditions: 'Search Conditions',
        startDate: 'Start Date',
        endDate: 'End Date',
        searchArticles: 'Search Articles',
        resetFilters: 'Reset Filters',
        totalArticles: 'Total {count} articles',
        articlesPerPage: 'articles per page',
        loginRequired: 'Login Required',
        loginRequiredDesc: 'Please log in first to manage your articles.',
        loginNow: 'Login Now',
        loadMore: 'Load More',
        
        // Article List
        articleTitle: 'Title',
        articleContent: 'Content',
        articleCreatedAt: 'Created Date',
        articleActions: 'Actions',
        editArticle: 'Edit',
        deleteArticle: 'Delete',
        viewOriginalUrl: 'View Original',
        noArticlesFound: 'No articles found',
        noArticlesFoundDesc: 'You haven\'t registered any articles yet.',
        
        // Loading States
        loadingArticles: 'Loading articles...',
        searching: 'Searching...',
        
        // Quick Date Selection
        quickSelect: 'Quick Select:',
        lastWeek: 'Last Week',
        lastMonth: 'Last Month',
        toToday: 'To Today',
        
        // Language Settings
        language: 'Language',
        languages: {
            en: 'English',
            ko: '한국어',
            zh: '中文',
            ja: '日本語',
            es: 'Español'
        }
    },
    ko: {
        appTitle: 'Open Librarian',
        searchPlaceholder: '궁금한 내용을 자유롭게 질문하세요...',
        searchWelcomeTitle: '지식을 검색해보세요',
        searchWelcomeDesc: '저장된 아티클들을 바탕으로 정확한 답변을 제공합니다.',
        searchRecent: '최근 검색',
        clearAll: '전체 삭제',
        noSearchHistory: '검색 기록이 없습니다',
        searchButton: '검색',
        addArticle: '아티클 추가',
        integratedSearch: '통합 검색',
        myArticles: '내 아티클',
        
        // Authentication
        login: '로그인',
        logout: '로그아웃',
        signup: '회원가입',
        email: '이메일',
        username: '사용자명',
        password: '비밀번호',
        confirmPassword: '비밀번호 확인',
        loginButton: '로그인',
        signupButton: '회원가입',
        emailPlaceholder: 'your@email.com',
        usernamePlaceholder: '사용자명을 입력하세요',
        passwordPlaceholder: '비밀번호를 입력하세요',
        confirmPasswordPlaceholder: '비밀번호를 다시 입력하세요',
        noAccount: '계정이 없으신가요?',
        hasAccount: '이미 계정이 있으신가요?',
        signupLink: '회원가입',
        loginLink: '로그인',
        loginSuccess: '로그인에 성공했습니다!',
        signupSuccess: '회원가입이 완료되었습니다!',
        logoutSuccess: '로그아웃되었습니다!',
        loginFailed: '로그인에 실패했습니다. 이메일과 비밀번호를 확인해주세요.',
        signupFailed: '회원가입에 실패했습니다. 다시 시도해주세요.',
        emailRequired: '이메일을 입력해주세요',
        usernameRequired: '사용자명을 입력해주세요',
        passwordRequired: '비밀번호를 입력해주세요',
        passwordMismatch: '비밀번호가 일치하지 않습니다',
        invalidEmail: '올바른 이메일 주소를 입력해주세요',
        
        // Common
        user: '사용자',
        unknown: '알 수 없음',
        home: '홈',
        
        // Authentication
        login: '로그인',
        logout: '로그아웃',
        signup: '회원가입',
        email: '이메일',
        username: '사용자 이름',
        password: '비밀번호',
        confirmPassword: '비밀번호 확인',
        loginButton: '로그인',
        signupButton: '회원가입',
        emailPlaceholder: 'your@email.com',
        usernamePlaceholder: '사용자 이름을 입력하세요',
        passwordPlaceholder: '비밀번호를 입력하세요',
        confirmPasswordPlaceholder: '비밀번호를 확인하세요',
        noAccount: '계정이 없으신가요?',
        hasAccount: '이미 계정이 있으신가요?',
        signupLink: '회원가입',
        loginLink: '로그인',
        loginSuccess: '로그인 성공!',
        signupSuccess: '계정이 성공적으로 생성되었습니다!',
        logoutSuccess: '로그아웃 성공!',
        loginFailed: '로그인 실패. 자격 증명을 확인하세요.',
        signupFailed: '회원가입 실패. 다시 시도해 주세요.',
        emailRequired: '이메일은 필수입니다',
        usernameRequired: '사용자 이름은 필수입니다',
        passwordRequired: '비밀번호는 필수입니다',
        passwordMismatch: '비밀번호가 일치하지 않습니다',
        invalidEmail: '유효한 이메일 주소를 입력하세요',
        
        // Article Management
        articleAdd: '아티클 추가',
        singleArticle: '단일 아티클',
        bulkUpload: '대량 업로드 (JSONL)',
        title: '제목',
        content: '내용',
        originalUrl: '원본 URL (선택)',
        author: '작성자 (선택)',
        createdDate: '작성일 (선택)',
        createdDateHelp: '미입력시 현재 시간으로 설정됩니다',
        titlePlaceholder: '아티클의 제목을 입력하세요',
        contentPlaceholder: '아티클의 전체 내용을 붙여넣으세요',
        urlPlaceholder: 'https://example.com/article',
        authorPlaceholder: '홍길동',
        addArticleButton: '아티클 추가하기',
        
        // JSONL Upload
        selectJsonlFile: 'JSONL 파일 선택',
        jsonlFormat: '각 줄은 다음 형식의 JSON 객체여야 합니다:',
        filePreview: '파일 미리보기',
        articlesFound: '개의 아티클이 발견되었습니다',
        uploadProgress: '업로드 진행률',
        uploadJsonlButton: 'JSONL 파일 업로드',
        
        // Messages
        answerFor: '에 대한 답변',
        references: '참고 자료',
        generating: '답변을 생성하고 있습니다...',
        errorOccurred: '오류가 발생했습니다',
        errorMessage: '죄송합니다. 답변 생성 중 오류가 발생했습니다. 잠시 후 다시 시도해 주세요.',
        processing: '처리 중...',
        uploading: '업로드 중...',
        uploadComplete: '업로드 완료! 성공: {success}개, 실패: {failed}개',
        articleAddedSuccess: '아티클이 성공적으로 추가되었습니다!',
        articleAddError: '아티클 추가 중 오류가 발생했습니다. 다시 시도해주세요.',
        noUploadData: '업로드할 데이터가 없습니다.',
        invalidFileType: 'JSONL 또는 JSON 파일만 업로드 가능합니다.',
        noValidArticles: '유효한 아티클이 없습니다. 각 줄에 title과 content가 포함된 JSON 객체가 있는지 확인하세요.',
        fileReadError: '파일 읽기 중 오류가 발생했습니다.',
        moveToSearch: '개의 아티클이 성공적으로 추가되었습니다. 검색 화면으로 이동하시겠습니까?',
        
        // Date formatting
        createdAt: '작성일',
        today: '오늘',
        yesterday: '어제',
        daysAgo: '일 전',
        weeksAgo: '주 전',
        monthsAgo: '개월 전',
        yearsAgo: '년 전',
        
        // Search History
        justNow: '방금 전',
        minutesAgo: '분 전',
        hoursAgo: '시간 전',
        removeResult: '결과 삭제',
        searchingInProgress: '검색 중...',
        
        // Article Actions
        deleteArticle: '아티클 삭제',
        deleteConfirm: '이 아티클을 삭제하시겠습니까?',
        deleteSuccess: '아티클이 성공적으로 삭제되었습니다',
        deleteError: '아티클 삭제에 실패했습니다',
        deletePermissionDenied: '등록자만 이 아티클을 삭제할 수 있습니다',
        
        // My Articles Page
        myArticlesPage: '내 아티클 관리',
        myArticlesDesc: '등록한 아티클을 조회하고 관리할 수 있습니다',
        searchConditions: '검색 조건',
        startDate: '시작 날짜',
        endDate: '종료 날짜',
        searchArticles: '아티클 검색',
        resetFilters: '필터 초기화',
        totalArticles: '총 {count}개의 아티클',
        articlesPerPage: '개씩 보기',
        loginRequired: '로그인이 필요합니다',
        loginRequiredDesc: '내 아티클을 관리하려면 먼저 로그인해주세요.',
        loginNow: '로그인 하기',
        loadMore: '더 보기',
        
        // Article List
        articleTitle: '제목',
        articleContent: '내용',
        articleCreatedAt: '작성일',
        articleActions: '작업',
        editArticle: '수정',
        deleteArticle: '삭제',
        viewOriginalUrl: '원본 보기',
        noArticlesFound: '아티클이 없습니다',
        noArticlesFoundDesc: '아직 등록한 아티클이 없습니다.',
        
        // Loading States
        loadingArticles: '아티클을 불러오는 중...',
        searching: '검색 중...',
        
        // Quick Date Selection
        quickSelect: '빠른 선택:',
        lastWeek: '최근 1주일',
        lastMonth: '최근 1개월',
        toToday: '오늘까지',
        
        // Language Settings
        language: '언어',
        languages: {
            en: 'English',
            ko: '한국어',
            zh: '中文',
            ja: '日本語',
            es: 'Español'
        }
    },
    zh: {
        appTitle: 'Open Librarian',
        searchPlaceholder: '请输入您想了解的内容...',
        searchWelcomeTitle: '搜索知识',
        searchWelcomeDesc: '基于存储的文章提供准确的答案。',
        searchRecent: '最近搜索',
        clearAll: '清除全部',
        noSearchHistory: '没有搜索记录',
        searchButton: '搜索',
        addArticle: '添加文章',
        integratedSearch: '综合搜索',
        myArticles: '我的文章',
        
        // Authentication
        login: '登录',
        logout: '登出',
        signup: '注册',
        email: '电子邮件',
        username: '用户名',
        password: '密码',
        confirmPassword: '确认密码',
        loginButton: '登录',
        signupButton: '注册',
        emailPlaceholder: 'your@email.com',
        usernamePlaceholder: '输入您的用户名',
        passwordPlaceholder: '输入您的密码',
        confirmPasswordPlaceholder: '确认您的密码',
        noAccount: '还没有账户？',
        hasAccount: '已经有账户了？',
        signupLink: '注册',
        loginLink: '登录',
        loginSuccess: '登录成功！',
        signupSuccess: '账户创建成功！',
        logoutSuccess: '成功登出！',
        loginFailed: '登录失败。请检查您的凭据。',
        signupFailed: '注册失败。请再试一次。',
        emailRequired: '电子邮件是必需的',
        usernameRequired: '用户名是必需的',
        passwordRequired: '密码是必需的',
        passwordMismatch: '密码不匹配',
        invalidEmail: '请输入有效的电子邮件地址',
        
        // Common
        user: '用户',
        unknown: '未知',
        home: '首页',
        
        // Article Management
        articleAdd: '添加文章',
        singleArticle: '单篇文章',
        bulkUpload: '批量上传 (JSONL)',
        title: '标题',
        content: '内容',
        originalUrl: '原始链接 (可选)',
        author: '作者 (可选)',
        createdDate: '创建日期 (可选)',
        createdDateHelp: '如未提供，将使用当前时间',
        titlePlaceholder: '请输入文章标题',
        contentPlaceholder: '请粘贴文章的完整内容',
        urlPlaceholder: 'https://example.com/article',
        authorPlaceholder: '张三',
        addArticleButton: '添加文章',
        
        // JSONL Upload
        selectJsonlFile: '选择 JSONL 文件',
        jsonlFormat: '每行必须是以下格式的 JSON 对象：',
        filePreview: '文件预览',
        articlesFound: '篇文章',
        uploadProgress: '上传进度',
        uploadJsonlButton: '上传 JSONL 文件',
        
        // Messages
        answerFor: '的答案',
        references: '参考资料',
        generating: '正在生成答案...',
        errorOccurred: '发生错误',
        errorMessage: '抱歉，生成答案时发生错误。请稍后重试。',
        processing: '处理中...',
        uploading: '上传中...',
        uploadComplete: '上传完成！成功：{success}篇，失败：{failed}篇',
        articleAddedSuccess: '文章添加成功！',
        articleAddError: '添加文章时发生错误。请重试。',
        noUploadData: '没有要上传的数据。',
        invalidFileType: '只能上传 JSONL 或 JSON 文件。',
        noValidArticles: '未找到有效文章。请确保每行包含具有 title 和 content 的 JSON 对象。',
        fileReadError: '读取文件时发生错误。',
        moveToSearch: '篇文章已成功添加。是否要转到搜索页面？',
        
        // Date formatting
        createdAt: '创建时间',
        today: '今天',
        yesterday: '昨天',
        daysAgo: '天前',
        weeksAgo: '周前',
        monthsAgo: '个月前',
        yearsAgo: '年前',
        
        // Search History
        justNow: '刚刚',
        minutesAgo: '分钟前',
        hoursAgo: '小时前',
        removeResult: '移除结果',
        searchingInProgress: '搜索中...',
        
        // Article Actions
        deleteArticle: '删除文章',
        deleteConfirm: '确定要删除这篇文章吗？',
        deleteSuccess: '文章删除成功',
        deleteError: '删除文章失败',
        deletePermissionDenied: '只有注册者可以删除这篇文章',
        
        // My Articles Page
        myArticlesPage: '我的文章管理',
        myArticlesDesc: '您可以查看和管理您注册的文章',
        searchConditions: '搜索条件',
        startDate: '开始日期',
        endDate: '结束日期',
        searchArticles: '搜索文章',
        resetFilters: '重置筛选',
        totalArticles: '共 {count} 篇文章',
        articlesPerPage: '篇每页',
        loginRequired: '需要登录',
        loginRequiredDesc: '请先登录以管理您的文章。',
        loginNow: '立即登录',
        loadMore: '加载更多',
        
        // Article List
        articleTitle: '标题',
        articleContent: '内容',
        articleCreatedAt: '创建日期',
        articleActions: '操作',
        editArticle: '编辑',
        deleteArticle: '删除',
        viewOriginalUrl: '查看原文',
        noArticlesFound: '未找到文章',
        noArticlesFoundDesc: '您还没有注册任何文章。',
        
        // Loading States
        loadingArticles: '正在加载文章...',
        searching: '搜索中...',
        
        // Quick Date Selection
        quickSelect: '快速选择：',
        lastWeek: '最近一周',
        lastMonth: '最近一个月',
        toToday: '至今天',
        
        // Language Settings
        language: '语言',
        languages: {
            en: 'English',
            ko: '한국어',
            zh: '中文',
            ja: '日本語',
            es: 'Español'
        }
    },
    ja: {
        appTitle: 'Open Librarian',
        searchPlaceholder: '知りたいことを自由に質問してください...',
        searchWelcomeTitle: '知識を検索する',
        searchWelcomeDesc: '保存された記事に基づいて正確な回答を提供します。',
        searchRecent: '最近の検索',
        clearAll: 'すべて削除',
        noSearchHistory: '検索履歴がありません',
        searchButton: '検索',
        addArticle: '記事追加',
        integratedSearch: '統合検索',
        myArticles: '私の記事',
        
        // Authentication
        login: 'ログイン',
        logout: 'ログアウト',
        signup: 'サインアップ',
        email: 'メール',
        username: 'ユーザー名',
        password: 'パスワード',
        confirmPassword: 'パスワード確認',
        loginButton: 'ログイン',
        signupButton: 'サインアップ',
        emailPlaceholder: 'your@email.com',
        usernamePlaceholder: 'ユーザー名を入力してください',
        passwordPlaceholder: 'パスワードを入力してください',
        confirmPasswordPlaceholder: 'パスワードを確認してください',
        noAccount: 'アカウントをお持ちでないですか？',
        hasAccount: 'すでにアカウントをお持ちですか？',
        signupLink: 'サインアップ',
        loginLink: 'ログイン',
        loginSuccess: 'ログインに成功しました！',
        signupSuccess: 'アカウントが正常に作成されました！',
        logoutSuccess: '正常にログアウトしました！',
        loginFailed: 'ログインに失敗しました。資格情報を確認してください。',
        signupFailed: 'サインアップに失敗しました。再試行してください。',
        emailRequired: 'メールは必須です',
        usernameRequired: 'ユーザー名は必須です',
        passwordRequired: 'パスワードは必須です',
        passwordMismatch: 'パスワードが一致しません',
        invalidEmail: '有効なメールアドレスを入力してください',
        
        // Common
        user: 'ユーザー',
        unknown: '不明',
        home: 'ホーム',
        
        // Article Management
        articleAdd: '記事追加',
        singleArticle: '単一記事',
        bulkUpload: '一括アップロード (JSONL)',
        title: 'タイトル',
        content: '内容',
        originalUrl: '元のURL (オプション)',
        author: '著者 (オプション)',
        createdDate: '作成日 (オプション)',
        createdDateHelp: '未入力の場合は現在時刻が設定されます',
        titlePlaceholder: '記事のタイトルを入力してください',
        contentPlaceholder: '記事の全内容を貼り付けてください',
        urlPlaceholder: 'https://example.com/article',
        authorPlaceholder: '田中太郎',
        addArticleButton: '記事を追加',
        
        // JSONL Upload
        selectJsonlFile: 'JSONL ファイルを選択',
        jsonlFormat: '各行は以下の形式のJSONオブジェクトである必要があります：',
        filePreview: 'ファイルプレビュー',
        articlesFound: '件の記事が見つかりました',
        uploadProgress: 'アップロード進行状況',
        uploadJsonlButton: 'JSONL ファイルをアップロード',
        
        // Messages
        answerFor: 'に対する回答',
        references: '参考資料',
        generating: '回答を生成しています...',
        errorOccurred: 'エラーが発生しました',
        errorMessage: '申し訳ございません。回答の生成中にエラーが発生しました。しばらくしてから再試行してください。',
        processing: '処理中...',
        uploading: 'アップロード中...',
        uploadComplete: 'アップロード完了！成功：{success}件、失敗：{failed}件',
        articleAddedSuccess: '記事が正常に追加されました！',
        articleAddError: '記事の追加中にエラーが発生しました。再試行してください。',
        noUploadData: 'アップロードするデータがありません。',
        invalidFileType: 'JSONL または JSON ファイルのみアップロード可能です。',
        noValidArticles: '有効な記事が見つかりませんでした。各行にtitleとcontentを含むJSONオブジェクトが含まれていることを確認してください。',
        fileReadError: 'ファイルの読み取り中にエラーが発生しました。',
        moveToSearch: '件の記事が正常に追加されました。検索画面に移動しますか？',
        
        // Date formatting
        createdAt: '作成日',
        today: '今日',
        yesterday: '昨日',
        daysAgo: '日前',
        weeksAgo: '週間前',
        monthsAgo: 'ヶ月前',
        yearsAgo: '年前',
        
        // Search History
        justNow: 'たった今',
        minutesAgo: '分前',
        hoursAgo: '時間前',
        removeResult: '結果を削除',
        searchingInProgress: '検索中...',
        
        // Article Actions
        deleteArticle: '記事削除',
        deleteConfirm: 'この記事を削除しますか？',
        deleteSuccess: '記事が正常に削除されました',
        deleteError: '記事の削除に失敗しました',
        deletePermissionDenied: '登録者のみがこの記事を削除できます',
        
        // My Articles Page
        myArticlesPage: '私の記事管理',
        myArticlesDesc: '登録した記事を閲覧・管理できます',
        searchConditions: '検索条件',
        startDate: '開始日',
        endDate: '終了日',
        searchArticles: '記事検索',
        resetFilters: 'フィルターリセット',
        totalArticles: '合計 {count} 件の記事',
        articlesPerPage: '件ずつ表示',
        loginRequired: 'ログインが必要です',
        loginRequiredDesc: '記事を管理するにはまずログインしてください。',
        loginNow: 'ログインする',
        loadMore: 'さらに表示',
        
        // Article List
        articleTitle: 'タイトル',
        articleContent: '内容',
        articleCreatedAt: '作成日',
        articleActions: '操作',
        editArticle: '編集',
        deleteArticle: '削除',
        viewOriginalUrl: '元記事を見る',
        noArticlesFound: '記事が見つかりません',
        noArticlesFoundDesc: 'まだ記事を登録していません。',
        
        // Loading States
        loadingArticles: '記事を読み込み中...',
        searching: '検索中...',
        
        // Quick Date Selection
        quickSelect: 'クイック選択：',
        lastWeek: '過去1週間',
        lastMonth: '過去1ヶ月',
        toToday: '今日まで',
        
        // Language Settings
        language: '言語',
        languages: {
            en: 'English',
            ko: '한국어',
            zh: '中文',
            ja: '日本語',
            es: 'Español'
        }
    },
    es: {
        appTitle: 'Open Librarian',
        searchPlaceholder: 'Pregunta libremente sobre lo que te interese...',
        searchWelcomeTitle: 'Buscar conocimiento',
        searchWelcomeDesc: 'Proporcionamos respuestas precisas basadas en artículos almacenados.',
        searchRecent: 'Búsquedas recientes',
        clearAll: 'Limpiar todo',
        noSearchHistory: 'No hay historial de búsqueda',
        searchButton: 'Buscar',
        addArticle: 'Agregar artículo',
        integratedSearch: 'Búsqueda integrada',
        myArticles: 'Mis artículos',
        
        // Authentication
        login: 'Iniciar sesión',
        logout: 'Cerrar sesión',
        signup: 'Registrarse',
        email: 'Correo electrónico',
        username: 'Nombre de usuario',
        password: 'Contraseña',
        confirmPassword: 'Confirmar contraseña',
        loginButton: 'Iniciar sesión',
        signupButton: 'Registrarse',
        emailPlaceholder: 'your@email.com',
        usernamePlaceholder: 'Ingrese su nombre de usuario',
        passwordPlaceholder: 'Ingrese su contraseña',
        confirmPasswordPlaceholder: 'Confirme su contraseña',
        noAccount: '¿No tienes una cuenta?',
        hasAccount: '¿Ya tienes una cuenta?',
        signupLink: 'Regístrate',
        loginLink: 'Inicia sesión',
        loginSuccess: '¡Inicio de sesión exitoso!',
        signupSuccess: '¡Cuenta creada exitosamente!',
        logoutSuccess: '¡Sesión cerrada exitosamente!',
        loginFailed: 'Error de inicio de sesión. Por favor, verifica tus credenciales.',
        signupFailed: 'Error de registro. Por favor, inténtalo de nuevo.',
        emailRequired: 'El correo electrónico es obligatorio',
        usernameRequired: 'El nombre de usuario es obligatorio',
        passwordRequired: 'La contraseña es obligatoria',
        passwordMismatch: 'Las contraseñas no coinciden',
        invalidEmail: 'Por favor, ingresa una dirección de correo electrónico válida',
        
        // Common
        user: 'Usuario',
        unknown: 'Desconocido',
        home: 'Inicio',
        
        // Article Management
        articleAdd: 'Agregar artículo',
        singleArticle: 'Artículo único',
        bulkUpload: 'Carga masiva (JSONL)',
        title: 'Título',
        content: 'Contenido',
        originalUrl: 'URL original (Opcional)',
        author: 'Autor (Opcional)',
        createdDate: 'Fecha de creación (Opcional)',
        createdDateHelp: 'Si no se proporciona, se usará la hora actual',
        titlePlaceholder: 'Ingrese el título del artículo',
        contentPlaceholder: 'Pegue el contenido completo del artículo',
        urlPlaceholder: 'https://example.com/article',
        authorPlaceholder: 'Juan Pérez',
        addArticleButton: 'Agregar artículo',
        
        // JSONL Upload
        selectJsonlFile: 'Seleccionar archivo JSONL',
        jsonlFormat: 'Cada línea debe ser un objeto JSON en el siguiente formato:',
        filePreview: 'Vista previa del archivo',
        articlesFound: 'artículos encontrados',
        uploadProgress: 'Progreso de carga',
        uploadJsonlButton: 'Subir archivo JSONL',
        
        // Messages
        answerFor: 'Respuesta para',
        references: 'Referencias',
        generating: 'Generando respuesta...',
        errorOccurred: 'Ocurrió un error',
        errorMessage: 'Lo sentimos, ocurrió un error al generar la respuesta. Por favor, inténtelo de nuevo más tarde.',
        processing: 'Procesando...',
        uploading: 'Subiendo...',
        uploadComplete: '¡Carga completa! Éxito: {success}, Falló: {failed}',
        articleAddedSuccess: '¡Artículo agregado exitosamente!',
        articleAddError: 'Ocurrió un error al agregar el artículo. Por favor, inténtelo de nuevo.',
        noUploadData: 'No hay datos para cargar.',
        invalidFileType: 'Solo se pueden cargar archivos JSONL o JSON.',
        noValidArticles: 'No se encontraron artículos válidos. Asegúrese de que cada línea contenga un objeto JSON con título y contenido.',
        fileReadError: 'Ocurrió un error al leer el archivo.',
        moveToSearch: 'artículos se han agregado con éxito. ¿Le gustaría ir a la pantalla de búsqueda?',
        
        // Date formatting
        createdAt: 'Creado',
        today: 'Hoy',
        yesterday: 'Ayer',
        daysAgo: 'días atrás',
        weeksAgo: 'semanas atrás',
        monthsAgo: 'meses atrás',
        yearsAgo: 'años atrás',
        
        // Search History
        justNow: 'ahora mismo',
        minutesAgo: 'min atrás',
        hoursAgo: 'h atrás',
        removeResult: 'Eliminar resultado',
        searchingInProgress: 'Buscando...',
        
        // Article Actions
        deleteArticle: 'Eliminar artículo',
        deleteConfirm: '¿Está seguro de que desea eliminar este artículo?',
        deleteSuccess: 'Artículo eliminado exitosamente',
        deleteError: 'Error al eliminar el artículo',
        deletePermissionDenied: 'Solo el registrador puede eliminar este artículo',
        
        // My Articles Page
        myArticlesPage: 'Gestión de Mis Artículos',
        myArticlesDesc: 'Puede ver y gestionar sus artículos registrados',
        searchConditions: 'Condiciones de Búsqueda',
        startDate: 'Fecha de Inicio',
        endDate: 'Fecha de Fin',
        searchArticles: 'Buscar Artículos',
        resetFilters: 'Restablecer Filtros',
        totalArticles: 'Total {count} artículos',
        articlesPerPage: 'artículos por página',
        loginRequired: 'Se Requiere Inicio de Sesión',
        loginRequiredDesc: 'Por favor, inicie sesión primero para gestionar sus artículos.',
        loginNow: 'Iniciar Sesión Ahora',
        loadMore: 'Cargar Más',
        
        // Article List
        articleTitle: 'Título',
        articleContent: 'Contenido',
        articleCreatedAt: 'Fecha de Creación',
        articleActions: 'Acciones',
        editArticle: 'Editar',
        deleteArticle: 'Eliminar',
        viewOriginalUrl: 'Ver Original',
        noArticlesFound: 'No se encontraron artículos',
        noArticlesFoundDesc: 'Aún no ha registrado ningún artículo.',
        
        // Loading States
        loadingArticles: 'Cargando artículos...',
        searching: 'Buscando...',
        
        // Quick Date Selection
        quickSelect: 'Selección Rápida:',
        lastWeek: 'Última Semana',
        lastMonth: 'Último Mes',
        toToday: 'Hasta Hoy',
        
        // Language Settings
        language: 'Idioma',
        languages: {
            en: 'English',
            ko: '한국어',
            zh: '中文',
            ja: '日本語',
            es: 'Español'
        }
    }
};

let currentLanguage = 'ko'; // 기본값을 한국어로 설정

// 번역 함수
function t(key, params = {}) {
    const translation = TRANSLATIONS[currentLanguage] || TRANSLATIONS['en'];
    const keys = key.split('.');
    let value = translation;
    
    for (const k of keys) {
        value = value[k];
        if (value === undefined) {
            // Fallback to English if translation not found
            value = TRANSLATIONS['en'];
            for (const k of keys) {
                value = value[k];
                if (value === undefined) {
                    return key; // Return key if not found in English either
                }
            }
            break;
        }
    }
    
    if (typeof value === 'string' && Object.keys(params).length > 0) {
        return value.replace(/\{(\w+)\}/g, (match, paramKey) => {
            return params[paramKey] !== undefined ? params[paramKey] : match;
        });
    }
    
    return value;
}

// 날짜 포맷팅 함수
function formatCreatedDate(dateString) {
    if (!dateString) return '';
    
    try {
        const createdDate = new Date(dateString);
        const now = new Date();
        const diffMs = now - createdDate;
        const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
        const diffWeeks = Math.floor(diffDays / 7);
        const diffMonths = Math.floor(diffDays / 30);
        const diffYears = Math.floor(diffDays / 365);
        
        if (diffDays === 0) {
            return t('today');
        } else if (diffDays === 1) {
            return t('yesterday');
        } else if (diffDays < 7) {
            return `${diffDays}${t('daysAgo')}`;
        } else if (diffWeeks < 4) {
            return `${diffWeeks}${t('weeksAgo')}`;
        } else if (diffMonths < 12) {
            return `${diffMonths}${t('monthsAgo')}`;
        } else {
            return `${diffYears}${t('yearsAgo')}`;
        }
    } catch (error) {
        console.warn('Invalid date format:', dateString);
        return '';
    }
}

// 언어 설정 저장/로드 (IndexedDB)
async function saveLanguagePreference(language) {
    if (typeof db === 'undefined' || !db) {
        localStorage.setItem('librarian_language', language);
        return;
    }
    
    try {
        const transaction = db.transaction(['settings'], 'readwrite');
        const store = transaction.objectStore('settings');
        await new Promise((resolve, reject) => {
            const request = store.put({
                key: 'language',
                value: language,
                timestamp: new Date().toISOString()
            });
            request.onsuccess = () => resolve();
            request.onerror = () => reject(request.error);
        });
    } catch (error) {
        console.error('Failed to save language preference:', error);
        localStorage.setItem('librarian_language', language);
    }
}

async function loadLanguagePreference() {
    if (typeof db === 'undefined' || !db) {
        return localStorage.getItem('librarian_language') || 'ko';
    }
    
    try {
        return new Promise((resolve, reject) => {
            const transaction = db.transaction(['settings'], 'readonly');
            const store = transaction.objectStore('settings');
            const request = store.get('language');
            
            request.onsuccess = () => {
                const result = request.result;
                resolve(result ? result.value : 'ko');
            };
            request.onerror = () => {
                resolve(localStorage.getItem('librarian_language') || 'ko');
            };
        });
    } catch (error) {
        console.error('Failed to load language preference:', error);
        return localStorage.getItem('librarian_language') || 'ko';
    }
}

// 언어 변경
async function changeLanguage(language) {
    currentLanguage = language;
    await saveLanguagePreference(language);
    updatePageLanguage();
}

// 페이지의 모든 텍스트 업데이트
function updatePageLanguage() {
    // Update HTML lang attribute
    const htmlRoot = document.getElementById('html-root') || document.documentElement;
    if (htmlRoot) {
        htmlRoot.lang = currentLanguage;
    }
    
    // Update page title based on the page type
    const titleElement = document.querySelector('title[data-i18n]');
    if (titleElement) {
        const titleKey = titleElement.getAttribute('data-i18n');
        document.title = t(titleKey) + ' - ' + t('appTitle');
    } else {
        document.title = t('appTitle');
    }
    
    // Update all elements with data-i18n attribute
    document.querySelectorAll('[data-i18n]').forEach(element => {
        const key = element.getAttribute('data-i18n');
        element.textContent = t(key);
    });
    
    // Update all elements with data-i18n-placeholder attribute
    document.querySelectorAll('[data-i18n-placeholder]').forEach(element => {
        const key = element.getAttribute('data-i18n-placeholder');
        element.placeholder = t(key);
    });
    
    // Update language selector
    updateLanguageSelector();
    
    // Update search history display if the function exists
    if (typeof updateHistoryDisplay === 'function') {
        updateHistoryDisplay();
    }
}

// 언어 선택기 업데이트
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

// 언어 초기화
async function initLanguage() {
    const savedLanguage = await loadLanguagePreference();
    currentLanguage = savedLanguage;
    updatePageLanguage();
}
