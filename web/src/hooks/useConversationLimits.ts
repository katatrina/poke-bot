import { useMemo } from 'react';
import { CONVERSATION_LIMITS } from '../types';
import type { Message } from '../types';

interface ConversationLimits {
    currentTurns: number;
    maxTurns: number;
    totalChars: number;
    maxChars: number;
    isNearLimit: boolean;
    isAtLimit: boolean;
    warningMessage: string | null;
    blockedMessage: string | null;
}

export function useConversationLimits(messages: Message[]): ConversationLimits {
    return useMemo(() => {
        const turns = Math.floor(messages.length / 2);
        const totalChars = messages.reduce((sum, msg) => sum + msg.content.length, 0);

        const turnsRemaining = CONVERSATION_LIMITS.MAX_TURNS - turns;
        const isNearLimit = turns >= CONVERSATION_LIMITS.WARNING_THRESHOLD;
        const isAtLimit = turns >= CONVERSATION_LIMITS.MAX_TURNS ||
                          totalChars >= CONVERSATION_LIMITS.MAX_TOTAL_CHARS;

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
            totalChars,
            maxChars: CONVERSATION_LIMITS.MAX_TOTAL_CHARS,
            isNearLimit,
            isAtLimit,
            warningMessage,
            blockedMessage,
        };
    }, [messages]);
}
