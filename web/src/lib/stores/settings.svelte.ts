import type { StreamingQuality, StreamingPreset, RepeatMode } from '$lib/types';
import { api } from '$lib/api';

const STREAMING_KEY = 'korus_streaming_quality';
const PLAYBACK_KEY = 'korus_playback_settings';

const PRESETS: Record<Exclude<StreamingPreset, 'custom'>, { format: string; bitrate: number } | null> = {
    original: null,
    lossless: { format: 'wav', bitrate: 0 },
    very_high: { format: 'opus', bitrate: 256 },
    high: { format: 'opus', bitrate: 192 },
    medium: { format: 'opus', bitrate: 128 },
    low: { format: 'opus', bitrate: 64 },
};

function createSettingsStore() {
    let streamingQuality = $state<StreamingQuality>({ preset: 'original' });
    let shuffle = $state(false);
    let repeat = $state<RepeatMode>('off');
    let loaded = $state(false);
    let syncing = $state(false);

    function loadLocal() {
        if (typeof localStorage === 'undefined') return;
        try {
            const streaming = localStorage.getItem(STREAMING_KEY);
            if (streaming) {
                streamingQuality = JSON.parse(streaming);
            }
            const playback = localStorage.getItem(PLAYBACK_KEY);
            if (playback) {
                const parsed = JSON.parse(playback);
                shuffle = parsed.shuffle ?? false;
                repeat = parsed.repeat ?? 'off';
            }
        } catch {
            // ignore
        }
    }

    function saveLocal() {
        if (typeof localStorage === 'undefined') return;
        localStorage.setItem(STREAMING_KEY, JSON.stringify(streamingQuality));
        localStorage.setItem(PLAYBACK_KEY, JSON.stringify({ shuffle, repeat }));
    }

    async function load() {
        if (loaded) return;
        loadLocal();
        
        try {
            const remote = await api.getSettings();
            shuffle = remote.shuffle;
            repeat = remote.repeat as RepeatMode;
            saveLocal();
            loaded = true;
        } catch {
            loaded = true;
        }
    }

    async function syncToServer() {
        if (syncing) return;
        syncing = true;
        try {
            await api.updateSettings({ shuffle, repeat });
        } catch (err) {
            console.error('Failed to sync settings:', err);
        } finally {
            syncing = false;
        }
    }

    async function setPreset(preset: StreamingPreset) {
        if (preset === 'custom') return;
        const config = PRESETS[preset];
        streamingQuality = config
            ? { preset, format: config.format, bitrate: config.bitrate }
            : { preset };
        saveLocal();
    }

    async function setCustom(format: string, bitrate: number) {
        streamingQuality = { preset: 'custom', format, bitrate };
        saveLocal();
    }

    function getStreamParams(): { format?: string; bitrate?: number } {
        if (streamingQuality.preset === 'original') {
            return {};
        }
        return { format: streamingQuality.format, bitrate: streamingQuality.bitrate };
    }

    async function setShuffle(value: boolean) {
        shuffle = value;
        saveLocal();
        await syncToServer();
    }

    async function setRepeat(value: RepeatMode) {
        repeat = value;
        saveLocal();
        await syncToServer();
    }

    async function toggleShuffle() {
        await setShuffle(!shuffle);
    }

    async function toggleRepeat() {
        const modes: RepeatMode[] = ['off', 'all', 'one'];
        const idx = modes.indexOf(repeat);
        await setRepeat(modes[(idx + 1) % modes.length]);
    }

    function reset() {
        streamingQuality = { preset: 'original' };
        shuffle = false;
        repeat = 'off';
        loaded = false;
        if (typeof localStorage !== 'undefined') {
            localStorage.removeItem(STREAMING_KEY);
            localStorage.removeItem(PLAYBACK_KEY);
        }
    }

    loadLocal();

    return {
        get quality() { return streamingQuality; },
        get preset() { return streamingQuality.preset; },
        get format() { return streamingQuality.format; },
        get bitrate() { return streamingQuality.bitrate; },
        get shuffle() { return shuffle; },
        get repeat() { return repeat; },
        get loaded() { return loaded; },
        setPreset,
        setCustom,
        getStreamParams,
        setShuffle,
        setRepeat,
        toggleShuffle,
        toggleRepeat,
        load,
        reset,
    };
}

export const settings = createSettingsStore();
