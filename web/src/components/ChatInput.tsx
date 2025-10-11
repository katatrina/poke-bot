import { useState, type FormEvent, type KeyboardEvent } from 'react';
import { validateAndSanitize } from '../utils/sanitize';

interface ChatInputProps {
    onSendMessage: (message: string) => void;
    disabled?: boolean;
    blockedMessage?: string | null;
}

export function ChatInput({ onSendMessage, disabled = false, blockedMessage }: ChatInputProps) {
    const [message, setMessage] = useState('');
    const [error, setError] = useState<string | null>(null);

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();

        if (!message.trim() || disabled) {
            return;
        }

        // Validate and sanitize input
        const result = validateAndSanitize(message);

        if (!result.sanitized) {
            setError(result.error || 'Invalid input');
            return;
        }

        // Clear error and send sanitized message
        setError(null);
        onSendMessage(result.sanitized);
        setMessage('');
    };

    const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSubmit(e);
        }
    };

    const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
        setMessage(e.target.value);
        // Clear error when user types
        if (error) {
            setError(null);
        }
    };

    return (
        <form onSubmit={handleSubmit} className="border-t bg-white p-4">
            {(blockedMessage || error) && (
                <div className="mb-3 text-sm text-red-600 bg-red-50 border border-red-200 rounded px-3 py-2">
                    {blockedMessage || error}
                </div>
            )}
            <div className="flex gap-2">
                <textarea
                    value={message}
                    onChange={handleChange}
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