package webui

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Local Agent</title>
    <link rel="icon" type="image/png" href="/static/favicon.png" sizes="150x150">
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

        .spinner.thinking {
            border-top-color: #7D56F4;
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

        .message-actions {
            display: flex;
            justify-content: flex-end;
            margin-top: 0.5rem;
        }

        .copy-btn {
            background: transparent;
            border: 1px solid var(--border-color);
            border-radius: 5px;
            color: var(--text-secondary);
            cursor: pointer;
            font-size: 0.75rem;
            padding: 0.2rem 0.6rem;
            transition: all 0.2s;
        }

        .copy-btn:hover {
            background: var(--bg-tertiary);
            color: var(--text-primary);
        }

        .copy-btn.copied {
            color: #4caf50;
            border-color: #4caf50;
        }

        .reasoning-block {
            margin: 0.5rem 0;
            border: 1px solid rgba(125, 86, 244, 0.3);
            border-radius: 6px;
            overflow: hidden;
        }

        .reasoning-block summary {
            cursor: pointer;
            padding: 0.4rem 0.75rem;
            background: rgba(125, 86, 244, 0.1);
            color: #9d79f5;
            font-weight: 600;
            font-size: 0.85rem;
            user-select: none;
            list-style: none;
            display: flex;
            align-items: center;
            gap: 0.4rem;
        }

        .reasoning-block summary::-webkit-details-marker { display: none; }

        .reasoning-block[open] summary { border-bottom: 1px solid rgba(125, 86, 244, 0.2); }

        .reasoning-content {
            padding: 0.75rem;
            margin: 0;
            font-size: 0.78rem;
            color: var(--text-secondary);
            white-space: pre-wrap;
            word-break: break-word;
            font-family: 'Menlo', 'Courier New', monospace;
            background: rgba(0, 0, 0, 0.2);
            max-height: 300px;
            overflow-y: auto;
        }

        .reasoning-preview {
            font-size: 0.78rem;
            color: #9d79f5;
            font-family: 'Menlo', 'Courier New', monospace;
            max-height: 120px;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-all;
            margin-top: 0.4rem;
            padding: 0.4rem 0.5rem;
            background: rgba(125, 86, 244, 0.06);
            border-radius: 4px;
            border-left: 2px solid rgba(125, 86, 244, 0.4);
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
            <h1><img src="/static/favicon.png" alt="Local Agent" style="width: 4rem; height: 4rem; vertical-align: middle; margin-right: 0.75rem;"> local-agent [interactive mode]</h1>
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
            <div class="status-item" id="thinkingIndicator" style="display: none;">
                <span style="color: #7D56F4; font-weight: 600; font-style: italic;">🧠 Thinking Mode</span>
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
        let isThinkingModel = false;

        // Load initial status
        async function loadStatus() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                document.getElementById('directory').textContent = data.directory;
                document.getElementById('model').textContent = data.model;
                document.getElementById('totalFiles').textContent = data.totalFiles;
                isThinkingModel = data.isThinking || false;
                const thinkingIndicator = document.getElementById('thinkingIndicator');
                if (thinkingIndicator) {
                    thinkingIndicator.style.display = isThinkingModel ? 'flex' : 'none';
                }
                
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

        // Render message content, turning [reasoning]...[/reasoning] blocks into
        // collapsible <details> elements and leaving the rest as plain text.
        function renderContent(content, container) {
            const re = /\[reasoning\]([\s\S]*?)\[\/reasoning\]\n?/g;
            let lastIndex = 0;
            let match;
            let found = false;
            while ((match = re.exec(content)) !== null) {
                found = true;
                const before = content.slice(lastIndex, match.index).trim();
                if (before) {
                    const div = document.createElement('div');
                    div.style.whiteSpace = 'pre-wrap';
                    div.textContent = before;
                    container.appendChild(div);
                }
                const details = document.createElement('details');
                details.className = 'reasoning-block';
                const summary = document.createElement('summary');
                summary.textContent = '\uD83E\uDDE0 Reasoning';
                const pre = document.createElement('pre');
                pre.className = 'reasoning-content';
                pre.textContent = match[1].trim();
                details.appendChild(summary);
                details.appendChild(pre);
                container.appendChild(details);
                lastIndex = match.index + match[0].length;
            }
            const remaining = content.slice(lastIndex);
            if (remaining.trim() || !found) {
                const div = document.createElement('div');
                div.style.whiteSpace = 'pre-wrap';
                div.textContent = remaining || content;
                container.appendChild(div);
            }
        }

        // Add message to chat
        function addMessage(role, content, timestamp) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + role;

            const contentDiv = document.createElement('div');
            renderContent(content, contentDiv);

            const timeDiv = document.createElement('div');
            timeDiv.className = 'message-timestamp';
            timeDiv.textContent = new Date(timestamp).toLocaleTimeString();

            messageDiv.appendChild(contentDiv);
            messageDiv.appendChild(timeDiv);

            if (role === 'assistant') {
                const actionsDiv = document.createElement('div');
                actionsDiv.className = 'message-actions';

                const copyBtn = document.createElement('button');
                copyBtn.className = 'copy-btn';
                copyBtn.textContent = 'Copy';
                copyBtn.addEventListener('click', () => {
                    const answerOnly = content.replace(/\[reasoning\][\s\S]*?\[\/reasoning\]\n?/g, '').trim();
                    navigator.clipboard.writeText(answerOnly).then(() => {
                        copyBtn.textContent = 'Copied!';
                        copyBtn.classList.add('copied');
                        setTimeout(() => {
                            copyBtn.textContent = 'Copy';
                            copyBtn.classList.remove('copied');
                        }, 2000);
                    });
                });

                actionsDiv.appendChild(copyBtn);
                messageDiv.appendChild(actionsDiv);
            }

            chatContainer.appendChild(messageDiv);
        }

        // Show loading indicator
        function showLoading() {
            const loadingDiv = document.createElement('div');
            loadingDiv.className = 'loading';
            loadingDiv.id = 'loading';
            if (isThinkingModel) {
                loadingDiv.innerHTML = '<div class="spinner thinking"></div>' +
                    '<div style="display:flex;flex-direction:column;gap:0.3rem;flex:1;min-width:0">' +
                    '<span id="loadingText">Analyzing...</span>' +
                    '<div id="reasoningPreview" class="reasoning-preview" style="display:none"></div>' +
                    '</div>';
            } else {
                loadingDiv.innerHTML = '<div class="spinner"></div><span id="loadingText">Analyzing...</span>';
            }
            chatContainer.appendChild(loadingDiv);
            scrollToBottom();
        }

        // Append a thinking line to the live reasoning preview
        function appendThinkLine(line) {
            const preview = document.getElementById('reasoningPreview');
            if (preview) {
                preview.style.display = 'block';
                preview.textContent += (preview.textContent ? '\n' : '') + line;
                preview.scrollTop = preview.scrollHeight;
                scrollToBottom();
            }
        }

        // Update loading progress text
        function updateLoadingText(text) {
            const el = document.getElementById('loadingText');
            if (el) el.textContent = text;
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

            // Open SSE progress stream
            const evtSource = new EventSource('/api/progress');
            evtSource.onmessage = function(e) {
                if (e.data === 'done') {
                    evtSource.close();
                } else if (e.data.startsWith('THINK:')) {
                    appendThinkLine(e.data.substring(6));
                } else {
                    updateLoadingText(e.data);
                    scrollToBottom();
                }
            };
            evtSource.onerror = function() { evtSource.close(); };

            try {
                const response = await fetch('/api/chat', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ message }),
                });

                const data = await response.json();
                evtSource.close();
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
