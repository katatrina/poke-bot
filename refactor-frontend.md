Đúng rồi! **Hard limit + force new session** là approach **SẠCH HƠN** và **UX RÕ RÀNG** hơn nhiều.

## 🎯 Design: Hard Limit with Clear UX

### 1. **Define Limits**

```yaml
# config.yaml
rag:
  chunk_size: 600
  chunk_overlap: 100
  top_k: 5
  temperature: 0.3
  max_conversation_turns: 15    # ← NEW: Max 15 turns (30 messages)
  max_total_characters: 10000   # ← NEW: ~2500 tokens total
```

```go
// internal/config/config.go
type RAGConfig struct {
    ChunkSize            int `yaml:"chunk_size"`
    ChunkOverlap         int `yaml:"chunk_overlap"`
    TopK                 int `yaml:"top_k"`
    Temperature          float64 `yaml:"temperature"`
    MaxConversationTurns int `yaml:"max_conversation_turns"`
    MaxTotalCharacters   int `yaml:"max_total_characters"`
}
```

### 2. **Backend Validation** (Hard Reject)

```go
// internal/service/rag.go
func (req ChatRequest) Validate() error {
    // Validate message
    if len(req.Message) == 0 {
        return errors.New("message cannot be empty")
    }
    if len(req.Message) > 1000 {
        return model.ErrMessageTooLong
    }

    // Hard limit on conversation length
    if len(req.ConversationHistory) > 30 { // 15 turns = 30 messages
        return ErrConversationTooLong
    }

    // Validate total size
    totalChars := len(req.Message)
    for _, msg := range req.ConversationHistory {
        if msg.Type != "user" && msg.Type != "assistant" {
            return fmt.Errorf("invalid message type: %s", msg.Type)
        }
        totalChars += len(msg.Content)
    }

    // Hard limit on total characters
    if totalChars > 10000 {
        return ErrConversationTooLong
    }

    return nil
}

// Add custom error
var ErrConversationTooLong = errors.New("conversation too long, please start a new chat session")
```

```go
// internal/handler/handler.go
func (hdl *HTTPHandler) Chat(c *gin.Context) {
    var req service.ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid request format",
            "details": err.Error(),
        })
        return
    }

    if err := req.Validate(); err != nil {
        // Special handling for conversation too long
        if errors.Is(err, service.ErrConversationTooLong) {
            c.JSON(http.StatusRequestEntityTooLarge, gin.H{
                "error":   "conversation_too_long",
                "message": "This conversation has reached the maximum length. Please start a new chat session to continue.",
                "details": err.Error(),
            })
            return
        }

        c.JSON(http.StatusBadRequest, gin.H{
            "error": err.Error(),
        })
        return
    }

    // Process the chat request
    resp, err := hdl.ragService.Chat(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "Failed to process chat request",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, resp)
}
```

### 3. **Frontend: Proactive Warning & Blocking**

```typescript
// web/src/types/index.ts
export const CONVERSATION_LIMITS = {
    MAX_TURNS: 15,           // 30 messages total
    MAX_TOTAL_CHARS: 10000,  // ~2500 tokens
    WARNING_THRESHOLD: 12,   // Warn at 12 turns (24 messages)
} as const;
```

```typescript
// web/src/hooks/useConversationLimits.ts
import { useMemo } from 'react';
import { CONVERSATION_LIMITS } from '../types';
import type { Message } from '../types';

interface ConversationLimits {
    currentTurns: number;
    maxTurns: number;
    totalChars: number;
    maxChars: number;
    isNearLimit: boolean;
    isAtLimit: boolean;
    warningMessage: string | null;
    blockedMessage: string | null;
}

export function useConversationLimits(messages: Message[]): ConversationLimits {
    return useMemo(() => {
        const turns = Math.floor(messages.length / 2);
        const totalChars = messages.reduce((sum, msg) => sum + msg.content.length, 0);
        
        const turnsRemaining = CONVERSATION_LIMITS.MAX_TURNS - turns;
        const isNearLimit = turns >= CONVERSATION_LIMITS.WARNING_THRESHOLD;
        const isAtLimit = turns >= CONVERSATION_LIMITS.MAX_TURNS || 
                          totalChars >= CONVERSATION_LIMITS.MAX_TOTAL_CHARS;

        let warningMessage: string | null = null;
        let blockedMessage: string | null = null;

        if (isAtLimit) {
            blockedMessage = 
                "This conversation has reached the maximum length. Please start a new chat to continue.";
        } else if (isNearLimit) {
            warningMessage = 
                `This conversation is approaching its limit (${turnsRemaining} ${turnsRemaining === 1 ? 'turn' : 'turns'} remaining). Consider starting a new chat soon.`;
        }

        return {
            currentTurns: turns,
            maxTurns: CONVERSATION_LIMITS.MAX_TURNS,
            totalChars,
            maxChars: CONVERSATION_LIMITS.MAX_TOTAL_CHARS,
            isNearLimit,
            isAtLimit,
            warningMessage,
            blockedMessage,
        };
    }, [messages]);
}
```

```tsx
// web/src/App.tsx
import { useConversationLimits } from './hooks/useConversationLimits';

function App() {
    // ... existing state

    const conversationLimits = useConversationLimits(
        currentSession?.messages || []
    );

    const handleSendMessage = async (content: string) => {
        if (!currentSessionId) return;
        
        // Check if at limit
        if (conversationLimits.isAtLimit) {
            // Don't send - show error
            const errorMessage: Message = {
                id: crypto.randomUUID(),
                type: 'error',
                content: conversationLimits.blockedMessage!,
                timestamp: new Date().toISOString(),
            };
            addMessage(currentSessionId, errorMessage);
            return;
        }

        // ... rest of existing code
    };

    return (
        <div className="flex h-screen bg-white">
            <SessionList
                sessions={sessionsList}
                currentSessionId={currentSessionId}
                onSelectSession={handleSelectSession}
                onNewSession={createNewSession}
                onDeleteSession={handleDeleteSession}
            />

            <div className="flex-1 flex flex-col">
                {/* Header */}
                <div className="border-b bg-white px-6 py-4">
                    <h1 className="text-xl font-semibold text-gray-800">
                        {currentSession?.title || 'Pokemon RAG Chat'}
                    </h1>
                    <p className="text-sm text-gray-500 mt-1">
                        Ask questions about Pokemon
                    </p>
                    
                    {/* Conversation limit indicator */}
                    {currentSession && currentSession.messages.length > 0 && (
                        <div className="flex items-center gap-2 mt-2">
                            <div className="text-xs text-gray-400">
                                {conversationLimits.currentTurns} / {conversationLimits.maxTurns} turns
                            </div>
                            <div className="flex-1 bg-gray-200 rounded-full h-1.5">
                                <div 
                                    className={`h-1.5 rounded-full transition-all ${
                                        conversationLimits.isAtLimit 
                                            ? 'bg-red-500' 
                                            : conversationLimits.isNearLimit 
                                                ? 'bg-yellow-500' 
                                                : 'bg-blue-500'
                                    }`}
                                    style={{ 
                                        width: `${(conversationLimits.currentTurns / conversationLimits.maxTurns) * 100}%` 
                                    }}
                                />
                            </div>
                        </div>
                    )}
                </div>

                {/* Messages */}
                <div className="flex-1 overflow-y-auto p-6">
                    {/* Warning banner */}
                    {conversationLimits.warningMessage && (
                        <div className="mb-4 bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                            <div className="flex items-start gap-3">
                                <span className="text-2xl">⚠️</span>
                                <div className="flex-1">
                                    <p className="text-sm text-yellow-800">
                                        {conversationLimits.warningMessage}
                                    </p>
                                    <button
                                        onClick={createNewSession}
                                        className="mt-2 text-sm text-yellow-900 underline hover:text-yellow-700"
                                    >
                                        Start new chat now
                                    </button>
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Blocked banner */}
                    {conversationLimits.isAtLimit && (
                        <div className="mb-4 bg-red-50 border border-red-200 rounded-lg p-4">
                            <div className="flex items-start gap-3">
                                <span className="text-2xl">🚫</span>
                                <div className="flex-1">
                                    <p className="text-sm text-red-800 font-medium">
                                        {conversationLimits.blockedMessage}
                                    </p>
                                    <button
                                        onClick={createNewSession}
                                        className="mt-3 px-4 py-2 bg-red-600 text-white text-sm rounded-lg hover:bg-red-700 transition-colors"
                                    >
                                        Start New Chat
                                    </button>
                                </div>
                            </div>
                        </div>
                    )}

                    {currentSession && currentSession.messages.length === 0 ? (
                        <div className="flex items-center justify-center h-full">
                            <div className="text-center text-gray-500">
                                <div className="text-4xl mb-4">💬</div>
                                <p className="text-lg">Start a conversation about Pokemon!</p>
                                <p className="text-sm mt-2">Ask about stats, types, abilities, or anything else.</p>
                            </div>
                        </div>
                    ) : (
                        currentSession?.messages.map(message => (
                            <ChatMessage key={message.id} message={message} />
                        ))
                    )}
                    {isLoading && (
                        <div className="flex justify-start mb-4">
                            <div className="bg-gray-100 rounded-lg px-4 py-3">
                                <div className="flex gap-1">
                                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }}></div>
                                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }}></div>
                                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }}></div>
                                </div>
                            </div>
                        </div>
                    )}
                    <div ref={messagesEndRef} />
                </div>

                {/* Input */}
                <ChatInput
                    onSendMessage={handleSendMessage}
                    disabled={isLoading || !currentSessionId || conversationLimits.isAtLimit}
                    blockedMessage={conversationLimits.blockedMessage}
                />
            </div>
        </div>
    );
}

export default App;
```

```tsx
// web/src/components/ChatInput.tsx
import { useState, type FormEvent, type KeyboardEvent } from 'react';

interface ChatInputProps {
    onSendMessage: (message: string) => void;
    disabled?: boolean;
    blockedMessage?: string | null;
}

export function ChatInput({ onSendMessage, disabled = false, blockedMessage }: ChatInputProps) {
    const [message, setMessage] = useState('');

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();
        if (message.trim() && !disabled) {
            onSendMessage(message.trim());
            setMessage('');
        }
    };

    const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSubmit(e);
        }
    };

    return (
        <form onSubmit={handleSubmit} className="border-t bg-white p-4">
            {blockedMessage && (
                <div className="mb-3 text-sm text-red-600 bg-red-50 border border-red-200 rounded px-3 py-2">
                    {blockedMessage}
                </div>
            )}
            <div className="flex gap-2">
                <textarea
                    value={message}
                    onChange={(e) => setMessage(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={
                        blockedMessage 
                            ? "Please start a new chat to continue..." 
                            : "Ask about Pokemon..."
                    }
                    disabled={disabled}
                    className="flex-1 resize-none rounded-lg border border-gray-300 px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
                    rows={3}
                />
                <button
                    type="submit"
                    disabled={disabled || !message.trim()}
                    className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
                >
                    Send
                </button>
            </div>
            <div className="mt-2 text-xs text-gray-500">
                Press Enter to send, Shift+Enter for new line
            </div>
        </form>
    );
}
```

### 4. **Session List: Show Limit Status**

```tsx
// web/src/components/SessionList.tsx
import type { Session } from '@/types';
import { CONVERSATION_LIMITS } from '@/types';

interface SessionListProps {
    sessions: Session[];
    currentSessionId: string | null;
    onSelectSession: (sessionId: string) => void;
    onNewSession: () => void;
    onDeleteSession: (sessionId: string) => void;
}

export function SessionList({
    sessions,
    currentSessionId,
    onSelectSession,
    onNewSession,
    onDeleteSession,
}: SessionListProps) {
    const getSessionStatus = (session: Session) => {
        const turns = Math.floor(session.messages.length / 2);
        const isAtLimit = turns >= CONVERSATION_LIMITS.MAX_TURNS;
        const isNearLimit = turns >= CONVERSATION_LIMITS.WARNING_THRESHOLD;
        
        return { isAtLimit, isNearLimit, turns };
    };

    return (
        <div className="w-64 bg-gray-50 border-r flex flex-col">
            <div className="p-4 border-b bg-white">
                <button
                    onClick={onNewSession}
                    className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                >
                    + New Chat
                </button>
            </div>

            <div className="flex-1 overflow-y-auto p-2">
                {sessions.length === 0 ? (
                    <div className="text-center text-gray-500 text-sm mt-8">
                        No sessions yet
                    </div>
                ) : (
                    <div className="space-y-1">
                        {sessions.map((session) => {
                            const status = getSessionStatus(session);
                            
                            return (
                                <div
                                    key={session.id}
                                    className={`group relative rounded-lg p-3 cursor-pointer transition-colors ${
                                        currentSessionId === session.id
                                            ? 'bg-blue-100 border border-blue-300'
                                            : 'bg-white hover:bg-gray-100 border border-transparent'
                                    }`}
                                    onClick={() => onSelectSession(session.id)}
                                >
                                    <div className="flex items-start justify-between gap-2">
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2">
                                                <div className="font-medium text-sm truncate">
                                                    {session.title}
                                                </div>
                                                {status.isAtLimit && (
                                                    <span className="text-xs px-1.5 py-0.5 bg-red-100 text-red-700 rounded">
                                                        Full
                                                    </span>
                                                )}
                                                {status.isNearLimit && !status.isAtLimit && (
                                                    <span className="text-xs px-1.5 py-0.5 bg-yellow-100 text-yellow-700 rounded">
                                                        Near Limit
                                                    </span>
                                                )}
                                            </div>
                                            <div className="text-xs text-gray-500 mt-1">
                                                {new Date(session.lastActivity).toLocaleDateString()}
                                            </div>
                                            <div className="flex items-center gap-2 text-xs text-gray-400 mt-1">
                                                <span>{session.messages.length} messages</span>
                                                <span>•</span>
                                                <span>{status.turns}/{CONVERSATION_LIMITS.MAX_TURNS} turns</span>
                                            </div>
                                        </div>
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                onDeleteSession(session.id);
                                            }}
                                            className="opacity-0 group-hover:opacity-100 text-red-500 hover:text-red-700 transition-opacity"
                                            title="Delete session"
                                        >
                                            ✕
                                        </button>
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                )}
            </div>

            <div className="p-4 border-t bg-white text-xs text-gray-500 text-center">
                Pokemon RAG Chat
            </div>
        </div>
    );
}
```

## 📊 UX Flow

### Normal Conversation (Turns 1-11)
```
✅ Input enabled
✅ Normal experience
✅ Progress bar: blue
```

### Warning Phase (Turns 12-14)
```
⚠️ Yellow banner appears
⚠️ "3 turns remaining"
⚠️ Progress bar: yellow
⚠️ Suggest starting new chat
✅ Can still send messages
```

### Blocked Phase (Turn 15+)
```
🚫 Red banner appears
🚫 "Conversation limit reached"
🚫 Progress bar: red (100%)
🚫 Input disabled
🚫 Big "Start New Chat" button
❌ Cannot send more messages
```

## ✅ Benefits

1. ✅ **Clear UX** - User knows exactly what's happening
2. ✅ **No confusion** - Hard limit prevents edge cases
3. ✅ **Better context** - New chats = clean context for RAG
4. ✅ **Simple code** - No complex truncation logic
5. ✅ **Backend safety** - Rejects oversized requests
6. ✅ **Predictable** - Same limit for all users

## 🎯 Implementation Checklist

**Backend:**
- [ ] Add `ErrConversationTooLong` error
- [ ] Update validation with hard limits
- [ ] Special error response (413 status)
- [ ] Add to config.yaml

**Frontend:**
- [ ] Create `useConversationLimits` hook
- [ ] Add progress indicator to header
- [ ] Warning banner (yellow)
- [ ] Blocked banner (red)
- [ ] Disable input when at limit
- [ ] Update SessionList with status badges

**Testing:**
- [ ] Test at exactly 15 turns
- [ ] Test warning threshold
- [ ] Test backend rejection
- [ ] Test "Start New Chat" button