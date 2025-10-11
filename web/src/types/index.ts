export interface Message {
    id: string;
    type: 'user' | 'assistant' | 'error';
    content: string;
    timestamp: string;
}

export interface Session {
    id: string;
    title: string;
    lastActivity: string;
    messages: Message[];
}

export interface SessionsState {
    [sessionId: string]: Session;
}

export interface ConversationMessage {
    type: 'user' | 'assistant';
    content: string;
}

export interface ChatRequest {
    message: string;
    conversation_history: ConversationMessage[];
}

export interface ChatResponse {
    response: string;
    context: string;
}

export interface ApiError {
    error: string;
    details?: string;
}

export const CONVERSATION_LIMITS = {
    MAX_TURNS: 15,              // 30 messages total before forcing new chat
    MAX_TOTAL_TOKENS: 2500,     // Max tokens (using cl100k_base encoding)
    WARNING_THRESHOLD: 12,      // Warn at 12 turns (24 messages)
    MAX_HISTORY_TURNS: 5,       // Send only last 5 turns (10 messages) to LLM
} as const;