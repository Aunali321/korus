<script lang="ts">
    import { Shield, RefreshCw, Server, Trash2 } from "lucide-svelte";
    import { api } from "$lib/api";
    import { auth } from "$lib/stores/auth.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import type { ScanJob, SystemInfo } from "$lib/types";

    let scanJob = $state<ScanJob | null>(null);
    let systemInfo = $state<SystemInfo | null>(null);
    let loading = $state(true);
    let loaded = $state(false);
    let scanning = $state(false);
    let cleanupDays = $state(30);
    let cleaning = $state(false);

    $effect(() => {
        if (auth.isAuthenticated && !loaded) {
            loadAdmin();
        }
    });

    async function loadAdmin() {
        loaded = true;
        try {
            const [status, sys] = await Promise.all([
                api.getScanStatus().catch(() => null),
                api.getSystemInfo().catch(() => null),
            ]);
            scanJob = status;
            systemInfo = sys;
        } catch (e) {
            console.error("Failed to load admin:", e);
        } finally {
            loading = false;
        }
    }

    async function startScan() {
        scanning = true;
        try {
            const job = await api.startScan();
            scanJob = job;
            toast.success("Scan started");
            pollScanStatus();
        } catch (e) {
            toast.error("Failed to start scan");
        } finally {
            scanning = false;
        }
    }

    async function pollScanStatus() {
        const poll = async () => {
            try {
                const status = await api.getScanStatus();
                scanJob = status;
                if (status.status === "running") {
                    setTimeout(poll, 2000);
                } else if (status.status === "completed") {
                    toast.success("Scan completed");
                    systemInfo = await api.getSystemInfo();
                }
            } catch {
                // Ignore polling errors
            }
        };
        poll();
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
        <section>
            <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
                <RefreshCw size={20} class="text-emerald-400" />
                Library Scan
            </h3>
            <div
                class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4"
            >
                <button
                    onclick={startScan}
                    disabled={scanning || scanJob?.status === "running"}
                    class="px-6 py-3 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-black font-semibold rounded-lg flex items-center gap-2"
                >
                    <RefreshCw
                        size={18}
                        class={scanJob?.status === "running"
                            ? "animate-spin"
                            : ""}
                    />
                    {scanJob?.status === "running"
                        ? "Scanning..."
                        : "Start Scan"}
                </button>

                {#if scanJob}
                    <div class="text-sm text-zinc-400">
                        <div>
                            Status: <span class="text-zinc-200"
                                >{scanJob.status}</span
                            >
                        </div>
                        {#if scanJob.status === "running"}
                            <div class="mt-2">
                                <div class="flex justify-between text-xs mb-1">
                                    <span
                                        >{scanJob.current_file ||
                                            "Scanning..."}</span
                                    >
                                    <span
                                        >{scanJob.progress} / {scanJob.total}</span
                                    >
                                </div>
                                <div
                                    class="h-2 bg-zinc-800 rounded-full overflow-hidden"
                                >
                                    <div
                                        class="h-full bg-emerald-500 transition-all"
                                        style="width: {scanJob.total
                                            ? (scanJob.progress /
                                                  scanJob.total) *
                                              100
                                            : 0}%"
                                    ></div>
                                </div>
                            </div>
                        {/if}
                        {#if scanJob.error}
                            <div class="text-red-400 mt-2">{scanJob.error}</div>
                        {/if}
                    </div>
                {/if}
            </div>
        </section>

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
