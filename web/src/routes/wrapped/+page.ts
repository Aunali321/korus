import { wrappedCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

function isWrappedSeason(): boolean {
    const now = new Date();
    const month = now.getMonth();
    const date = now.getDate();
    const lastDay = new Date(now.getFullYear(), month + 1, 0).getDate();
    return month === 11 || lastDay - date < 7;
}

export const load: PageLoad = async ({ url }) => {
    const period = url.searchParams.get('period') === 'month' ? 'month' : 'year';
    if (!isWrappedSeason()) {
        return { period: period as 'year' | 'month', wrapped: null, inSeason: false };
    }
    const wrapped = await wrappedCache.load(period);
    return { period: period as 'year' | 'month', wrapped, inSeason: true };
};
