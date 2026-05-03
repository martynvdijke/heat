const chatMessages = document.getElementById('chat-messages')!;
const chatInput = document.getElementById('chat-input') as HTMLInputElement;
const charCount = document.getElementById('char-count')!;
const viewerCount = document.getElementById('viewer-count')!;
let ws: WebSocket | null = null;
let username = 'Fan' + Math.floor(Math.random() * 1000);

function connect(): void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onopen = () => {
        addSystemMessage('Connected to live feed');
        sendJoinMessage();
    };

    ws.onmessage = (event: MessageEvent) => {
        try {
            const data = JSON.parse(event.data);
            handleMessage(data);
        } catch (e) {
            console.log('Received:', event.data);
        }
    };

    ws.onclose = () => {
        addSystemMessage('Disconnected. Reconnecting...');
        setTimeout(connect, 3000);
    };
}

interface WsMessage {
    type: string;
    text?: string;
    author?: string;
    flag?: string;
    racers?: unknown[];
    username?: string;
}

function handleMessage(data: WsMessage): void {
    if (data.type === 'commentary') {
        addCommentaryMessage(data.text || '', data.author);
    } else if (data.type === 'flag') {
        handleFlagEvent(data.flag || '');
    } else if (data.racers) {
        updateViewerCount(data.racers.length || 1);
    }
}

function sendMessage(): void {
    const text = chatInput.value.trim();
    if (text && ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'chat', text, username }));
        chatInput.value = '';
        charCount.textContent = '0';
    }
}

function sendJoinMessage(): void {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'join', username }));
    }
}

function addMessage(text: string, type: string = 'user'): void {
    const div = document.createElement('div');
    div.className = `message message-${type}`;
    div.innerHTML = type === 'system' ? text : `
        <small class="opacity-50">${new Date().toLocaleTimeString()}</small><br>${text}
    `;
    chatMessages.appendChild(div);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

function addCommentaryMessage(text: string, author: string = 'Commentator'): void {
    const div = document.createElement('div');
    div.className = 'message message-commentary';
    div.innerHTML = `
        <small class="text-warning"><i class="fa-solid fa-microphone me-1"></i>${author}</small><br>
        <strong>${text}</strong>
    `;
    chatMessages.appendChild(div);
    chatMessages.scrollTop = chatMessages.scrollHeight;
    playNotification();
}

function addSystemMessage(text: string): void {
    const div = document.createElement('div');
    div.className = 'message message-system';
    div.innerHTML = `<small class="opacity-75">${text}</small>`;
    chatMessages.appendChild(div);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

function handleFlagEvent(flag: string): void {
    const messages: Record<string, string> = {
        'yellow': '⚠️ YELLOW FLAG - Caution ahead',
        'safety': '🚨 SAFETY CAR - Clear the track',
        'chequered': '🏁 CHEQUERED FLAG - Race finished!'
    };
    addSystemMessage(messages[flag] || flag);
    playNotification();
}

function updateViewerCount(count: number): void {
    viewerCount.textContent = count + ' viewers';
}

function toggleEmoji(): void {
}

function playNotification(): void {
    if (Notification.permission === 'granted') {
        new Notification('HEAT Racing', { body: 'New commentary!' });
    }
}

chatInput.addEventListener('input', () => {
    charCount.textContent = String(chatInput.value.length);
});

chatInput.addEventListener('keypress', (e: KeyboardEvent) => {
    if (e.key === 'Enter') sendMessage();
});

if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission();
}

connect();

export {};
