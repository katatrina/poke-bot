import type {ChatRequest, ChatResponse, ApiError} from '@/types';

const API_BASE_URL = '/api/v1';

class ApiService {
    async chat(request: ChatRequest): Promise<ChatResponse> {
        try {
            const response = await fetch(`${API_BASE_URL}/chat`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(request),
            });

            if (!response.ok) {
                const errorData: ApiError = await response.json();
                throw new Error(errorData.error || 'Failed to send message');
            }

            return await response.json();
        } catch (error) {
            if (error instanceof Error) {
                throw error;
            }
            throw new Error('Network error occurred');
        }
    }

    async ingestPokemon(source: string = 'pokemondb', crawlLimit: number = 10): Promise<void> {
        try {
            const response = await fetch(`${API_BASE_URL}/ingest`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    source,
                    crawl_limit: crawlLimit,
                }),
            });

            if (!response.ok) {
                const errorData: ApiError = await response.json();
                throw new Error(errorData.error || 'Failed to ingest data');
            }
        } catch (error) {
            if (error instanceof Error) {
                throw error;
            }
            throw new Error('Network error occurred');
        }
    }

    async healthCheck(): Promise<boolean> {
        try {
            const response = await fetch(`${API_BASE_URL}/health`);
            return response.ok;
        } catch {
            return false;
        }
    }
}

export const apiService = new ApiService();