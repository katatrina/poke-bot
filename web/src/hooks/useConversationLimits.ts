import { useMemo } from 'react';
import { CONVERSATION_LIMITS } from '../types';
import type { Message } from '../types';

interface ConversationLimits {
    currentTurns: number;
    maxTurns: number;
    totalTokens: number;
    maxTokens: number;
    isNearLimit: boolean;
    isAtLimit: boolean;
    warningMessage: string | null;
    blockedMessage: string | null;
}

// Approximate token count (characters / 4 is a common heuristic)
function approximateTokenCount(text: string): number {
    return Math.ceil(text.length / 4);
}

/**
 * Tracks conversation limits for UI purposes.
 * Note: This tracks the FULL conversation for display and warnings,
 * but only the last MAX_HISTORY_TURNS are actually sent to the LLM (sliding window).
 */
export function useConversationLimits(messages: Message[]): ConversationLimits {
    return useMemo(() => {
        const turns = Math.floor(messages.length / 2);
        const totalTokens = messages.reduce((sum, msg) => sum + approximateTokenCount(msg.content), 0);

        const turnsRemaining = CONVERSATION_LIMITS.MAX_TURNS - turns;
        const isNearLimit = turns >= CONVERSATION_LIMITS.WARNING_THRESHOLD;
        const isAtLimit = turns >= CONVERSATION_LIMITS.MAX_TURNS ||
                          totalTokens >= CONVERSATION_LIMITS.MAX_TOTAL_TOKENS;

        let warningMessage: string | null = null;
        let blockedMessage: string | null = null;

        if (isAtLimit) {
            blockedMessage =
                "This conversation has reached the maximum length. Please start a new chat to continue.";
        } else if (isNearLimit) {
            warningMessage =
                `This conversation is approaching its limit (${turnsRemaining} ${turnsRemaining === 1 ? 'turn' : 'turns'} remaining). Consider starting a new chat soon.`;
        }

        return {
            currentTurns: turns,
            maxTurns: CONVERSATION_LIMITS.MAX_TURNS,
            totalTokens,
            maxTokens: CONVERSATION_LIMITS.MAX_TOTAL_TOKENS,
            isNearLimit,
            isAtLimit,
            warningMessage,
            blockedMessage,
        };
    }, [messages]);
}
