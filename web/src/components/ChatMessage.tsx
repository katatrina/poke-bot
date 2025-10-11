// web/src/components/ChatMessage.tsx - REFACTORED
import { memo } from 'react';
import type { Message } from '@/types';
import { MessageContent } from './MessageContent';

interface ChatMessageProps {
    message: Message;
}

export const ChatMessage = memo(({ message }: ChatMessageProps) => {
    const isUser = message.type === 'user';
    const isError = message.type === 'error';

    return (
        <div
            className={`
                group flex gap-3 mb-6
                ${isUser ? 'flex-row-reverse' : 'flex-row'}
                animate-in slide-in-from-bottom-2 duration-300
            `}
        >
            {/* Avatar */}
            <div className={`
                flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium
                ${isError
                ? 'bg-red-100 text-red-600'
                : isUser
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-200 text-gray-700'
            }
            `}>
                {isError ? '⚠️' : isUser ? 'U' : '🤖'}
            </div>

            {/* Message Content */}
            <div className="flex-1 max-w-[80%]">
                <div className={`
                    rounded-2xl px-4 py-3 shadow-sm
                    ${isError
                    ? 'bg-red-50 border border-red-200'
                    : isUser
                        ? 'bg-blue-600 text-white'
                        : 'bg-white border border-gray-200'
                }
                `}>
                    <MessageContent
                        content={message.content}
                        isUser={isUser}
                        isError={isError}
                    />
                </div>

                {/* Timestamp */}
                <div className={`
                    text-xs mt-1 px-1
                    ${isUser ? 'text-right' : 'text-left'}
                    text-gray-500
                `}>
                    {new Date(message.timestamp).toLocaleTimeString('en-US', {
                        hour: 'numeric',
                        minute: '2-digit',
                    })}
                </div>
            </div>
        </div>
    );
});

ChatMessage.displayName = 'ChatMessage';