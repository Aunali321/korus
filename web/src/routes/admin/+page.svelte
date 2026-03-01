<script lang="ts">
    import Shield from "@lucide/svelte/icons/shield";
    import Server from "@lucide/svelte/icons/server";
    import Trash2 from "@lucide/svelte/icons/trash-2";
    import Radio from "@lucide/svelte/icons/radio";
    import Database from "@lucide/svelte/icons/database";
    import Download from "@lucide/svelte/icons/download";
    import Upload from "@lucide/svelte/icons/upload";
    import AlertTriangle from "@lucide/svelte/icons/alert-triangle";
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
    let restoring = $state(false);
    let showRestoreConfirm = $state(false);
    let restoreFile = $state<File | null>(null);
    let fileInput: HTMLInputElement;

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

    function downloadBackup() {
        const url = api.getBackupUrl();
        window.open(url, "_blank");
        toast.success("Backup download started");
    }

    function handleFileSelect(event: Event) {
        const input = event.target as HTMLInputElement;
        if (input.files && input.files.length > 0) {
            restoreFile = input.files[0];
            showRestoreConfirm = true;
        }
    }

    function cancelRestore() {
        showRestoreConfirm = false;
        restoreFile = null;
        if (fileInput) fileInput.value = "";
    }

    async function confirmRestore() {
        if (!restoreFile) return;
        restoring = true;
        try {
            const result = await api.restoreDatabase(restoreFile);
            toast.success(result.message);
            setTimeout(() => {
                toast.info("Please wait for the server to restart, then refresh the page.");
            }, 1000);
        } catch (e) {
            toast.error(e instanceof Error ? e.message : "Failed to restore database");
            cancelRestore();
        } finally {
            restoring = false;
        }
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
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-4 md:p-6">
                <div class="flex items-center justify-between gap-4">
                    <div class="min-w-0">
                        <p class="font-medium">Enable Radio</p>
                        <p class="text-sm text-zinc-400">Allow users to start radio stations from songs</p>
                    </div>
                    <button
                        onclick={toggleRadio}
                        disabled={savingRadio}
                        aria-label="Toggle radio"
                        class="relative w-12 h-6 rounded-full transition-colors shrink-0 {radioEnabled ? 'bg-emerald-500' : 'bg-zinc-700'}"
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
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-4 md:p-6">
                <div class="flex flex-wrap items-center gap-3 md:gap-4">
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

        <section>
            <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                <Database size={20} class="text-emerald-400" />
                Database Management
            </h3>
            <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-4 md:p-6 space-y-6">
                <div class="flex items-center justify-between gap-4">
                    <div class="min-w-0">
                        <p class="font-medium flex items-center gap-2">
                            <Download size={16} />
                            Backup Database
                        </p>
                        <p class="text-sm text-zinc-400">Download a complete backup of your database</p>
                    </div>
                    <button
                        onclick={downloadBackup}
                        class="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-white font-medium rounded-lg flex items-center gap-2 shrink-0"
                    >
                        <Download size={16} />
                        Download
                    </button>
                </div>

                <hr class="border-zinc-700" />

                <div class="flex items-center justify-between gap-4">
                    <div class="min-w-0">
                        <p class="font-medium flex items-center gap-2">
                            <Upload size={16} />
                            Restore Database
                        </p>
                        <p class="text-sm text-zinc-400">Restore from a backup file. Server will exit and needs to be restarted.</p>
                    </div>
                    <div class="shrink-0">
                        <input
                            bind:this={fileInput}
                            type="file"
                            accept=".db"
                            onchange={handleFileSelect}
                            class="hidden"
                            id="restore-file"
                        />
                        <label
                            for="restore-file"
                            class="px-4 py-2 bg-zinc-700 hover:bg-zinc-600 text-white font-medium rounded-lg flex items-center gap-2 cursor-pointer"
                        >
                            <Upload size={16} />
                            Select File
                        </label>
                    </div>
                </div>
            </div>
        </section>
    {/if}
</div>

<!-- Restore Confirmation Modal -->
{#if showRestoreConfirm}
    <div class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
        <div class="bg-zinc-900 border border-zinc-700 rounded-xl p-6 max-w-md w-full">
            <div class="flex items-center gap-3 mb-4 text-amber-400">
                <AlertTriangle size={24} />
                <h3 class="text-xl font-bold">Confirm Restore</h3>
            </div>
            <p class="text-zinc-300 mb-4">
                You are about to restore the database from:
            </p>
            <p class="text-sm bg-zinc-800 p-3 rounded-lg mb-4 font-mono break-all">
                {restoreFile?.name}
            </p>
            <div class="bg-amber-900/30 border border-amber-700 rounded-lg p-3 mb-6">
                <p class="text-sm text-amber-200">
                    <strong>Warning:</strong> This will replace your current database. A safety backup will be created automatically. The server will exit after restore - if you're using Docker with restart policy or systemd, it will restart automatically.
                </p>
            </div>
            <div class="flex gap-3 justify-end">
                <button
                    onclick={cancelRestore}
                    disabled={restoring}
                    class="px-4 py-2 bg-zinc-700 hover:bg-zinc-600 disabled:opacity-50 text-white font-medium rounded-lg"
                >
                    Cancel
                </button>
                <button
                    onclick={confirmRestore}
                    disabled={restoring}
                    class="px-4 py-2 bg-red-500 hover:bg-red-600 disabled:opacity-50 text-white font-medium rounded-lg flex items-center gap-2"
                >
                    {#if restoring}
                        <span class="animate-spin">⏳</span>
                        Restoring...
                    {:else}
                        Restore Database
                    {/if}
                </button>
            </div>
        </div>
    </div>
{/if}
