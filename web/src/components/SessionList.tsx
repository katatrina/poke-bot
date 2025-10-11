// web/src/components/SessionList.tsx - REFACTORED với mobile support
import { useState } from 'react';
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
    const [isOpen, setIsOpen] = useState(false);

    const getSessionStatus = (session: Session) => {
        const turns = Math.floor(session.messages.length / 2);
        const isAtLimit = turns >= CONVERSATION_LIMITS.MAX_TURNS;
        const isNearLimit = turns >= CONVERSATION_LIMITS.WARNING_THRESHOLD;
        return { isAtLimit, isNearLimit, turns };
    };

    return (
        <>
            {/* Mobile Toggle Button */}
            <button
                onClick={() => setIsOpen(!isOpen)}
                className="lg:hidden fixed top-4 left-4 z-50 p-2 bg-white rounded-lg shadow-lg border border-gray-200"
                aria-label="Toggle sidebar"
            >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                </svg>
            </button>

            {/* Overlay cho mobile */}
            {isOpen && (
                <div
                    className="lg:hidden fixed inset-0 bg-black/50 z-40"
                    onClick={() => setIsOpen(false)}
                />
            )}

            {/* Sidebar */}
            <aside className={`
                fixed lg:static inset-y-0 left-0 z-40
                w-80 bg-gray-50 border-r border-gray-200 flex flex-col
                transform transition-transform duration-300 ease-in-out
                ${isOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
            `}>
                {/* Header */}
                <div className="p-4 border-b border-gray-200 bg-white">
                    <div className="flex items-center justify-between mb-3">
                        <h2 className="text-lg font-semibold text-gray-900">Conversations</h2>
                        <button
                            onClick={() => setIsOpen(false)}
                            className="lg:hidden p-1 hover:bg-gray-100 rounded"
                        >
                            ✕
                        </button>
                    </div>
                    <button
                        onClick={() => {
                            onNewSession();
                            setIsOpen(false);
                        }}
                        className="w-full px-4 py-2.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center justify-center gap-2 shadow-sm"
                    >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                        </svg>
                        New Chat
                    </button>
                </div>

                {/* Sessions List */}
                <div className="flex-1 overflow-y-auto p-3 scrollbar-thin">
                    {sessions.length === 0 ? (
                        <div className="text-center py-12 px-4">
                            <div className="text-4xl mb-3">💬</div>
                            <p className="text-sm text-gray-500">No conversations yet</p>
                            <p className="text-xs text-gray-400 mt-1">Start a new chat to begin!</p>
                        </div>
                    ) : (
                        <div className="space-y-2">
                            {sessions.map((session) => {
                                const status = getSessionStatus(session);
                                const isActive = currentSessionId === session.id;

                                return (
                                    <div
                                        key={session.id}
                                        onClick={() => {
                                            onSelectSession(session.id);
                                            setIsOpen(false);
                                        }}
                                        className={`
                                            group relative w-full text-left rounded-lg p-3 transition-all cursor-pointer
                                            ${isActive
                                            ? 'bg-blue-50 border-2 border-blue-500 shadow-sm'
                                            : 'bg-white hover:bg-gray-50 border border-gray-200 hover:border-gray-300'
                                        }
                                        `}
                                    >
                                        <div className="flex items-start justify-between gap-2">
                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center gap-2 mb-1">
                                                    <h3 className="font-medium text-sm truncate text-gray-900">
                                                        {session.title}
                                                    </h3>
                                                    {status.isAtLimit && (
                                                        <span className="flex-shrink-0 text-xs px-2 py-0.5 bg-red-100 text-red-700 rounded-full font-medium">
                                                            Full
                                                        </span>
                                                    )}
                                                    {status.isNearLimit && !status.isAtLimit && (
                                                        <span className="flex-shrink-0 text-xs px-2 py-0.5 bg-yellow-100 text-yellow-700 rounded-full font-medium">
                                                            Near Limit
                                                        </span>
                                                    )}
                                                </div>
                                                <p className="text-xs text-gray-500">
                                                    {new Date(session.lastActivity).toLocaleDateString('en-US', {
                                                        month: 'short',
                                                        day: 'numeric',
                                                        hour: 'numeric',
                                                        minute: '2-digit'
                                                    })}
                                                </p>
                                                <div className="flex items-center gap-2 mt-1.5">
                                                    <span className="text-xs text-gray-400">
                                                        {session.messages.length} messages
                                                    </span>
                                                    <span className="text-xs text-gray-300">•</span>
                                                    <span className={`text-xs ${
                                                        status.isAtLimit ? 'text-red-600' :
                                                            status.isNearLimit ? 'text-yellow-600' :
                                                                'text-gray-400'
                                                    }`}>
                                                        {status.turns}/{CONVERSATION_LIMITS.MAX_TURNS} turns
                                                    </span>
                                                </div>
                                            </div>
                                            <button
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    onDeleteSession(session.id);
                                                }}
                                                className="opacity-0 group-hover:opacity-100 flex-shrink-0 p-1 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-all"
                                                title="Delete conversation"
                                            >
                                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                                </svg>
                                            </button>
                                        </div>
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="p-4 border-t border-gray-200 bg-white">
                    <div className="flex items-center gap-2 text-xs text-gray-500">
                        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                            <path d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-11a1 1 0 10-2 0v2H7a1 1 0 100 2h2v2a1 1 0 102 0v-2h2a1 1 0 100-2h-2V7z" />
                        </svg>
                        <span>Pokemon RAG Chat</span>
                    </div>
                </div>
            </aside>
        </>
    );
}