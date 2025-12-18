import type { StreamingQuality, StreamingPreset } from '$lib/types';
import { api } from '$lib/api';

const STORAGE_KEY = 'korus_streaming_quality';

const PRESETS: Record<Exclude<StreamingPreset, 'custom'>, { format: string; bitrate: number } | null> = {
    original: null,
    lossless: { format: 'wav', bitrate: 0 },
    very_high: { format: 'opus', bitrate: 256 },
    high: { format: 'opus', bitrate: 192 },
    medium: { format: 'opus', bitrate: 128 },
    low: { format: 'opus', bitrate: 64 },
};

function createSettingsStore() {
    let quality = $state<StreamingQuality>({ preset: 'original' });
    let loaded = $state(false);
    let syncing = $state(false);

    function loadLocal() {
        if (typeof localStorage === 'undefined') return;
        try {
            const stored = localStorage.getItem(STORAGE_KEY);
            if (stored) {
                quality = JSON.parse(stored);
            }
        } catch {
            // ignore
        }
    }

    function saveLocal() {
        if (typeof localStorage === 'undefined') return;
        localStorage.setItem(STORAGE_KEY, JSON.stringify(quality));
    }

    async function load() {
        if (loaded) return;
        loadLocal(); // Load local first for instant UI
        
        try {
            const remote = await api.getSettings();
            quality = {
                preset: remote.streaming_preset as StreamingPreset,
                format: remote.streaming_format,
                bitrate: remote.streaming_bitrate,
            };
            saveLocal();
            loaded = true;
        } catch {
            // Use local settings if API fails
            loaded = true;
        }
    }

    async function syncToServer() {
        if (syncing) return;
        syncing = true;
        try {
            await api.updateSettings({
                streaming_preset: quality.preset,
                streaming_format: quality.format,
                streaming_bitrate: quality.bitrate,
            });
        } catch (err) {
            console.error('Failed to sync settings:', err);
        } finally {
            syncing = false;
        }
    }

    async function setPreset(preset: StreamingPreset) {
        if (preset === 'custom') return;
        const config = PRESETS[preset];
        quality = config
            ? { preset, format: config.format, bitrate: config.bitrate }
            : { preset };
        saveLocal();
        await syncToServer();
    }

    async function setCustom(format: string, bitrate: number) {
        quality = { preset: 'custom', format, bitrate };
        saveLocal();
        await syncToServer();
    }

    function getStreamParams(): { format?: string; bitrate?: number } {
        if (quality.preset === 'original') {
            return {};
        }
        return { format: quality.format, bitrate: quality.bitrate };
    }

    function reset() {
        quality = { preset: 'original' };
        loaded = false;
        if (typeof localStorage !== 'undefined') {
            localStorage.removeItem(STORAGE_KEY);
        }
    }

    // Load local on init for instant availability
    loadLocal();

    return {
        get quality() { return quality; },
        get preset() { return quality.preset; },
        get format() { return quality.format; },
        get bitrate() { return quality.bitrate; },
        get loaded() { return loaded; },
        setPreset,
        setCustom,
        getStreamParams,
        load,
        reset,
    };
}

export const settings = createSettingsStore();
