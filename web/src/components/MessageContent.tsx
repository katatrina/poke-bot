// web/src/components/MessageContent.tsx - NEW FILE
import { memo } from 'react';

interface MessageContentProps {
    content: string;
    isUser: boolean;
    isError: boolean;
}

export const MessageContent = memo(({ content, isUser, isError }: MessageContentProps) => {
    // Simple markdown-like parsing for code blocks and formatting
    const renderContent = () => {
        // Detect code blocks
        const codeBlockRegex = /```(\w+)?\n([\s\S]*?)```/g;
        const parts: React.ReactNode[] = [];
        let lastIndex = 0;
        let match;

        while ((match = codeBlockRegex.exec(content)) !== null) {
            // Add text before code block
            if (match.index > lastIndex) {
                parts.push(
                    <span key={`text-${lastIndex}`}>
                        {content.substring(lastIndex, match.index)}
                    </span>
                );
            }

            // Add code block
            const language = match[1] || 'text';
            const code = match[2];
            parts.push(
                <div
                    key={`code-${match.index}`}
                    className={`
                        mt-2 mb-2 rounded-lg overflow-hidden
                        ${isUser ? 'bg-blue-700' : 'bg-gray-100'}
                    `}
                >
                    <div className={`
                        px-3 py-1 text-xs font-mono
                        ${isUser ? 'bg-blue-800 text-blue-200' : 'bg-gray-200 text-gray-600'}
                    `}>
                        {language}
                    </div>
                    <pre className={`
                        p-3 overflow-x-auto text-sm font-mono
                        ${isUser ? 'text-blue-50' : 'text-gray-800'}
                    `}>
                        <code>{code}</code>
                    </pre>
                </div>
            );

            lastIndex = match.index + match[0].length;
        }

        // Add remaining text
        if (lastIndex < content.length) {
            parts.push(
                <span key={`text-${lastIndex}`}>
                    {content.substring(lastIndex)}
                </span>
            );
        }

        return parts.length > 0 ? parts : content;
    };

    return (
        <div className={`
            text-sm leading-relaxed whitespace-pre-wrap break-words
            ${isError ? 'text-red-800 font-medium' : ''}
        `}>
            {renderContent()}
        </div>
    );
});

MessageContent.displayName = 'MessageContent';