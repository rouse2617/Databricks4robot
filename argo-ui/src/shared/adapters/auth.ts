const TOKEN_KEY = 'cyber-databrew-token';

export const CyberAuth = {
    getToken(): string | null {
        try {
            return localStorage.getItem(TOKEN_KEY);
        } catch {
            return null;
        }
    },

    setToken(token: string): void {
        try {
            localStorage.setItem(TOKEN_KEY, token);
        } catch {
            // localStorage may be unavailable
        }
    },

    clearToken(): void {
        try {
            localStorage.removeItem(TOKEN_KEY);
        } catch {
            // localStorage may be unavailable
        }
    },

    isAuthenticated(): boolean {
        return !!this.getToken();
    }
};
