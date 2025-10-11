// web/src/components/ChatInput.tsx - REFACTORED
import { useState, useRef, useEffect, type FormEvent, type KeyboardEvent } from 'react';
import { validateAndSanitize } from '../utils/sanitize';

interface ChatInputProps {
    onSendMessage: (message: string) => void;
    disabled?: boolean;
    blockedMessage?: string | null;
}

export function ChatInput({ onSendMessage, disabled = false, blockedMessage }: ChatInputProps) {
    const [message, setMessage] = useState('');
    const [error, setError] = useState<string | null>(null);
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    // Auto-resize textarea
    useEffect(() => {
        if (textareaRef.current) {
            textareaRef.current.style.height = 'auto';
            textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 200)}px`;
        }
    }, [message]);

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();

        if (!message.trim() || disabled) {
            return;
        }

        const result = validateAndSanitize(message);

        if (!result.sanitized) {
            setError(result.error || 'Invalid input');
            return;
        }

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
        if (error) {
            setError(null);
        }
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-3">
            {(blockedMessage || error) && (
                <div className="flex items-start gap-2 px-3 py-2 bg-red-50 border border-red-200 rounded-lg text-sm text-red-800 animate-in slide-in-from-top-2">
                    <svg className="w-5 h-5 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                    </svg>
                    <span>{blockedMessage || error}</span>
                </div>
            )}

            <div className="flex gap-2">
                <div className="flex-1 relative">
                    <textarea
                        ref={textareaRef}
                        value={message}
                        onChange={handleChange}
                        onKeyDown={handleKeyDown}
                        placeholder={
                            blockedMessage
                                ? "Start a new chat to continue..."
                                : "Ask about Pokemon... (Shift+Enter for new line)"
                        }
                        disabled={disabled}
                        rows={1}
                        className={`
                            w-full resize-none rounded-xl px-4 py-3
                            border-2 transition-all
                            focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
                            disabled:bg-gray-100 disabled:cursor-not-allowed disabled:text-gray-500
                            ${error || blockedMessage ? 'border-red-300' : 'border-gray-200 hover:border-gray-300'}
                        `}
                        style={{ minHeight: '52px', maxHeight: '200px' }}
                    />
                </div>
                <button
                    type="submit"
                    disabled={disabled || !message.trim()}
                    className={`
                        px-6 py-3 rounded-xl font-medium transition-all shadow-sm
                        flex items-center gap-2
                        ${disabled || !message.trim()
                        ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                        : 'bg-blue-600 text-white hover:bg-blue-700 hover:shadow-md active:scale-95'
                    }
                    `}
                >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                    </svg>
                    Send
                </button>
            </div>

            <div className="flex items-center justify-between text-xs text-gray-500">
                <span>
                    Press <kbd className="px-1.5 py-0.5 bg-gray-100 rounded border border-gray-300 font-mono">Enter</kbd> to send
                </span>
                <span className={message.length > 800 ? 'text-yellow-600 font-medium' : ''}>
                    {message.length}/1000
                </span>
            </div>
        </form>
    );
}