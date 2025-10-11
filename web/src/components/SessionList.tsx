import type {Session} from '@/types';
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