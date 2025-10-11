import { useState, useEffect, useRef } from 'react';
import { ChatMessage } from './components/ChatMessage';
import { ChatInput } from './components/ChatInput';
import { SessionList } from './components/SessionList';
import { useLocalStorage } from './hooks/useLocalStorage';
import { apiService } from './services/api';
import type { Message, Session, SessionsState, ConversationMessage } from './types';

function App() {
    const [sessions, setSessions] = useLocalStorage<SessionsState>('chat-sessions', {});
    const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    const currentSession = currentSessionId ? sessions[currentSessionId] : null;

    useEffect(() => {
        // Auto-scroll to bottom when new messages arrive
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [currentSession?.messages]);

    useEffect(() => {
        // Create initial session if none exists
        if (Object.keys(sessions).length === 0) {
            createNewSession();
        } else if (!currentSessionId) {
            // Select the most recent session
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

        const userMessage: Message = {
            id: crypto.randomUUID(),
            type: 'user',
            content,
            timestamp: new Date().toISOString(),
        };

        addMessage(currentSessionId, userMessage);

        // Update session title with first message
        if (currentSession && currentSession.messages.length === 0) {
            updateSessionTitle(currentSessionId, content);
        }

        setIsLoading(true);

        try {
            // Build conversation history
            const conversationHistory: ConversationMessage[] = currentSession!.messages.map(msg => ({
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
                sources: response.sources,
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

    const handleSelectSession = (sessionId: string) => {
        setCurrentSessionId(sessionId);
    };

    const handleDeleteSession = (sessionId: string) => {
        setSessions(prev => {
            const newSessions = { ...prev };
            delete newSessions[sessionId];
            return newSessions;
        });

        if (currentSessionId === sessionId) {
            const remainingSessions = Object.keys(sessions).filter(id => id !== sessionId);
            setCurrentSessionId(remainingSessions.length > 0 ? remainingSessions[0] : null);
        }
    };

    const sessionsList = Object.values(sessions).sort(
        (a, b) => new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime()
    );

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
                </div>

                {/* Messages */}
                <div className="flex-1 overflow-y-auto p-6">
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
                    disabled={isLoading || !currentSessionId}
                />
            </div>
        </div>
    );
}

export default App;