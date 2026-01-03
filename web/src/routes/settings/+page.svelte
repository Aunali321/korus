<script lang="ts">
    import { Settings, LogOut, Server, Volume2, RefreshCw, Radio } from "lucide-svelte";
    import { auth } from "$lib/stores/auth.svelte";
    import { settings } from "$lib/stores/settings.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import { library } from "$lib/stores/library.svelte";
    import { api, setApiUrl } from "$lib/api";
    import { goto } from "$app/navigation";
    import type { StreamingOptions, StreamingPreset, ScanJob } from "$lib/types";

    let apiUrl = $state("");
    let showAdvanced = $state(false);
    let streamingOptions = $state<StreamingOptions | null>(null);
    let customFormat = $state("opus");
    let customBitrate = $state(128);
    let scanJob = $state<ScanJob | null>(null);
    let scanning = $state(false);

    const phaseLabels: Record<string, string> = {
        scanning: "Scanning files",
        enriching: "Enriching metadata",
        processing: "Processing artists",
        cleanup: "Cleaning up",
        playlists: "Importing playlists",
        completed: "Complete",
    };

    const presets: { value: StreamingPreset; label: string; description: string }[] = [
        { value: "original", label: "Original", description: "No transcoding (~600 MB/hr for lossless)" },
        { value: "lossless", label: "Lossless", description: "WAV transcoding (~635-950 MB/hr)" },
        { value: "very_high", label: "Very High", description: "Opus 256 kbps (~115 MB/hr)" },
        { value: "high", label: "High", description: "Opus 192 kbps (~86 MB/hr)" },
        { value: "medium", label: "Medium", description: "Opus 128 kbps (~58 MB/hr)" },
        { value: "low", label: "Low", description: "Opus 64 kbps (~29 MB/hr)" },
    ];

    $effect(() => {
        if (typeof localStorage !== "undefined") {
            apiUrl = localStorage.getItem("korus_api_url") || "/api";
        }
    });

    $effect(() => {
        if (settings.preset === "custom") {
            showAdvanced = true;
            customFormat = settings.format || "opus";
            customBitrate = settings.bitrate || 128;
        }
    });

    $effect(() => {
        api.getStreamingOptions().then((opts) => {
            streamingOptions = opts;
        }).catch(() => {
            // ignore
        });
        api.getScanStatus().then((status) => {
            if (status.status === "running") {
                scanJob = status;
                pollScanStatus();
            }
        }).catch(() => {
            // ignore
        });
    });

    function saveApiUrl() {
        if (!apiUrl.trim()) return;
        setApiUrl(apiUrl);
        toast.success("API URL updated");
    }

    function handleLogout() {
        auth.logout();
        goto("/login");
    }

    function handlePresetChange(preset: StreamingPreset) {
        settings.setPreset(preset);
        toast.success("Streaming quality updated");
    }

    function handleCustomChange() {
        settings.setCustom(customFormat, customBitrate);
        toast.success("Streaming quality updated");
    }

    const availableBitrates = $derived(
        streamingOptions?.formats.find(f => f.format === customFormat)?.bitrates || []
    );

    async function startScan() {
        scanning = true;
        try {
            await api.startScan();
            toast.success("Scan started");
            pollScanStatus();
        } catch {
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
                    setTimeout(poll, 1000);
                } else if (status.status === "completed") {
                    toast.success("Scan completed");
                    library.invalidate();
                    scanJob = null;
                }
            } catch {
                // Ignore polling errors
            }
        };
        poll();
    }
</script>

<div class="p-6 space-y-8">
    <div class="flex items-center gap-3">
        <Settings class="text-zinc-400" size={32} />
        <h2 class="text-3xl font-bold">Settings</h2>
    </div>

    <section>
        <h3 class="text-xl font-bold mb-4">Account</h3>
        <div
            class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4"
        >
            {#if auth.user}
                <div class="flex items-center gap-4">
                    <div
                        class="w-16 h-16 rounded-full bg-gradient-to-br from-emerald-700 to-cyan-800 flex items-center justify-center"
                    >
                        <span class="text-2xl font-bold"
                            >{auth.user.username.charAt(0).toUpperCase()}</span
                        >
                    </div>
                    <div>
                        <div class="text-xl font-semibold">
                            {auth.user.username}
                        </div>
                        <div class="text-sm text-zinc-400">
                            {auth.user.email}
                        </div>
                        {#if auth.isAdmin}
                            <span
                                class="text-xs px-2 py-0.5 bg-red-500/20 text-red-400 rounded mt-1 inline-block"
                                >Admin</span
                            >
                        {/if}
                    </div>
                </div>
            {/if}

            <button
                onclick={handleLogout}
                class="flex items-center gap-2 px-4 py-2 bg-zinc-800 hover:bg-zinc-700 rounded-lg text-red-400"
            >
                <LogOut size={18} />
                Sign Out
            </button>
        </div>
    </section>

    <section>
        <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
            <Volume2 size={20} class="text-zinc-400" />
            Streaming Quality
        </h3>
        <div
            class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4"
        >
            {#if streamingOptions && !streamingOptions.ffmpeg_available}
                <div class="text-sm text-amber-400 bg-amber-400/10 px-3 py-2 rounded-lg">
                    FFmpeg not available on server. Only original quality is supported.
                </div>
            {/if}

            <div class="space-y-2">
                {#each presets as preset}
                    <label
                        class="flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors {settings.preset === preset.value ? 'bg-zinc-800' : 'hover:bg-zinc-800/50'}"
                    >
                        <input
                            type="radio"
                            name="quality"
                            value={preset.value}
                            checked={settings.preset === preset.value}
                            onchange={() => handlePresetChange(preset.value)}
                            disabled={preset.value !== 'original' && streamingOptions && !streamingOptions.ffmpeg_available}
                            class="w-4 h-4 accent-emerald-500"
                        />
                        <div>
                            <div class="font-medium">{preset.label}</div>
                            <div class="text-sm text-zinc-400">{preset.description}</div>
                        </div>
                    </label>
                {/each}
            </div>

            <div class="border-t border-zinc-800 pt-4">
                <label class="flex items-center gap-2 cursor-pointer">
                    <input
                        type="checkbox"
                        bind:checked={showAdvanced}
                        disabled={streamingOptions && !streamingOptions.ffmpeg_available}
                        class="w-4 h-4 accent-emerald-500"
                    />
                    <span class="text-sm text-zinc-400">Advanced options</span>
                </label>

                {#if showAdvanced && streamingOptions?.ffmpeg_available}
                    <div class="mt-4 flex gap-4">
                        <div class="flex-1">
                            <label for="format" class="block text-sm text-zinc-400 mb-1">Format</label>
                            <select
                                id="format"
                                bind:value={customFormat}
                                onchange={handleCustomChange}
                                class="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg"
                            >
                                {#each streamingOptions.formats as fmt}
                                    <option value={fmt.format}>{fmt.format.toUpperCase()}</option>
                                {/each}
                            </select>
                        </div>
                        <div class="flex-1">
                            <label for="bitrate" class="block text-sm text-zinc-400 mb-1">Bitrate</label>
                            <select
                                id="bitrate"
                                bind:value={customBitrate}
                                onchange={handleCustomChange}
                                class="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg"
                            >
                                {#each availableBitrates as br}
                                    <option value={br}>{br} kbps</option>
                                {/each}
                            </select>
                        </div>
                    </div>
                    <p class="text-xs text-zinc-500 mt-2">
                        Custom settings override presets. Opus is recommended for best quality-to-size ratio.
                    </p>
                {/if}
            </div>
        </div>
    </section>

    {#if settings.radioEnabled}
    <section>
        <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
            <Radio size={20} class="text-zinc-400" />
            Radio Mode
        </h3>
        <div
            class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4"
        >
            <div class="space-y-2">
                <label
                    class="flex items-start gap-3 p-3 rounded-lg cursor-pointer transition-colors {settings.radioMode === 'curator' ? 'bg-zinc-800' : 'hover:bg-zinc-800/50'}"
                >
                    <input
                        type="radio"
                        name="radioMode"
                        value="curator"
                        checked={settings.radioMode === 'curator'}
                        onchange={() => settings.setRadioMode('curator')}
                        class="w-4 h-4 mt-1 accent-emerald-500"
                    />
                    <div>
                        <div class="font-medium">Curator</div>
                        <div class="text-sm text-zinc-400">Better quality recommendations</div>
                        <div class="text-xs text-zinc-500 mt-1">Best for: Most queries, indie, discovery</div>
                    </div>
                </label>
                <label
                    class="flex items-start gap-3 p-3 rounded-lg cursor-pointer transition-colors {settings.radioMode === 'mainstream' ? 'bg-zinc-800' : 'hover:bg-zinc-800/50'}"
                >
                    <input
                        type="radio"
                        name="radioMode"
                        value="mainstream"
                        checked={settings.radioMode === 'mainstream'}
                        onchange={() => settings.setRadioMode('mainstream')}
                        class="w-4 h-4 mt-1 accent-emerald-500"
                    />
                    <div>
                        <div class="font-medium">Mainstream</div>
                        <div class="text-sm text-zinc-400">More predictable recommendations</div>
                        <div class="text-xs text-zinc-500 mt-1">Best for: Mainstream music only</div>
                    </div>
                </label>
            </div>
        </div>
    </section>
    {/if}

    <section>
        <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
            <RefreshCw size={20} class="text-zinc-400" />
            Library
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
                    : "Rescan Library"}
            </button>

            {#if scanJob}
                <div class="text-sm text-zinc-400">
                    <div>
                        Status: <span class="text-zinc-200"
                            >{scanJob.status === "running" ? (phaseLabels[scanJob.phase] || scanJob.phase) : scanJob.status}</span
                        >
                    </div>
                    {#if scanJob.status === "running"}
                        <div class="mt-2">
                            <div class="flex justify-between text-xs mb-1">
                                <span
                                    >{scanJob.phase === "scanning" ? (scanJob.current_file || "Scanning...") : (phaseLabels[scanJob.phase] || scanJob.phase)}</span
                                >
                                {#if scanJob.phase === "scanning" || scanJob.phase === "enriching" || scanJob.phase === "processing"}
                                <span
                                    >{scanJob.progress} / {scanJob.total}</span
                                >
                                {/if}
                            </div>
                            {#if scanJob.phase === "scanning" || scanJob.phase === "enriching" || scanJob.phase === "processing"}
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
                            {/if}
                        </div>
                    {/if}
                    {#if scanJob.error}
                        <div class="text-red-400 mt-2">{scanJob.error}</div>
                    {/if}
                </div>
            {/if}
            <p class="text-xs text-zinc-500">
                Rescan your music library to detect new or modified files.
            </p>
        </div>
    </section>

    <section>
        <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
            <Server size={20} class="text-zinc-400" />
            API Configuration
        </h3>
        <div
            class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4"
        >
            <div>
                <label
                    for="apiUrl"
                    class="block text-sm font-medium text-zinc-400 mb-2"
                    >API Base URL</label
                >
                <div class="flex gap-2">
                    <input
                        id="apiUrl"
                        type="text"
                        bind:value={apiUrl}
                        class="flex-1 px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500"
                        placeholder="/api or https://your-server.com/api"
                    />
                    <button
                        onclick={saveApiUrl}
                        class="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-black font-medium rounded-lg"
                    >
                        Save
                    </button>
                </div>
                <p class="text-xs text-zinc-500 mt-2">
                    Default: /api (same origin). Change this if your API is on a
                    different server.
                </p>
            </div>
        </div>
    </section>
</div>
