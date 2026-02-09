package webui

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>local-agent</title>
    <link rel="icon" type="image/png" href="/static/favicon.png" sizes="32x32">
    <style>
        :root {
            --bg-primary: #1a1a1a;
            --bg-secondary: #2d2d2d;
            --bg-tertiary: #252525;
            --bg-input: #1a1a1a;
            --border-color: #3d3d3d;
            --text-primary: #e0e0e0;
            --text-secondary: #888;
            --text-label: #aaa;
            --accent-color: #4a9eff;
            --accent-hover: #3a8eef;
            --message-user-bg: #4a9eff;
            --message-assistant-bg: #2d2d2d;
            --scrollbar-track: #1a1a1a;
            --scrollbar-thumb: #3d3d3d;
            --scrollbar-thumb-hover: #4d4d4d;
            --shadow-color: rgba(0,0,0,0.3);
        }

        body.light-theme {
            --bg-primary: #f5f5f5;
            --bg-secondary: #ffffff;
            --bg-tertiary: #e8e8e8;
            --bg-input: #ffffff;
            --border-color: #d0d0d0;
            --text-primary: #1a1a1a;
            --text-secondary: #666;
            --text-label: #555;
            --accent-color: #2563eb;
            --accent-hover: #1d4ed8;
            --message-user-bg: #2563eb;
            --message-assistant-bg: #f0f0f0;
            --scrollbar-track: #e8e8e8;
            --scrollbar-thumb: #c0c0c0;
            --scrollbar-thumb-hover: #a0a0a0;
            --shadow-color: rgba(0,0,0,0.1);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            height: 100vh;
            display: flex;
            flex-direction: column;
            transition: background 0.3s, color 0.3s;
        }

        .header {
            background: var(--bg-secondary);
            padding: 1rem 2rem;
            border-bottom: 2px solid var(--border-color);
            box-shadow: 0 2px 10px var(--shadow-color);
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
        }

        .header-content {
            flex: 1;
        }

        .header h1 {
            font-size: 1.5rem;
            color: var(--accent-color);
            margin-bottom: 0.5rem;
        }

        .theme-toggle {
            background: var(--bg-tertiary);
            border: 2px solid var(--border-color);
            border-radius: 8px;
            padding: 0.5rem 1rem;
            cursor: pointer;
            font-size: 1.2rem;
            transition: all 0.2s;
            color: var(--text-primary);
            margin-left: 1rem;
        }

        .theme-toggle:hover {
            background: var(--border-color);
            transform: scale(1.05);
        }

        .status-bar {
            display: flex;
            gap: 2rem;
            font-size: 0.9rem;
            color: var(--text-secondary);
        }

        .status-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .status-label {
            font-weight: 600;
            color: var(--text-label);
        }

        .chat-container {
            flex: 1;
            overflow-y: auto;
            padding: 2rem;
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .message {
            max-width: 80%;
            padding: 1rem 1.5rem;
            border-radius: 12px;
            line-height: 1.6;
            white-space: pre-wrap;
            word-wrap: break-word;
        }

        .message.user {
            align-self: flex-end;
            background: var(--message-user-bg);
            color: white;
            margin-left: auto;
        }

        .message.assistant {
            align-self: flex-start;
            background: var(--message-assistant-bg);
            border: 1px solid var(--border-color);
        }

        .message-timestamp {
            font-size: 0.75rem;
            opacity: 0.6;
            margin-top: 0.5rem;
        }

        .input-container {
            background: var(--bg-secondary);
            padding: 1.5rem 2rem;
            border-top: 2px solid var(--border-color);
            box-shadow: 0 -2px 10px var(--shadow-color);
        }

        .input-wrapper {
            display: flex;
            gap: 1rem;
            max-width: 1200px;
            margin: 0 auto;
        }

        #messageInput {
            flex: 1;
            background: var(--bg-input);
            border: 2px solid var(--border-color);
            border-radius: 8px;
            padding: 0.75rem 1rem;
            color: var(--text-primary);
            font-size: 1rem;
            font-family: inherit;
            transition: border-color 0.2s;
        }

        #messageInput:focus {
            outline: none;
            border-color: var(--accent-color);
        }

        #sendButton {
            background: var(--accent-color);
            color: white;
            border: none;
            border-radius: 8px;
            padding: 0.75rem 2rem;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.2s;
        }

        #sendButton:hover:not(:disabled) {
            background: var(--accent-hover);
        }

        #sendButton:disabled {
            background: #555;
            cursor: not-allowed;
            opacity: 0.5;
        }

        .loading {
            display: flex;
            gap: 0.5rem;
            align-items: center;
            color: var(--text-secondary);
            padding: 1rem;
        }

        .spinner {
            width: 20px;
            height: 20px;
            border: 3px solid var(--border-color);
            border-top-color: var(--accent-color);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        .commands-hint {
            padding: 0.5rem 1rem;
            background: var(--bg-tertiary);
            border-radius: 6px;
            font-size: 0.85rem;
            color: var(--text-secondary);
            margin-top: 0.5rem;
        }

        .commands-hint code {
            background: var(--bg-input);
            padding: 0.2rem 0.4rem;
            border-radius: 3px;
            color: var(--accent-color);
        }

        /* Scrollbar styling */
        ::-webkit-scrollbar {
            width: 10px;
        }

        ::-webkit-scrollbar-track {
            background: var(--scrollbar-track);
        }

        ::-webkit-scrollbar-thumb {
            background: var(--scrollbar-thumb);
            border-radius: 5px;
        }

        ::-webkit-scrollbar-thumb:hover {
            background: var(--scrollbar-thumb-hover);
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <h1><img src="/static/favicon.png" alt="Local Agent" style="width: 2.5rem; height: 2.5rem; vertical-align: middle; margin-right: 0.75rem;"> local-agent [interactive mode]</h1>
            <div class="status-bar">
            <div class="status-item">
                <span class="status-label">Directory:</span>
                <span id="directory">-</span>
            </div>
            <div class="status-item">
                <span class="status-label">Model:</span>
                <span id="model">-</span>
            </div>
            <div class="status-item">
                <span class="status-label">Files:</span>
                <span id="totalFiles">-</span>
            </div>
            <div class="status-item" id="focusItem" style="display: none;">
                <span class="status-label">Focus:</span>
                <span id="focusedPath">-</span>
            </div>
            </div>
        </div>
        <button class="theme-toggle" id="themeToggle" title="Toggle theme">🌙</button>
    </div>

    <div class="chat-container" id="chatContainer"></div>

    <div class="input-container">
        <div class="input-wrapper">
            <input 
                type="text" 
                id="messageInput" 
                placeholder="Ask a question about your codebase..."
                autocomplete="off"
            />
            <button id="sendButton">Send</button>
        </div>
        <div class="commands-hint">
            💡 To know more run: <code>help</code> • 🌐 Web UI: <code>http://localhost:5050</code>
        </div>
    </div>

    <script>
        const chatContainer = document.getElementById('chatContainer');
        const messageInput = document.getElementById('messageInput');
        const sendButton = document.getElementById('sendButton');
        let isProcessing = false;

        // Load initial status
        async function loadStatus() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                document.getElementById('directory').textContent = data.directory;
                document.getElementById('model').textContent = data.model;
                document.getElementById('totalFiles').textContent = data.totalFiles;
                
                if (data.focusedPath) {
                    document.getElementById('focusedPath').textContent = data.focusedPath;
                    document.getElementById('focusItem').style.display = 'flex';
                } else {
                    document.getElementById('focusItem').style.display = 'none';
                }
            } catch (error) {
                console.error('Failed to load status:', error);
            }
        }

        // Load initial messages
        async function loadMessages() {
            try {
                const response = await fetch('/api/messages');
                const messages = await response.json();
                chatContainer.innerHTML = '';
                messages.forEach(msg => addMessage(msg.role, msg.content, msg.timestamp));
                scrollToBottom();
            } catch (error) {
                console.error('Failed to load messages:', error);
            }
        }

        // Add message to chat
        function addMessage(role, content, timestamp) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + role;
            
            const contentDiv = document.createElement('div');
            contentDiv.textContent = content;
            
            const timeDiv = document.createElement('div');
            timeDiv.className = 'message-timestamp';
            timeDiv.textContent = new Date(timestamp).toLocaleTimeString();
            
            messageDiv.appendChild(contentDiv);
            messageDiv.appendChild(timeDiv);
            chatContainer.appendChild(messageDiv);
        }

        // Show loading indicator
        function showLoading() {
            const loadingDiv = document.createElement('div');
            loadingDiv.className = 'loading';
            loadingDiv.id = 'loading';
            loadingDiv.innerHTML = '<div class="spinner"></div><span>Processing...</span>';
            chatContainer.appendChild(loadingDiv);
            scrollToBottom();
        }

        // Hide loading indicator
        function hideLoading() {
            const loadingDiv = document.getElementById('loading');
            if (loadingDiv) {
                loadingDiv.remove();
            }
        }

        // Scroll to bottom
        function scrollToBottom() {
            chatContainer.scrollTop = chatContainer.scrollHeight;
        }

        // Send message
        async function sendMessage() {
            const message = messageInput.value.trim();
            if (!message || isProcessing) return;

            isProcessing = true;
            sendButton.disabled = true;
            messageInput.disabled = true;

            // Add user message
            addMessage('user', message, new Date().toISOString());
            messageInput.value = '';
            scrollToBottom();

            showLoading();

            try {
                const response = await fetch('/api/chat', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ message }),
                });

                const data = await response.json();
                hideLoading();

                if (data.success && data.message) {
                    addMessage(data.message.role, data.message.content, data.message.timestamp);
                    scrollToBottom();
                    
                    // Reload status in case focus or other settings changed
                    await loadStatus();
                } else {
                    addMessage('assistant', '❌ Error: ' + (data.error || 'Unknown error'), new Date().toISOString());
                }
            } catch (error) {
                hideLoading();
                addMessage('assistant', '❌ Network error: ' + error.message, new Date().toISOString());
            } finally {
                isProcessing = false;
                sendButton.disabled = false;
                messageInput.disabled = false;
                messageInput.focus();
            }
        }

        // Theme toggle functionality
        const themeToggle = document.getElementById('themeToggle');
        const body = document.body;
        
        // Load saved theme
        const savedTheme = localStorage.getItem('theme') || 'dark';
        if (savedTheme === 'light') {
            body.classList.add('light-theme');
            themeToggle.textContent = '☀️';
        }
        
        themeToggle.addEventListener('click', () => {
            body.classList.toggle('light-theme');
            const isLight = body.classList.contains('light-theme');
            themeToggle.textContent = isLight ? '☀️' : '🌙';
            localStorage.setItem('theme', isLight ? 'light' : 'dark');
        });

        // Event listeners
        sendButton.addEventListener('click', sendMessage);
        messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });

        // Initialize
        loadStatus();
        loadMessages();
        messageInput.focus();

        // Auto-refresh status every 5 seconds
        setInterval(loadStatus, 5000);
    </script>
</body>
</html>
`
