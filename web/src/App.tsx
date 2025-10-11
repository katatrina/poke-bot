// web/src/App.tsx - REFACTORED
import {useEffect, useRef, useState} from 'react';
import {ChatMessage} from './components/ChatMessage';
import {ChatInput} from './components/ChatInput';
import {SessionList} from './components/SessionList';
import {TypingIndicator} from './components/TypingIndicator';
import {useLocalStorage} from './hooks/useLocalStorage';
import {useConversationLimits} from './hooks/useConversationLimits';
import {apiService} from './services/api';
import type {ConversationMessage, Message, Session, SessionsState} from './types';
import {CONVERSATION_LIMITS} from './types';

function App() {
    const [sessions, setSessions] = useLocalStorage<SessionsState>('chat-sessions', {});
    const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const messagesContainerRef = useRef<HTMLDivElement>(null);

    const currentSession = currentSessionId ? sessions[currentSessionId] : null;
    const conversationLimits = useConversationLimits(currentSession?.messages || []);

    // Auto-scroll với smooth behavior
    useEffect(() => {
        if (messagesEndRef.current) {
            messagesEndRef.current.scrollIntoView({
                behavior: 'smooth',
                block: 'end'
            });
        }
    }, [currentSession?.messages, isLoading]);

    // Initialize first session
    useEffect(() => {
        if (Object.keys(sessions).length === 0) {
            createNewSession();
        } else if (!currentSessionId) {
            const sortedSessions = Object.values(sessions).sort(
                (a, b) => new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime()
            );
            if (sortedSessions.length > 0) {
                setCurrentSessionId(sortedSessions[0].id);
            }
        }
    }, []);

    const createNewSession = () => {
        const newSessionId = crypto.randomUUID();
        const newSession: Session = {
            id: newSessionId,
            title: 'New Chat',
            lastActivity: new Date().toISOString(),
            messages: [],
        };

        setSessions(prev => ({
            ...prev,
            [newSessionId]: newSession,
        }));
        setCurrentSessionId(newSessionId);
    };

    const updateSessionTitle = (sessionId: string, firstMessage: string) => {
        const title = firstMessage.slice(0, 50) + (firstMessage.length > 50 ? '...' : '');
        setSessions(prev => ({
            ...prev,
            [sessionId]: {
                ...prev[sessionId],
                title,
            },
        }));
    };

    const addMessage = (sessionId: string, message: Message) => {
        setSessions(prev => ({
            ...prev,
            [sessionId]: {
                ...prev[sessionId],
                messages: [...prev[sessionId].messages, message],
                lastActivity: new Date().toISOString(),
            },
        }));
    };

    const handleSendMessage = async (content: string) => {
        if (!currentSessionId) return;

        if (conversationLimits.isAtLimit) {
            const errorMessage: Message = {
                id: crypto.randomUUID(),
                type: 'error',
                content: conversationLimits.blockedMessage!,
                timestamp: new Date().toISOString(),
            };
            addMessage(currentSessionId, errorMessage);
            return;
        }

        // ✅ CREATE user message
        const userMessage: Message = {
            id: crypto.randomUUID(),
            type: 'user',
            content,
            timestamp: new Date().toISOString(),
        };

        // ✅ ADD user message FIRST
        addMessage(currentSessionId, userMessage);

        // Update title if first message
        if (currentSession && currentSession.messages.length === 0) {
            updateSessionTitle(currentSessionId, content);
        }

        // ✅ SET loading AFTER adding message
        setIsLoading(true);

        try {
            // ✅ BUILD conversation history WITHOUT relying on updated state
            // Get current messages và manually add userMessage
            const existingMessages = currentSession?.messages || [];
            const allMessagesForHistory = [...existingMessages, userMessage];

            const maxHistoryMessages = CONVERSATION_LIMITS.MAX_HISTORY_TURNS * 2;
            const recentMessages = allMessagesForHistory.slice(-maxHistoryMessages);

            const conversationHistory: ConversationMessage[] = recentMessages
                .filter(msg => msg.type !== 'error')
                .map(msg => ({
                    type: msg.type === 'user' ? 'user' : 'assistant',
                    content: msg.content,
                }));

            const response = await apiService.chat({
                message: content,
                conversation_history: conversationHistory,
            });

            const assistantMessage: Message = {
                id: crypto.randomUUID(),
                type: 'assistant',
                content: response.response,
                timestamp: new Date().toISOString(),
            };

            addMessage(currentSessionId, assistantMessage);
        } catch (error) {
            const errorMessage: Message = {
                id: crypto.randomUUID(),
                type: 'error',
                content: error instanceof Error ? error.message : 'Failed to send message',
                timestamp: new Date().toISOString(),
            };

            addMessage(currentSessionId, errorMessage);
        } finally {
            setIsLoading(false);
        }
    };

    const sessionsList = Object.values(sessions).sort(
        (a, b) => new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime()
    );

    return (
        <div className="flex h-screen bg-gray-100">
            <SessionList
                sessions={sessionsList}
                currentSessionId={currentSessionId}
                onSelectSession={setCurrentSessionId}
                onNewSession={createNewSession}
                onDeleteSession={(sessionId) => {
                    setSessions(prev => {
                        const newSessions = {...prev};
                        delete newSessions[sessionId];
                        return newSessions;
                    });

                    if (currentSessionId === sessionId) {
                        const remainingSessions = Object.keys(sessions).filter(id => id !== sessionId);
                        setCurrentSessionId(remainingSessions.length > 0 ? remainingSessions[0] : null);
                    }
                }}
            />

            <main className="flex-1 flex flex-col lg:ml-0 ml-0">
                {/* Header */}
                <header className="bg-white border-b border-gray-200 px-6 py-4 shadow-sm">
                    <div className="max-w-4xl mx-auto">
                        <h1 className="text-xl font-bold text-gray-900">
                            {currentSession?.title || 'Pokemon RAG Chat'}
                        </h1>
                        <p className="text-sm text-gray-600 mt-0.5">
                            Ask me anything about Pokemon!
                        </p>

                        {/* Progress Bar - CHỈ HIỆN KHI GẦN LIMIT */}
                        {currentSession &&
                            currentSession.messages.length > 0 &&
                            conversationLimits.isNearLimit && ( // ← Thêm điều kiện này
                                <div className="flex items-center gap-3 mt-3">
        <span className="text-xs text-gray-500 font-medium">
            {conversationLimits.currentTurns}/{conversationLimits.maxTurns}
        </span>
                                    <div className="flex-1 bg-gray-200 rounded-full h-2 overflow-hidden">
                                        <div
                                            className={`h-full transition-all duration-500 ease-out ${
                                                conversationLimits.isAtLimit
                                                    ? 'bg-red-500'
                                                    : 'bg-yellow-500'
                                            }`}
                                            style={{
                                                width: `${Math.min((conversationLimits.currentTurns / conversationLimits.maxTurns) * 100, 100)}%`
                                            }}
                                        />
                                    </div>
                                </div>
                            )}
                    </div>
                </header>

                {/* Messages Container */}
                <div
                    ref={messagesContainerRef}
                    className="flex-1 overflow-y-auto px-6 py-6 scrollbar-thin"
                >
                    <div className="max-w-4xl mx-auto">
                        {/* Warning Banner */}
                        {conversationLimits.warningMessage && !conversationLimits.isAtLimit && (
                            <div
                                className="mb-6 bg-yellow-50 border-l-4 border-yellow-400 rounded-r-lg p-4 shadow-sm animate-in slide-in-from-top-4">
                                <div className="flex items-start gap-3">
                                    <div className="text-2xl">⚠️</div>
                                    <div className="flex-1">
                                        <h3 className="text-sm font-semibold text-yellow-800 mb-1">
                                            Approaching Limit
                                        </h3>
                                        <p className="text-sm text-yellow-700">
                                            {conversationLimits.warningMessage}
                                        </p>
                                        <button
                                            onClick={createNewSession}
                                            className="mt-2 text-sm text-yellow-900 underline hover:text-yellow-700 font-medium"
                                        >
                                            Start new chat now →
                                        </button>
                                    </div>
                                </div>
                            </div>
                        )}

                        {/* Blocked Banner */}
                        {conversationLimits.isAtLimit && (
                            <div
                                className="mb-6 bg-red-50 border-l-4 border-red-500 rounded-r-lg p-4 shadow-sm animate-in slide-in-from-top-4">
                                <div className="flex items-start gap-3">
                                    <div className="text-2xl">🚫</div>
                                    <div className="flex-1">
                                        <h3 className="text-sm font-semibold text-red-800 mb-1">
                                            Conversation Limit Reached
                                        </h3>
                                        <p className="text-sm text-red-700 mb-3">
                                            {conversationLimits.blockedMessage}
                                        </p>
                                        <button
                                            onClick={createNewSession}
                                            className="px-4 py-2 bg-red-600 text-white text-sm font-medium rounded-lg hover:bg-red-700 transition-colors shadow-sm"
                                        >
                                            Start New Chat
                                        </button>
                                    </div>
                                </div>
                            </div>
                        )}

                        {/* Render messages hoặc empty state */}
                        {!currentSession || currentSession.messages.length === 0 ? (
                            // Empty State - chỉ hiện khi THỰC SỰ không có messages
                            !isLoading && (
                                <div className="flex items-center justify-center h-full min-h-[400px]">
                                    <div className="text-center max-w-md">
                                        <div className="text-6xl mb-6 animate-bounce">💬</div>
                                        <h2 className="text-2xl font-bold text-gray-900 mb-3">
                                            Start a Conversation
                                        </h2>
                                        <p className="text-gray-600 mb-6">
                                            Ask me anything about Pokemon - stats, types, abilities, evolutions, and
                                            more!
                                        </p>
                                        <div className="grid grid-cols-1 gap-2">
                                            {[
                                                'What type is Charizard?',
                                                'Compare Pikachu and Raichu stats',
                                                'What is Dragonite weak against?'
                                            ].map((example) => (
                                                <button
                                                    key={example}
                                                    onClick={() => handleSendMessage(example)}
                                                    disabled={isLoading}
                                                    className="px-4 py-2 text-sm bg-white border border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors text-left disabled:opacity-50 disabled:cursor-not-allowed"
                                                >
                                                    {example}
                                                </button>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            )
                        ) : (
                            // Messages List
                            <>
                                {currentSession.messages.map((message) => (
                                    <ChatMessage key={message.id} message={message}/>
                                ))}
                                {isLoading && <TypingIndicator/>}
                            </>
                        )}
                        <div ref={messagesEndRef}/>
                    </div>
                </div>

                {/* Input Area */}
                <div className="bg-white border-t border-gray-200 px-6 py-4 shadow-lg">
                    <div className="max-w-4xl mx-auto">
                        <ChatInput
                            onSendMessage={handleSendMessage}
                            disabled={isLoading || !currentSessionId || conversationLimits.isAtLimit}
                            blockedMessage={conversationLimits.blockedMessage}
                        />
                    </div>
                </div>
            </main>
        </div>
    );
}

export default App;