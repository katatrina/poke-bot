// web/src/styles/theme.ts - NEW FILE
export const theme = {
    colors: {
        primary: {
            50: '#eff6ff',
            100: '#dbeafe',
            500: '#3b82f6',
            600: '#2563eb',
            700: '#1d4ed8',
        },
        success: '#10b981',
        warning: '#f59e0b',
        error: '#ef4444',
    },
    animations: {
        fadeIn: 'animate-in fade-in duration-300',
        slideUp: 'animate-in slide-in-from-bottom-4 duration-300',
        slideDown: 'animate-in slide-in-from-top-4 duration-300',
    }
} as const;