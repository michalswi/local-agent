package webui

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Local Agent</title>
    <link rel="icon" type="image/png" href="/static/favicon.png" sizes="150x150">
    <style>
        /* ── Apple dark mode palette ──────────────────────────── */
        :root {
            --bg-primary:   #000000;
            --bg-secondary: #1c1c1e;
            --bg-tertiary:  #2c2c2e;
            --bg-input:     #1c1c1e;
            --bg-glass:     rgba(28,28,30,0.82);
            --border-color: rgba(255,255,255,0.08);
            --text-primary:   #f2f2f7;
            --text-secondary: #8e8e93;
            --text-label:     #636366;
            --accent-color:  #0a84ff;
            --accent-hover:  #0070e0;
            --bubble-user:   #0a84ff;
            --bubble-assist: #2c2c2e;
            --scrollbar-track: transparent;
            --scrollbar-thumb: #3a3a3c;
            --scrollbar-thumb-hover: #48484a;
            --shadow-color: rgba(0,0,0,0.5);
            --reasoning-color: #bf5af2;
        }

        /* ── Apple light mode palette ─────────────────────────── */
        body.light-theme {
            --bg-primary:   #f2f2f7;
            --bg-secondary: #ffffff;
            --bg-tertiary:  #e5e5ea;
            --bg-input:     #ffffff;
            --bg-glass:     rgba(255,255,255,0.82);
            --border-color: rgba(0,0,0,0.08);
            --text-primary:   #1c1c1e;
            --text-secondary: #6c6c70;
            --text-label:     #8e8e93;
            --accent-color:  #007aff;
            --accent-hover:  #005ecb;
            --bubble-user:   #007aff;
            --bubble-assist: #e5e5ea;
            --scrollbar-track: transparent;
            --scrollbar-thumb: #c7c7cc;
            --scrollbar-thumb-hover: #aeaeb2;
            --shadow-color: rgba(0,0,0,0.12);
            --reasoning-color: #8944ab;
        }

        *, *::before, *::after { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Helvetica Neue', Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            height: 100vh;
            display: flex;
            flex-direction: column;
            transition: background 0.25s, color 0.25s;
            -webkit-font-smoothing: antialiased;
        }

        /* ── Header (frosted glass bar) ───────────────────────── */
        .header {
            background: var(--bg-glass);
            backdrop-filter: blur(20px) saturate(180%);
            -webkit-backdrop-filter: blur(20px) saturate(180%);
            padding: 0.85rem 1.5rem;
            border-bottom: 1px solid var(--border-color);
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: sticky;
            top: 0;
            z-index: 10;
        }

        .header-content {
            display: flex;
            align-items: center;
            gap: 1rem;
            flex: 1;
            min-width: 0;
        }

        .header-logo {
            width: 2.2rem;
            height: 2.2rem;
            flex-shrink: 0;
            border-radius: 0.55rem;
        }

        .header-title {
            font-size: 1rem;
            font-weight: 700;
            letter-spacing: -0.01em;
            color: var(--text-primary);
            white-space: nowrap;
        }

        .header-divider {
            width: 1px;
            height: 1.1rem;
            background: var(--border-color);
            flex-shrink: 0;
        }

        .status-bar {
            display: flex;
            gap: 1.2rem;
            font-size: 0.8rem;
            color: var(--text-secondary);
            flex-wrap: wrap;
            min-width: 0;
            overflow: hidden;
        }

        .status-item {
            display: flex;
            align-items: center;
            gap: 0.3rem;
            white-space: nowrap;
        }

        .status-label {
            font-weight: 600;
            color: var(--text-label);
        }

        /* ── Theme toggle pill ───────────────────────────────── */
        .theme-toggle {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            padding: 0.3rem 0.7rem;
            cursor: pointer;
            font-size: 1rem;
            line-height: 1;
            transition: background 0.2s, transform 0.15s;
            color: var(--text-primary);
            flex-shrink: 0;
        }

        .theme-toggle:hover { background: var(--border-color); transform: scale(1.06); }

        /* ── Chat area ───────────────────────────────────────── */
        .chat-container {
            flex: 1;
            overflow-y: auto;
            padding: 1.5rem 1.25rem;
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
        }

        /* ── iMessage-style bubbles ──────────────────────────── */
        .message {
            max-width: 72%;
            padding: 0.65rem 1rem;
            border-radius: 18px;
            line-height: 1.6;
            white-space: pre-wrap;
            word-wrap: break-word;
        }

        .message-content {
            display: flex;
            flex-direction: column;
            gap: 0.55rem;
            white-space: normal;
        }

        .message-text {
            white-space: pre-wrap;
            word-break: break-word;
            line-height: 1.55;
            font-size: 0.95rem;
        }

        .file-analysis-block {
            border: 1px solid var(--border-color);
            border-radius: 11px;
            background: var(--bg-primary);
            overflow: hidden;
        }

        .file-analysis-block summary {
            cursor: pointer;
            list-style: none;
            user-select: none;
            display: flex;
            align-items: center;
            gap: 0.45rem;
            padding: 0.5rem 0.7rem;
            font-size: 0.84rem;
            font-weight: 600;
            color: var(--text-primary);
            background: var(--bg-tertiary);
        }

        .file-analysis-block summary::-webkit-details-marker { display: none; }

        .file-analysis-content {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
            padding: 0.62rem 0.7rem 0.72rem;
        }

        .md-table-wrap {
            overflow-x: auto;
            border: 1px solid var(--border-color);
            border-radius: 10px;
            background: var(--bg-primary);
        }

        .md-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.875rem;
            white-space: normal;
            min-width: 520px;
        }

        .md-table th,
        .md-table td {
            border-bottom: 1px solid var(--border-color);
            border-right: 1px solid var(--border-color);
            text-align: left;
            vertical-align: top;
            padding: 0.5rem 0.7rem;
            line-height: 1.45;
            word-break: break-word;
            white-space: normal;
        }

        .md-table th:last-child,
        .md-table td:last-child { border-right: none; }

        .md-table thead th {
            background: var(--bg-tertiary);
            color: var(--text-primary);
            font-weight: 600;
        }

        .md-table tbody tr:last-child td { border-bottom: none; }

        /* ── Bubble shapes (iMessage style) ─────────────────── */
        .message.user {
            align-self: flex-end;
            background: var(--bubble-user);
            color: #ffffff;
            margin-left: auto;
            border-bottom-right-radius: 4px;
        }

        .message.assistant {
            align-self: flex-start;
            background: var(--bubble-assist);
            border: 1px solid var(--border-color);
            border-bottom-left-radius: 4px;
        }

        /* timestamp sits below bubble, outside it */
        .message-timestamp {
            font-size: 0.72rem;
            color: var(--text-label);
            margin-top: 0.3rem;
            padding: 0 0.25rem;
        }

        .message.user + .message-timestamp  { text-align: right; }

        /* ── Bottom input area ───────────────────────────────── */
        .input-container {
            background: var(--bg-glass);
            backdrop-filter: blur(20px) saturate(180%);
            -webkit-backdrop-filter: blur(20px) saturate(180%);
            padding: 0.9rem 1.25rem 1rem;
            border-top: 1px solid var(--border-color);
        }

        .input-wrapper {
            display: flex;
            gap: 0.6rem;
            max-width: 900px;
            margin: 0 auto;
            align-items: center;
        }

        #messageInput {
            flex: 1;
            background: var(--bg-input);
            border: 1.5px solid var(--border-color);
            border-radius: 22px;
            padding: 0.6rem 1.1rem;
            color: var(--text-primary);
            font-size: 0.95rem;
            font-family: inherit;
            transition: border-color 0.18s, box-shadow 0.18s;
            line-height: 1.4;
        }

        #messageInput:focus {
            outline: none;
            border-color: var(--accent-color);
            box-shadow: 0 0 0 3px rgba(10,132,255,0.18);
        }

        #sendButton {
            background: var(--accent-color);
            color: white;
            border: none;
            border-radius: 22px;
            padding: 0.6rem 1.4rem;
            font-size: 0.92rem;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.18s, transform 0.12s;
            white-space: nowrap;
        }

        #sendButton:hover:not(:disabled) {
            background: var(--accent-hover);
            transform: scale(1.03);
        }

        #sendButton:disabled { background: var(--bg-tertiary); cursor: not-allowed; opacity: 0.45; }

        #stopButton {
            background: #ff453a;
            color: white;
            border: none;
            border-radius: 22px;
            padding: 0.6rem 1.1rem;
            font-size: 0.92rem;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.18s, transform 0.12s;
            white-space: nowrap;
        }

        #stopButton:hover:not(:disabled) { background: #d93025; transform: scale(1.03); }
        #stopButton:disabled { background: var(--bg-tertiary); cursor: not-allowed; opacity: 0.45; }

        #dirButton {
            background: var(--bg-tertiary);
            color: var(--text-primary);
            border: 1px solid var(--border-color);
            border-radius: 22px;
            padding: 0.6rem 1.1rem;
            font-size: 0.92rem;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.18s, transform 0.12s;
            white-space: nowrap;
        }

        #dirButton:hover:not(:disabled) { background: var(--border-color); transform: scale(1.03); }
        #dirButton:disabled { cursor: not-allowed; opacity: 0.45; }

        /* ── Change-dir modal ────────────────────────────────── */
        .modal-overlay {
            display: none;
            position: fixed;
            inset: 0;
            background: rgba(0,0,0,0.55);
            z-index: 100;
            align-items: center;
            justify-content: center;
        }

        .modal-overlay.open { display: flex; }

        .modal-box {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1.5rem;
            width: min(520px, 92vw);
            box-shadow: 0 8px 32px var(--shadow-color);
        }

        .modal-title {
            font-size: 1rem;
            font-weight: 700;
            margin-bottom: 0.9rem;
            color: var(--text-primary);
        }

        #dirInput {
            width: 100%;
            background: var(--bg-input);
            border: 1.5px solid var(--border-color);
            border-radius: 10px;
            padding: 0.6rem 0.9rem;
            color: var(--text-primary);
            font-size: 0.93rem;
            font-family: 'Menlo', 'SF Mono', 'Courier New', monospace;
            margin-bottom: 0.9rem;
            transition: border-color 0.18s;
        }

        #dirInput:focus {
            outline: none;
            border-color: var(--accent-color);
            box-shadow: 0 0 0 3px rgba(10,132,255,0.18);
        }

        .modal-actions {
            display: flex;
            justify-content: flex-end;
            gap: 0.5rem;
        }

        .modal-btn {
            border: none;
            border-radius: 8px;
            font-size: 0.88rem;
            font-weight: 600;
            cursor: pointer;
            padding: 0.45rem 1rem;
            transition: background 0.15s;
        }

        .modal-btn.confirm { background: var(--accent-color); color: #fff; }
        .modal-btn.confirm:hover:not(:disabled) { background: var(--accent-hover); }
        .modal-btn.cancel { background: transparent; color: var(--text-primary); border: 1px solid var(--border-color); }
        .modal-btn.cancel:hover:not(:disabled) { background: var(--bg-tertiary); }
        .modal-btn:disabled { cursor: not-allowed; opacity: 0.55; }

        /* ── Typing indicator ────────────────────────────────── */
        .loading {
            display: flex;
            gap: 0.5rem;
            align-items: flex-start;
            color: var(--text-secondary);
            padding: 0.5rem 0.25rem;
        }

        .spinner {
            width: 18px;
            height: 18px;
            flex-shrink: 0;
            border: 2px solid var(--border-color);
            border-top-color: var(--accent-color);
            border-radius: 50%;
            animation: spin 0.85s linear infinite;
            margin-top: 2px;
        }

        .spinner.thinking { border-top-color: var(--reasoning-color); }

        @keyframes spin { to { transform: rotate(360deg); } }
        @keyframes msgIn {
            from { opacity: 0; transform: translateY(6px); }
            to   { opacity: 1; transform: translateY(0); }
        }

        /* ── Hint bar below input ────────────────────────────── */
        .commands-hint {
            max-width: 900px;
            margin: 0.45rem auto 0;
            font-size: 0.78rem;
            color: var(--text-label);
        }

        .commands-hint code {
            background: var(--bg-tertiary);
            padding: 0.1rem 0.35rem;
            border-radius: 4px;
            color: var(--accent-color);
            font-size: 0.75rem;
        }

        /* ── Session prompt accordion ────────────────────────── */
        .session-prompt-panel {
            max-width: 900px;
            margin: 0 auto 0.6rem;
            border: 1px solid var(--border-color);
            border-radius: 12px;
            background: var(--bg-tertiary);
            overflow: hidden;
        }

        .session-prompt-panel summary {
            cursor: pointer;
            padding: 0.55rem 0.9rem;
            font-size: 0.82rem;
            font-weight: 600;
            color: var(--text-secondary);
            user-select: none;
            list-style: none;
            display: flex;
            align-items: center;
            gap: 0.4rem;
        }

        .session-prompt-panel summary::-webkit-details-marker { display: none; }

        .session-prompt-body {
            padding: 0.7rem 0.9rem 0.85rem;
            border-top: 1px solid var(--border-color);
        }

        #sessionPromptInput {
            width: 100%;
            min-height: 90px;
            resize: vertical;
            background: var(--bg-input);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            color: var(--text-primary);
            font-size: 0.88rem;
            font-family: inherit;
            line-height: 1.5;
            padding: 0.6rem 0.75rem;
            margin-bottom: 0.5rem;
            transition: border-color 0.18s;
        }

        #sessionPromptInput:focus {
            outline: none;
            border-color: var(--accent-color);
            box-shadow: 0 0 0 3px rgba(10,132,255,0.15);
        }

        .session-prompt-actions {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            flex-wrap: wrap;
        }

        .session-prompt-btn {
            border: none;
            border-radius: 8px;
            font-size: 0.8rem;
            font-weight: 600;
            cursor: pointer;
            padding: 0.38rem 0.75rem;
            transition: background 0.15s, transform 0.1s;
        }

        .session-prompt-btn.apply { background: var(--accent-color); color: #fff; }
        .session-prompt-btn.apply:hover:not(:disabled) { background: var(--accent-hover); transform: scale(1.03); }
        .session-prompt-btn.clear { background: transparent; color: var(--text-primary); border: 1px solid var(--border-color); }
        .session-prompt-btn.clear:hover:not(:disabled) { background: var(--bg-secondary); }
        .session-prompt-btn:disabled { cursor: not-allowed; opacity: 0.55; }

        .session-prompt-state { font-size: 0.77rem; color: var(--text-secondary); }

        .session-prompt-file-row {
            display: flex;
            align-items: center;
            gap: 0.6rem;
            flex-wrap: wrap;
            margin-bottom: 0.55rem;
        }

        .session-prompt-file-input { display: none; }

        .session-prompt-file-status {
            display: inline-flex;
            align-items: center;
            gap: 0.4rem;
            min-height: 1.5rem;
            color: var(--text-secondary);
        }

        .session-prompt-file-dot {
            width: 0.56rem;
            height: 0.56rem;
            border-radius: 999px;
            background: #8e8e93;
            flex-shrink: 0;
        }

        .session-prompt-file-status.loaded .session-prompt-file-dot {
            background: #30d158;
        }

        .session-prompt-file-name {
            font-size: 0.77rem;
            max-width: 420px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        /* ── Copy button ────────────────────────────────────── */
        .message-actions {
            display: flex;
            justify-content: flex-end;
            margin-top: 0.4rem;
        }

        .copy-btn {
            background: transparent;
            border: 1px solid var(--border-color);
            border-radius: 7px;
            color: var(--text-secondary);
            cursor: pointer;
            font-size: 0.72rem;
            padding: 0.18rem 0.55rem;
            transition: background 0.15s, color 0.15s;
        }

        .copy-btn:hover { background: var(--bg-tertiary); color: var(--text-primary); }
        .copy-btn.copied { color: #30d158; border-color: #30d158; }

        /* ── Reasoning collapsible ───────────────────────────── */
        .reasoning-block {
            margin: 0.4rem 0;
            border: 1px solid rgba(191,90,242,0.25);
            border-radius: 10px;
            overflow: hidden;
        }

        .reasoning-block summary {
            cursor: pointer;
            padding: 0.38rem 0.7rem;
            background: rgba(191,90,242,0.08);
            color: var(--reasoning-color);
            font-weight: 600;
            font-size: 0.8rem;
            user-select: none;
            list-style: none;
            display: flex;
            align-items: center;
            gap: 0.35rem;
        }

        .reasoning-block summary::-webkit-details-marker { display: none; }
        .reasoning-block[open] summary { border-bottom: 1px solid rgba(191,90,242,0.18); }

        .reasoning-content {
            padding: 0.7rem;
            margin: 0;
            font-size: 0.77rem;
            color: var(--text-secondary);
            white-space: pre-wrap;
            word-break: break-word;
            font-family: 'Menlo', 'SF Mono', 'Courier New', monospace;
            background: rgba(0,0,0,0.15);
            max-height: 280px;
            overflow-y: auto;
        }

        .reasoning-preview {
            font-size: 0.77rem;
            color: var(--reasoning-color);
            font-family: 'Menlo', 'SF Mono', 'Courier New', monospace;
            max-height: 110px;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-all;
            margin-top: 0.35rem;
            padding: 0.38rem 0.5rem;
            background: rgba(191,90,242,0.05);
            border-radius: 6px;
            border-left: 2px solid rgba(191,90,242,0.35);
        }

        /* ── Scrollbars (thin, macOS style) ─────────────────── */
        ::-webkit-scrollbar { width: 6px; height: 6px; }
        ::-webkit-scrollbar-track { background: var(--scrollbar-track); }
        ::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 3px; }
        ::-webkit-scrollbar-thumb:hover { background: var(--scrollbar-thumb-hover); }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <img src="/static/favicon.png" alt="Local Agent" class="header-logo">
            <span class="header-title">local-agent</span>
            <div class="header-divider"></div>
            <div class="status-bar">
                <div class="status-item">
                    <span class="status-label">dir</span>
                    <span id="directory">-</span>
                </div>
                <div class="status-item">
                    <span class="status-label">model</span>
                    <span id="model">-</span>
                </div>
                <div class="status-item">
                    <span class="status-label">files</span>
                    <span id="totalFiles">-</span>
                </div>
                <div class="status-item" id="focusItem" style="display:none;">
                    <span class="status-label">focus</span>
                    <span id="focusedPath">-</span>
                </div>
                <div class="status-item" id="thinkingIndicator" style="display:none;">
                    <span style="color:var(--reasoning-color);font-weight:600;">🧠 thinking</span>
                </div>
                <div class="status-item" id="sessionPromptIndicator" style="display:none;">
                    <span style="color:#30d158;font-weight:600;">● prompt</span>
                </div>
            </div>
        </div>
        <button class="theme-toggle" id="themeToggle" title="Toggle theme">🌙</button>
    </div>

    <div class="chat-container" id="chatContainer"></div>

    <div class="input-container">
        <details class="session-prompt-panel" id="sessionPromptPanel">
            <summary>⚙ Session prompt<span id="sessionPromptSummaryBadge" style="display:none;margin-left:0.45rem;color:#30d158;font-size:0.85em;" title="Session prompt is active">●</span></summary>
            <div class="session-prompt-body">
                <div class="session-prompt-file-row">
                    <button id="sessionPromptLoadFile" type="button" class="session-prompt-btn clear">Load prompt file</button>
                    <button id="sessionPromptDetachFile" type="button" class="session-prompt-btn clear" disabled>Detach file</button>
                    <input id="sessionPromptFileInput" class="session-prompt-file-input" type="file">
                    <span id="sessionPromptFileStatus" class="session-prompt-file-status" title="No file attached">
                        <span class="session-prompt-file-dot" aria-hidden="true"></span>
                        <span id="sessionPromptFileName" class="session-prompt-file-name">No file attached</span>
                    </span>
                </div>
                <textarea id="sessionPromptInput" placeholder="Optional extra instructions for every message in this session…"></textarea>
                <div class="session-prompt-actions">
                    <button id="sessionPromptApply" class="session-prompt-btn apply">Apply</button>
                    <button id="sessionPromptClear" class="session-prompt-btn clear">Clear</button>
                    <span id="sessionPromptState" class="session-prompt-state">Not set.</span>
                </div>
            </div>
        </details>
        <div class="input-wrapper">
            <input
                type="text"
                id="messageInput"
                placeholder="Ask about your codebase…"
                autocomplete="off"
            />
            <button id="sendButton">Send</button>
            <button id="stopButton" disabled>Stop</button>
            <button id="dirButton" title="Change working directory">Dir</button>
        </div>
        <div class="commands-hint">
            type <code>help</code> for commands
        </div>
    </div>

    <div class="modal-overlay" id="dirModal">
        <div class="modal-box">
            <div class="modal-title">📂 Change working directory</div>
            <input type="text" id="dirInput" placeholder="/path/to/project" autocomplete="off" spellcheck="false">
            <div class="modal-actions">
                <button class="modal-btn cancel" id="dirModalCancel">Cancel</button>
                <button class="modal-btn confirm" id="dirModalConfirm">Change</button>
            </div>
        </div>
    </div>

    <script>
        const chatContainer = document.getElementById('chatContainer');
        const messageInput = document.getElementById('messageInput');
        const sendButton = document.getElementById('sendButton');
        const stopButton = document.getElementById('stopButton');
        const dirButton = document.getElementById('dirButton');
        const dirModal = document.getElementById('dirModal');
        const dirInput = document.getElementById('dirInput');
        const dirModalCancel = document.getElementById('dirModalCancel');
        const dirModalConfirm = document.getElementById('dirModalConfirm');
        const sessionPromptPanel = document.getElementById('sessionPromptPanel');
        const sessionPromptInput = document.getElementById('sessionPromptInput');
        const sessionPromptApplyButton = document.getElementById('sessionPromptApply');
        const sessionPromptClearButton = document.getElementById('sessionPromptClear');
        const sessionPromptLoadFileButton = document.getElementById('sessionPromptLoadFile');
        const sessionPromptDetachFileButton = document.getElementById('sessionPromptDetachFile');
        const sessionPromptFileInput = document.getElementById('sessionPromptFileInput');
        const sessionPromptFileStatus = document.getElementById('sessionPromptFileStatus');
        const sessionPromptFileName = document.getElementById('sessionPromptFileName');
        const sessionPromptState = document.getElementById('sessionPromptState');
        let isProcessing = false;
        let isThinkingModel = false;
        let sessionPromptDirty = false;
        let sessionPromptActiveOnServer = false;
        let sessionPromptAttachedFileName = '';
        let sessionPromptAttachedFilePrompt = '';
        const sessionPromptAttachedFileStorageKey = 'localAgent.sessionPromptAttachedFile';

        function updateSessionPromptIndicator() {
            const sessionPromptIndicator = document.getElementById('sessionPromptIndicator');
            const sessionPromptSummaryBadge = document.getElementById('sessionPromptSummaryBadge');

            const shouldShow = sessionPromptActiveOnServer || !!sessionPromptAttachedFileName;
            if (sessionPromptIndicator) {
                sessionPromptIndicator.style.display = shouldShow ? 'flex' : 'none';
            }
            if (sessionPromptSummaryBadge) {
                sessionPromptSummaryBadge.style.display = shouldShow ? 'inline' : 'none';
            }
        }

        function persistSessionPromptAttachedFile() {
            try {
                if (!sessionPromptAttachedFileName) {
                    sessionStorage.removeItem(sessionPromptAttachedFileStorageKey);
                    return;
                }

                sessionStorage.setItem(sessionPromptAttachedFileStorageKey, JSON.stringify({
                    fileName: sessionPromptAttachedFileName,
                    promptText: sessionPromptAttachedFilePrompt,
                }));
            } catch (error) {
                // Ignore storage failures (private mode, quota, disabled storage).
            }
        }

        function restoreSessionPromptAttachedFile() {
            try {
                const raw = sessionStorage.getItem(sessionPromptAttachedFileStorageKey);
                if (!raw) {
                    setSessionPromptAttachedFile('', '');
                    return;
                }

                const data = JSON.parse(raw);
                if (!data || typeof data.fileName !== 'string' || typeof data.promptText !== 'string') {
                    setSessionPromptAttachedFile('', '');
                    return;
                }

                setSessionPromptAttachedFile(data.fileName, data.promptText);
            } catch (error) {
                setSessionPromptAttachedFile('', '');
            }
        }

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

                const hasSessionPrompt = !!data.hasSessionPrompt;
                sessionPromptActiveOnServer = hasSessionPrompt;
                updateSessionPromptIndicator();

                if (sessionPromptInput && (!sessionPromptDirty || document.activeElement !== sessionPromptInput)) {
                    sessionPromptInput.value = data.sessionPrompt || '';
                }

                if (!sessionPromptDirty) {
                    updateSessionPromptState(hasSessionPrompt ? 'Session prompt is active for this session.' : 'Session prompt is not set.', false);
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

        function updateSessionPromptState(message, isError) {
            if (!sessionPromptState) {
                return;
            }

            sessionPromptState.textContent = message;
            sessionPromptState.style.color = isError ? '#ef4444' : 'var(--text-secondary)';
        }

        function normalizePromptText(text) {
            return String(text || '').replace(/\r\n/g, '\n').trim();
        }

        function setSessionPromptAttachedFile(fileName, promptText) {
            sessionPromptAttachedFileName = fileName || '';
            sessionPromptAttachedFilePrompt = String(promptText || '');

            const loaded = !!sessionPromptAttachedFileName;
            if (sessionPromptFileStatus && sessionPromptFileName) {
                sessionPromptFileStatus.classList.toggle('loaded', loaded);
                sessionPromptFileName.textContent = loaded ? sessionPromptAttachedFileName : 'No file attached';
                sessionPromptFileStatus.title = loaded ? sessionPromptAttachedFileName : 'No file attached';
            }
            if (sessionPromptDetachFileButton) {
                sessionPromptDetachFileButton.disabled = !loaded;
            }
            persistSessionPromptAttachedFile();
            updateSessionPromptIndicator();
        }

        function detachSessionPromptFile() {
            setSessionPromptAttachedFile('', '');
        }

        async function loadSessionPromptFromFile(file) {
            if (!file || !sessionPromptInput) {
                return;
            }

            try {
                const content = await file.text();
                sessionPromptInput.value = content;
                setSessionPromptAttachedFile(file.name, content);
                sessionPromptDirty = true;
                if (sessionPromptPanel) {
                    sessionPromptPanel.open = true;
                }
                updateSessionPromptState('Loaded prompt from file "' + file.name + '". Click Apply to activate.', false);
            } catch (error) {
                updateSessionPromptState('Failed to load prompt file: ' + error.message, true);
                detachSessionPromptFile();
            } finally {
                if (sessionPromptFileInput) {
                    sessionPromptFileInput.value = '';
                }
            }
        }

        async function saveSessionPrompt(prompt) {
            const response = await fetch('/api/session-prompt', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ prompt: prompt }),
            });

            const data = await response.json();
            if (!response.ok || !data.success) {
                throw new Error(data.error || 'Failed to save session prompt');
            }

            return data;
        }

        async function applySessionPrompt() {
            if (!sessionPromptInput) {
                return;
            }

            const promptRaw = sessionPromptInput.value;
            const prompt = promptRaw.trim();
            const normalizedPrompt = normalizePromptText(promptRaw);
            const shouldDetachFile = !!sessionPromptAttachedFileName && normalizedPrompt !== '' && normalizedPrompt !== normalizePromptText(sessionPromptAttachedFilePrompt);
            if (shouldDetachFile) {
                detachSessionPromptFile();
            }

            sessionPromptApplyButton.disabled = true;
            sessionPromptClearButton.disabled = true;

            try {
                await saveSessionPrompt(prompt);
                sessionPromptDirty = false;
                sessionPromptActiveOnServer = prompt !== '';
                updateSessionPromptIndicator();
                let statusMessage = prompt ? 'Session prompt saved and active.' : 'Session prompt is not set.';
                if (shouldDetachFile) {
                    statusMessage += ' Attached prompt file was detached after edits.';
                }
                updateSessionPromptState(statusMessage, false);
                await loadStatus();
            } catch (error) {
                updateSessionPromptState('Failed to save session prompt: ' + error.message, true);
            } finally {
                sessionPromptApplyButton.disabled = false;
                sessionPromptClearButton.disabled = false;
            }
        }

        async function clearSessionPrompt() {
            if (!sessionPromptInput) {
                return;
            }

            sessionPromptInput.value = '';
            sessionPromptDirty = true;
            await applySessionPrompt();
        }

        function splitMarkdownRow(line) {
            let text = line.trim();
            if (text.startsWith('|')) {
                text = text.slice(1);
            }
            if (text.endsWith('|')) {
                text = text.slice(0, -1);
            }
            return text.split('|').map(function(cell) {
                return cell.trim();
            });
        }

        function isMarkdownSeparator(line) {
            const cells = splitMarkdownRow(line);
            if (cells.length < 2) {
                return false;
            }

            return cells.every(function(cell) {
                if (!cell || cell.indexOf('-') === -1) {
                    return false;
                }
                return /^:?-{3,}:?$/.test(cell);
            });
        }

        function isMarkdownTableStart(lines, idx) {
            if (idx + 1 >= lines.length) {
                return false;
            }

            const header = lines[idx].trim();
            if (!header || header.indexOf('|') === -1) {
                return false;
            }

            return isMarkdownSeparator(lines[idx + 1]);
        }

        function renderMarkdownTable(lines, start, container) {
            const headers = splitMarkdownRow(lines[start]);
            const colCount = Math.max(headers.length, 2);

            const tableWrap = document.createElement('div');
            tableWrap.className = 'md-table-wrap';

            const table = document.createElement('table');
            table.className = 'md-table';

            const thead = document.createElement('thead');
            const headerRow = document.createElement('tr');
            for (let i = 0; i < colCount; i++) {
                const th = document.createElement('th');
                th.textContent = (headers[i] || ('Column ' + (i + 1))).trim();
                headerRow.appendChild(th);
            }
            thead.appendChild(headerRow);
            table.appendChild(thead);

            const tbody = document.createElement('tbody');
            let i = start + 2;
            while (i < lines.length) {
                const line = lines[i];
                const trimmed = line.trim();
                if (!trimmed || trimmed.indexOf('|') === -1) {
                    break;
                }

                const cells = splitMarkdownRow(trimmed);
                const tr = document.createElement('tr');
                for (let c = 0; c < colCount; c++) {
                    const td = document.createElement('td');
                    td.textContent = (cells[c] || '').trim();
                    tr.appendChild(td);
                }
                tbody.appendChild(tr);
                i++;
            }

            table.appendChild(tbody);
            tableWrap.appendChild(table);
            container.appendChild(tableWrap);

            return i - start;
        }

        function appendTextBlock(lines, container) {
            if (!lines || lines.length === 0) {
                return;
            }

            const text = lines.join('\n');
            if (!text.trim()) {
                return;
            }

            const div = document.createElement('div');
            div.className = 'message-text';
            div.textContent = text;
            container.appendChild(div);
        }

        function renderMarkdownAwareText(text, container) {
            const lines = text.split('\n');
            let plainBuffer = [];
            let inCodeFence = false;
            const codeFenceMarker = String.fromCharCode(96) + String.fromCharCode(96) + String.fromCharCode(96);

            function flushText() {
                appendTextBlock(plainBuffer, container);
                plainBuffer = [];
            }

            for (let i = 0; i < lines.length; ) {
                const line = lines[i];
                const trimmed = line.trim();

                if (trimmed.startsWith(codeFenceMarker)) {
                    inCodeFence = !inCodeFence;
                    plainBuffer.push(line);
                    i++;
                    continue;
                }

                if (!inCodeFence && isMarkdownTableStart(lines, i)) {
                    flushText();
                    const consumed = renderMarkdownTable(lines, i, container);
                    i += consumed;
                    continue;
                }

                plainBuffer.push(line);
                i++;
            }

            flushText();
        }

        function splitFileAnalysisSections(content) {
            const lines = String(content || '').split('\n');
            const sections = [];
            const preamble = [];
            let activeSection = null;

            for (let i = 0; i < lines.length; i++) {
                const line = lines[i];
                const match = line.match(/^===\s*(.+?)\s*===\s*$/);

                if (match) {
                    if (activeSection) {
                        sections.push(activeSection);
                    }
                    activeSection = {
                        name: match[1],
                        lines: [],
                    };
                    continue;
                }

                if (activeSection) {
                    activeSection.lines.push(line);
                } else {
                    preamble.push(line);
                }
            }

            if (activeSection) {
                sections.push(activeSection);
            }

            return {
                preamble: preamble.join('\n'),
                sections: sections,
            };
        }

        // Render message content, turning [reasoning]...[/reasoning] blocks into
        // collapsible <details> elements and rendering markdown tables in answer text.
        function renderReasoningAndMarkdown(content, container) {
            const re = /\[reasoning\]([\s\S]*?)\[\/reasoning\]\n?/g;
            let lastIndex = 0;
            let match;
            let found = false;
            while ((match = re.exec(content)) !== null) {
                found = true;
                const before = content.slice(lastIndex, match.index);
                if (before.trim()) {
                    renderMarkdownAwareText(before, container);
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
                renderMarkdownAwareText(remaining || content, container);
            }
        }

        // Render assistant message content with per-file grouping for responses that
        // use "=== file ===" separators.
        function renderContent(content, container) {
            const parsed = splitFileAnalysisSections(content);

            if (!parsed.sections.length) {
                renderReasoningAndMarkdown(content, container);
                return;
            }

            if (parsed.preamble.trim()) {
                renderReasoningAndMarkdown(parsed.preamble, container);
            }

            for (let i = 0; i < parsed.sections.length; i++) {
                const section = parsed.sections[i];
                const details = document.createElement('details');
                details.className = 'file-analysis-block';
                details.open = true;

                const summary = document.createElement('summary');
                summary.textContent = '\uD83D\uDCC4 ' + section.name;

                const body = document.createElement('div');
                body.className = 'file-analysis-content';

                const sectionText = section.lines.join('\n');
                if (sectionText.trim()) {
                    renderReasoningAndMarkdown(sectionText, body);
                } else {
                    const empty = document.createElement('div');
                    empty.className = 'message-text';
                    empty.textContent = 'No content returned for this file.';
                    body.appendChild(empty);
                }

                details.appendChild(summary);
                details.appendChild(body);
                container.appendChild(details);
            }
        }

        // Add message to chat
        function addMessage(role, content, timestamp) {
            // Wrapper keeps bubble + timestamp together
            const wrapper = document.createElement('div');
            wrapper.style.cssText = 'display:flex;flex-direction:column;' + (role === 'user' ? 'align-items:flex-end;' : 'align-items:flex-start;');
            wrapper.style.animation = 'msgIn 0.18s ease';

            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + role;

            const contentDiv = document.createElement('div');
            contentDiv.className = 'message-content';

            if (role === 'assistant') {
                renderContent(content, contentDiv);
            } else {
                const div = document.createElement('div');
                div.className = 'message-text';
                div.textContent = content;
                contentDiv.appendChild(div);
            }

            messageDiv.appendChild(contentDiv);

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

            // Timestamp sits below bubble, outside it
            const timeDiv = document.createElement('div');
            timeDiv.className = 'message-timestamp';
            timeDiv.textContent = new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'});

            wrapper.appendChild(messageDiv);
            wrapper.appendChild(timeDiv);
            chatContainer.appendChild(wrapper);
        }

        // Show loading indicator
        function showLoading() {
            const loadingDiv = document.createElement('div');
            loadingDiv.className = 'loading';
            loadingDiv.id = 'loading';
            const innerContent =
                '<div style="display:flex;flex-direction:column;gap:0.4rem;flex:1;min-width:0">' +
                    '<div style="display:flex;align-items:center;gap:0.5rem">' +
                        '<div class="spinner' + (isThinkingModel ? ' thinking' : '') + '"></div>' +
                        '<span id="loadingText">Analyzing...</span>' +
                    '</div>' +
                    '<ul id="activeFileList" style="list-style:none;margin:0;padding:0 0 0 0.25rem;display:none"></ul>' +
                    (isThinkingModel ? '<div id="reasoningPreview" class="reasoning-preview" style="display:none"></div>' : '') +
                '</div>';
            loadingDiv.innerHTML = innerContent;
            chatContainer.appendChild(loadingDiv);
            scrollToBottom();
        }

        const _activeFiles = new Map(); // name -> start timestamp (ms)
        const _doneFiles = new Map();   // name -> elapsed string

        function _renderActiveList() {
            const ul = document.getElementById('activeFileList');
            if (!ul) return;
            ul.innerHTML = '';
            const now = Date.now();
            _activeFiles.forEach(function(startMs, name) {
                const elapsed = ((now - startMs) / 1000).toFixed(1);
                const li = document.createElement('li');
                li.style.cssText = 'font-size:0.78rem;color:var(--text-secondary);padding:0.1rem 0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;';
                li.textContent = '\u23f3 ' + name + ' (' + elapsed + 's…)';
                ul.appendChild(li);
            });
            _doneFiles.forEach(function(elapsed, name) {
                const li = document.createElement('li');
                li.style.cssText = 'font-size:0.78rem;color:var(--text-secondary);padding:0.1rem 0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;opacity:0.75;';
                li.textContent = '\u2713 ' + name + ' (' + elapsed + ')';
                ul.appendChild(li);
            });
            ul.style.display = (_activeFiles.size + _doneFiles.size) > 0 ? 'block' : 'none';
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
            stopButton.disabled = false;
            messageInput.disabled = true;
            _activeFiles.clear();
            _doneFiles.clear();
            _renderActiveList();

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
                } else if (e.data.startsWith('ANALYZING:')) {
                    const name = e.data.substring(10);
                    _activeFiles.set(name, Date.now());
                    _renderActiveList();
                } else if (e.data.startsWith('DONE:')) {
                    // "DONE:<name>:<elapsed>" — file finished, move to done list
                    const rest = e.data.substring(5);
                    const lastColon = rest.lastIndexOf(':');
                    if (lastColon !== -1) {
                        const name = rest.substring(0, lastColon);
                        const elapsed = rest.substring(lastColon + 1);
                        _activeFiles.delete(name);
                        _doneFiles.set(name, elapsed);
                    }
                    _renderActiveList();
                } else if (e.data.startsWith('Reviewed ')) {
                    // "Reviewed N/M: filename" — update progress text
                    _renderActiveList();
                    updateLoadingText(e.data);
                    scrollToBottom();
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
                    const errText = data.error || 'Unknown error';
                    if (errText.toLowerCase().includes('stop')) {
                        addMessage('assistant', '⏹️ Request stopped.', new Date().toISOString());
                    } else {
                        addMessage('assistant', '❌ Error: ' + errText, new Date().toISOString());
                    }
                }
            } catch (error) {
                hideLoading();
                addMessage('assistant', '❌ Network error: ' + error.message, new Date().toISOString());
            } finally {
                isProcessing = false;
                sendButton.disabled = false;
                stopButton.disabled = true;
                stopButton.textContent = 'Stop';
                messageInput.disabled = false;
                messageInput.focus();
                _activeFiles.clear();
                _doneFiles.clear();
                _renderActiveList();
            }
        }

        async function stopProcessing() {
            if (!isProcessing) {
                return;
            }

            stopButton.disabled = true;
            stopButton.textContent = 'Stopping...';

            try {
                await fetch('/api/stop', {
                    method: 'POST',
                });
            } catch (error) {
                addMessage('assistant', '⚠️ Failed to stop request: ' + error.message, new Date().toISOString());
                stopButton.disabled = false;
                stopButton.textContent = 'Stop';
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

        // Dir button / change directory
        function openDirModal() {
            dirInput.value = document.getElementById('directory').textContent.trim() || '';
            dirModal.classList.add('open');
            setTimeout(() => { dirInput.focus(); dirInput.select(); }, 50);
        }

        function closeDirModal() {
            dirModal.classList.remove('open');
        }

        async function changeDir() {
            const path = dirInput.value.trim();
            if (!path) return;

            dirModalConfirm.disabled = true;
            dirModalCancel.disabled = true;

            try {
                const response = await fetch('/api/changedir', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path }),
                });
                const data = await response.json();
                closeDirModal();

                if (data.success && data.message) {
                    addMessage(data.message.role, data.message.content, data.message.timestamp);
                    scrollToBottom();
                    await loadStatus();
                } else {
                    addMessage('assistant', '\u274c ' + (data.error || 'Failed to change directory'), new Date().toISOString());
                    scrollToBottom();
                }
            } catch (error) {
                closeDirModal();
                addMessage('assistant', '\u274c Network error: ' + error.message, new Date().toISOString());
                scrollToBottom();
            } finally {
                dirModalConfirm.disabled = false;
                dirModalCancel.disabled = false;
            }
        }

        dirButton.addEventListener('click', openDirModal);
        dirModalCancel.addEventListener('click', closeDirModal);
        dirModalConfirm.addEventListener('click', changeDir);
        dirModal.addEventListener('click', (e) => { if (e.target === dirModal) closeDirModal(); });
        dirInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') changeDir();
            if (e.key === 'Escape') closeDirModal();
        });

        // Event listeners
        sendButton.addEventListener('click', sendMessage);
        stopButton.addEventListener('click', stopProcessing);
        if (sessionPromptLoadFileButton && sessionPromptFileInput) {
            sessionPromptLoadFileButton.addEventListener('click', () => {
                sessionPromptFileInput.click();
            });
            sessionPromptFileInput.addEventListener('change', (e) => {
                const file = e.target.files && e.target.files[0];
                if (!file) {
                    return;
                }
                loadSessionPromptFromFile(file);
            });
        }
        if (sessionPromptDetachFileButton) {
            sessionPromptDetachFileButton.addEventListener('click', () => {
                if (!sessionPromptAttachedFileName) {
                    return;
                }
                detachSessionPromptFile();
                sessionPromptDirty = true;
                updateSessionPromptState('Prompt file detached. Current prompt text is unchanged.', false);
            });
        }
        if (sessionPromptInput) {
            sessionPromptInput.addEventListener('input', () => {
                sessionPromptDirty = true;
                updateSessionPromptState('Session prompt changed. Click Apply to activate.', false);
            });
        }
        if (sessionPromptApplyButton) {
            sessionPromptApplyButton.addEventListener('click', applySessionPrompt);
        }
        if (sessionPromptClearButton) {
            sessionPromptClearButton.addEventListener('click', clearSessionPrompt);
        }
        messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });

        // Initialize
        restoreSessionPromptAttachedFile();
        loadStatus();
        loadMessages();
        messageInput.focus();

        // Auto-refresh status every 5 seconds
        setInterval(loadStatus, 5000);
    </script>
</body>
</html>
`
