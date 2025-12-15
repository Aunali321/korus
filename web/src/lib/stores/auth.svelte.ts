import type { User } from '$lib/types';
import { api, setTokens, clearTokens } from '$lib/api';
import { favorites } from './favorites.svelte';
import { library } from './library.svelte';
import { settings } from './settings.svelte';

let initPromise: Promise<void> | null = null;

function createAuthStore() {
    let user = $state<User | null>(null);
    let isAuthenticated = $state(false);
    let isLoading = $state(true);

    async function init() {
        if (initPromise) return initPromise;

        initPromise = (async () => {
            if (typeof localStorage === 'undefined') {
                isLoading = false;
                return;
            }

            const token = localStorage.getItem('korus_access_token');
            if (!token) {
                isLoading = false;
                return;
            }

            try {
                user = await api.me();
                isAuthenticated = true;
                settings.load();
            } catch {
                clearTokens();
            } finally {
                isLoading = false;
            }
        })();

        return initPromise;
    }

    async function waitForInit() {
        if (initPromise) await initPromise;
    }

    async function login(username: string, password: string) {
        const res = await api.login(username, password);
        setTokens(res.access_token, res.refresh_token);
        user = res.user;
        isAuthenticated = true;
        settings.load();
    }

    async function register(username: string, email: string, password: string) {
        const res = await api.register(username, email, password);
        setTokens(res.access_token, res.refresh_token);
        user = res.user;
        isAuthenticated = true;
        settings.load();
    }

    function logout() {
        api.logout().catch(() => { });
        clearTokens();
        user = null;
        isAuthenticated = false;
        initPromise = null;
        favorites.reset();
        library.reset();
        settings.reset();
    }

    return {
        get user() { return user; },
        get isAuthenticated() { return isAuthenticated; },
        get isLoading() { return isLoading; },
        get isAdmin() { return user?.role === 'admin'; },
        init,
        waitForInit,
        login,
        register,
        logout
    };
}

export const auth = createAuthStore();
