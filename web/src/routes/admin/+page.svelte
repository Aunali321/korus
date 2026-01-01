<script lang="ts">
    import { Shield, Server, Trash2, Radio } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import type { SystemInfo } from "$lib/types";

    let systemInfo = $state<SystemInfo | null>(null);
    let loading = $state(true);
    let loaded = $state(false);
    let cleanupDays = $state(30);
    let cleaning = $state(false);
    let radioEnabled = $state(false);
    let savingRadio = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadAdmin();
        }
    });

    async function loadAdmin() {
        loaded = true;
        try {
            systemInfo = await api.getSystemInfo().catch(() => null);
            const settings = await api.getAppSettings().catch(() => null);
            if (settings) {
                radioEnabled = settings.radio_enabled;
            }
        } catch (e) {
            console.error("Failed to load admin:", e);
        } finally {
            loading = false;
        }
    }

    async function toggleRadio() {
        savingRadio = true;
        try {
            const result = await api.updateAppSettings({ radio_enabled: !radioEnabled });
            radioEnabled = result.radio_enabled;
            toast.success(radioEnabled ? "Radio enabled" : "Radio disabled");
        } catch {
            toast.error("Failed to update radio setting");
        } finally {
            savingRadio = false;
        }
    }

    async function cleanupSessions() {
        cleaning = true;
        try {
            const result = await api.cleanupSessions(cleanupDays);
            toast.success(`Cleaned up ${result.deleted} sessions`);
        } catch {
            toast.error("Failed to cleanup sessions");
        } finally {
            cleaning = false;
        }
    }

    function formatBytes(bytes: number): string {
        if (bytes < 1024) return bytes + " B";
        if (bytes < 1048576) return (bytes / 1024).toFixed(1) + " KB";
        if (bytes < 1073741824) return (bytes / 1048576).toFixed(1) + " MB";
        return (bytes / 1073741824).toFixed(1) + " GB";
    }

    function formatUptime(seconds: number): string {
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const mins = Math.floor((seconds % 3600) / 60);
        if (days > 0) return `${days}d ${hours}h`;
        if (hours > 0) return `${hours}h ${mins}m`;
        return `${mins}m`;
    }
</script>

<div class="p-6 space-y-8">
    <div class="flex items-center gap-3">
        <Shield class="text-red-400" size={32} />
        <h2 class="text-3xl font-bold">Admin</h2>
    </div>

    {#if loading}
        <div class="flex justify-center py-12">
            <div class="text-zinc-500">Loading...</div>
        </div>
    {:else}
        {#if systemInfo}
            <section>
                <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                    <Server size={20} class="text-blue-400" />
                    System Status
                </h3>
                <div class="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
                    <div
                        class="bg-zinc-900 border border-zinc-800 rounded-xl p-4"
                    >
                        <div class="text-2xl font-bold">
                            {systemInfo.total_songs}
                        </div>
                        <div class="text-sm text-zinc-400">Songs</div>
                    </div>
                    <div
                        class="bg-zinc-900 border border-zinc-800 rounded-xl p-4"
                    >
                        <div class="text-2xl font-bold">
                            {systemInfo.total_albums}
                        </div>
                        <div class="text-sm text-zinc-400">Albums</div>
                    </div>
                    <div
                        class="bg-zinc-900 border border-zinc-800 rounded-xl p-4"
                    >
                        <div class="text-2xl font-bold">
                            {systemInfo.total_artists}
                        </div>
                        <div class="text-sm text-zinc-400">Artists</div>
                    </div>
                    <div
                        class="bg-zinc-900 border border-zinc-800 rounded-xl p-4"
                    >
                        <div class="text-2xl font-bold">
                            {formatBytes(systemInfo.database_size)}
                        </div>
                        <div class="text-sm text-zinc-400">Database Size</div>
                    </div>
                </div>
                <div class="mt-4 text-sm text-zinc-500">
                    Version: {systemInfo.version} • Uptime: {formatUptime(
                        systemInfo.uptime,
                    )}
                </div>
            </section>
        {/if}

        <section>
            <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                <Radio size={20} class="text-emerald-400" />
                Radio Settings
            </h3>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="flex items-center justify-between">
                    <div>
                        <p class="font-medium">Enable Radio</p>
                        <p class="text-sm text-zinc-400">Allow users to start radio stations from songs</p>
                    </div>
                    <button
                        onclick={toggleRadio}
                        disabled={savingRadio}
                        aria-label="Toggle radio"
                        class="relative w-12 h-6 rounded-full transition-colors {radioEnabled ? 'bg-emerald-500' : 'bg-zinc-700'}"
                    >
                        <span
                            class="absolute top-1 left-1 w-4 h-4 bg-white rounded-full transition-transform {radioEnabled ? 'translate-x-6' : ''}"
                        ></span>
                    </button>
                </div>
            </div>
        </section>

        <section>
            <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                <Trash2 size={20} class="text-red-400" />
                Session Cleanup
            </h3>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div class="flex items-center gap-4">
                    <label for="cleanup-days" class="text-sm text-zinc-400"
                        >Delete sessions older than</label
                    >
                    <input
                        id="cleanup-days"
                        type="number"
                        bind:value={cleanupDays}
                        min="1"
                        class="w-20 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg"
                    />
                    <span class="text-zinc-400">days</span>
                    <button
                        onclick={cleanupSessions}
                        disabled={cleaning}
                        class="px-4 py-2 bg-red-500 hover:bg-red-600 disabled:opacity-50 text-white font-medium rounded-lg"
                    >
                        {cleaning ? "Cleaning..." : "Cleanup"}
                    </button>
                </div>
            </div>
        </section>
    {/if}
</div>
