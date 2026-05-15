import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { auth } from '$lib/stores/auth.svelte';
import { settings } from '$lib/stores/settings.svelte';
import { player } from '$lib/stores/player.svelte';
import type { LayoutLoad } from './$types';

export const prerender = false;
export const ssr = false;

const PUBLIC_ROUTES = ['/login', '/register', '/setup'];

export const load: LayoutLoad = async ({ url }) => {
    if (!browser) return {};

    await auth.init();

    const isPublic = PUBLIC_ROUTES.some((r) => url.pathname.startsWith(r));

    if (!auth.isAuthenticated && !isPublic) {
        throw redirect(307, '/login');
    }

    if (auth.isAuthenticated) {
        await Promise.all([settings.load(), player.loadState()]);
    }

    return {};
};
