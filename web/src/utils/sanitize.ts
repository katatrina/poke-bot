import DOMPurify from 'dompurify';

/**
 * Sanitize user input to prevent XSS attacks
 * Uses DOMPurify to clean potentially malicious content
 */
export function sanitizeInput(input: string): string {
    if (!input) return '';

    // Configure DOMPurify for text-only output
    const config: DOMPurify.Config = {
        ALLOWED_TAGS: [], // No HTML tags allowed
        ALLOWED_ATTR: [], // No attributes allowed
        KEEP_CONTENT: true, // Keep text content
    };

    // Sanitize the input
    const sanitized = DOMPurify.sanitize(input, config);

    // Additional cleaning: trim whitespace
    return sanitized.trim();
}

/**
 * Validate message length
 */
export function validateMessageLength(message: string, maxLength: number = 1000): {
    valid: boolean;
    error?: string;
} {
    const trimmed = message.trim();

    if (trimmed.length === 0) {
        return {
            valid: false,
            error: 'Message cannot be empty',
        };
    }

    if (trimmed.length > maxLength) {
        return {
            valid: false,
            error: `Message is too long (max ${maxLength} characters)`,
        };
    }

    return { valid: true };
}

/**
 * Detect potential prompt injection patterns
 */
export function detectPromptInjection(input: string): boolean {
    const lowerInput = input.toLowerCase();

    const suspiciousPatterns = [
        /ignore\s+(previous|above|all|prior)\s+(instructions?|prompts?|rules?)/i,
        /disregard\s+(previous|above|all|prior)\s+(instructions?|prompts?|rules?)/i,
        /forget\s+(previous|above|all|prior)\s+(instructions?|prompts?|rules?)/i,
        /you\s+are\s+(now|actually)\s+a/i,
        /new\s+instructions?:/i,
        /system\s*:\s*/i,
        /override\s+(previous|above|all|prior)/i,
        /act\s+as\s+if\s+you\s+are/i,
    ];

    return suspiciousPatterns.some(pattern => pattern.test(lowerInput));
}

/**
 * Comprehensive input validation and sanitization
 * Returns sanitized input or null if invalid
 */
export function validateAndSanitize(input: string): {
    sanitized: string | null;
    error?: string;
} {
    // First trim
    const trimmed = input.trim();

    // Check length
    const lengthValidation = validateMessageLength(trimmed);
    if (!lengthValidation.valid) {
        return {
            sanitized: null,
            error: lengthValidation.error,
        };
    }

    // Check for prompt injection
    if (detectPromptInjection(trimmed)) {
        return {
            sanitized: null,
            error: 'Your message contains patterns that are not allowed. Please rephrase your question.',
        };
    }

    // Sanitize
    const sanitized = sanitizeInput(trimmed);

    if (!sanitized) {
        return {
            sanitized: null,
            error: 'Message cannot be empty after sanitization',
        };
    }

    return { sanitized };
}
