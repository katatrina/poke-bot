export interface Message {
    id: string;
    type: 'user' | 'assistant' | 'error';
    content: string;
    sources?: string[];
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
    sources: string[];
    context: string;
}

export interface ApiError {
    error: string;
    details?: string;
}