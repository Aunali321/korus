import { library } from '$lib/stores/library.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
    await library.load();
    return {};
};
