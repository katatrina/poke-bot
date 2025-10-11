import type { Message } from '@/types';

interface ChatMessageProps {
    message: Message;
}

export function ChatMessage({ message }: ChatMessageProps) {
    const isUser = message.type === 'user';
    const isError = message.type === 'error';

    return (
        <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-4`}>
            <div className={`max-w-[80%] rounded-lg px-4 py-3 ${
                isError
                    ? 'bg-red-100 text-red-800 border border-red-300'
                    : isUser
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-100 text-gray-900'
            }`}>
                <div className="text-sm whitespace-pre-wrap break-words">
                    {message.content}
                </div>

                <div className={`text-xs mt-1 ${
                    isError ? 'text-red-600' : isUser ? 'text-blue-200' : 'text-gray-500'
                }`}>
                    {new Date(message.timestamp).toLocaleTimeString()}
                </div>
            </div>
        </div>
    );
}