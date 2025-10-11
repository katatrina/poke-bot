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
    const [shouldAutoScroll, setShouldAutoScroll] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const messagesContainerRef = useRef<HTMLDivElement>(null);
    const previousMessageCountRef = useRef(0);

    const currentSession = currentSessionId ? sessions[currentSessionId] : null;
    const conversationLimits = useConversationLimits(currentSession?.messages || []);

    // Auto-scroll chỉ khi có tin nhắn mới
    useEffect(() => {
        const currentMessageCount = currentSession?.messages.length || 0;
        
        if (shouldAutoScroll && messagesEndRef.current && currentMessageCount > previousMessageCountRef.current) {
            messagesEndRef.current.scrollIntoView({
                behavior: 'smooth',
                block: 'end'
            });
        }
        
        previousMessageCountRef.current = currentMessageCount;
    }, [currentSession?.messages, isLoading, shouldAutoScroll]);

    // Reset auto-scroll flag khi chuyển session
    useEffect(() => {
        setShouldAutoScroll(false);
        previousMessageCountRef.current = currentSession?.messages.length || 0;
    }, [currentSessionId]);

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
                messages: [...(prev[sessionId]?.messages || []), message],
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

        // Enable auto-scroll cho tin nhắn mới
        setShouldAutoScroll(true);

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
                {/* Messages Container */}
                <div
                    ref={messagesContainerRef}
                    className="flex-1 overflow-y-auto px-6 pt-8 pb-6 scrollbar-thin"
                >
                    <div className="max-w-3xl mx-auto">

                        {/* Empty State */}
                        {(!currentSession || currentSession.messages.length === 0) && !isLoading && (
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
                        )}

                        {/* Messages List */}
                        {currentSession && currentSession.messages.length > 0 && (
                            <>
                                {currentSession.messages.map((message) => (
                                    <ChatMessage key={message.id} message={message}/>
                                ))}
                            </>
                        )}

                        {/* Loading Indicator */}
                        {isLoading && <TypingIndicator/>}
                        <div ref={messagesEndRef}/>
                    </div>
                </div>

                {/* Input Area */}
                <div className="bg-gray-100 px-6 py-4">
                    <div className="max-w-3xl mx-auto">
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