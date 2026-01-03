import type { SearchResults } from '$lib/types';

function createSearchStore() {
    let query = $state('');
    let results = $state<SearchResults | null>(null);
    let activeTab = $state<'all' | 'songs' | 'albums' | 'artists' | 'playlists'>('all');

    function setQuery(q: string) {
        query = q;
    }

    function setResults(r: SearchResults | null) {
        results = r;
    }

    function setActiveTab(tab: typeof activeTab) {
        activeTab = tab;
    }

    function reset() {
        query = '';
        results = null;
        activeTab = 'all';
    }

    return {
        get query() { return query; },
        get results() { return results; },
        get activeTab() { return activeTab; },
        setQuery,
        setResults,
        setActiveTab,
        reset
    };
}

export const search = createSearchStore();
