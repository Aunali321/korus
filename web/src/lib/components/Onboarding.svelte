<script lang="ts">
    import { X, ChevronRight, ChevronLeft, Music, Settings, FolderSync } from "lucide-svelte";
    import { api } from "$lib/api";
    import { settings } from "$lib/stores/settings.svelte";
    import type { StreamingPreset } from "$lib/types";

    let { onComplete }: { onComplete: () => void } = $props();

    let step = $state(0);
    let scanning = $state(false);
    let scanStatus = $state<string | null>(null);
    let scanPhase = $state<string | null>(null);

    const presets: { value: StreamingPreset; label: string; desc: string }[] = [
        { value: "original", label: "Original", desc: "No transcoding, highest quality" },
        { value: "lossless", label: "Lossless", desc: "WAV format, large files" },
        { value: "very_high", label: "Very High", desc: "Opus 256kbps" },
        { value: "high", label: "High", desc: "Opus 192kbps" },
        { value: "medium", label: "Medium", desc: "Opus 128kbps, balanced" },
        { value: "low", label: "Low", desc: "Opus 64kbps, saves data" },
    ];

    const phaseLabels: Record<string, string> = {
        scanning: "Scanning files",
        enriching: "Enriching metadata",
        processing: "Processing artists",
        cleanup: "Cleaning up",
        playlists: "Importing playlists",
        completed: "Complete",
    };

    async function startScan() {
        scanning = true;
        scanStatus = "Starting scan...";
        scanPhase = null;
        try {
            await api.startScan();
            pollScanStatus();
        } catch (e) {
            scanStatus = "Failed to start scan";
            scanning = false;
        }
    }

    async function pollScanStatus() {
        try {
            const status = await api.getScanStatus();
            scanPhase = status.phase;
            if (status.status === "running") {
                const phaseLabel = phaseLabels[status.phase] || status.phase;
                if (status.phase === "scanning" || status.phase === "enriching" || status.phase === "processing") {
                    scanStatus = `${phaseLabel}: ${status.progress}/${status.total}`;
                } else {
                    scanStatus = phaseLabel;
                }
                setTimeout(pollScanStatus, 1000);
            } else if (status.status === "completed") {
                scanStatus = `Scan complete: ${status.total} files processed`;
                scanPhase = "completed";
                scanning = false;
            } else {
                scanStatus = status.status;
                scanning = false;
            }
        } catch {
            scanning = false;
        }
    }

    async function finish() {
        try {
            await api.completeOnboarding();
        } catch (e) {
            console.error("Failed to complete onboarding:", e);
        }
        onComplete();
    }

    function next() {
        if (step < 2) step++;
        else finish();
    }

    function prev() {
        if (step > 0) step--;
    }
</script>

<div class="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl max-w-lg w-full p-6 relative">
        <button
            onclick={finish}
            class="absolute top-4 right-4 p-1 text-zinc-500 hover:text-zinc-300"
        >
            <X size={20} />
        </button>

        <div class="flex gap-2 mb-6">
            {#each [0, 1, 2] as i}
                <div class="flex-1 h-1 rounded-full {step >= i ? 'bg-emerald-500' : 'bg-zinc-700'}"></div>
            {/each}
        </div>

        {#if step === 0}
            <div class="text-center">
                <div class="w-16 h-16 bg-emerald-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                    <Music size={32} class="text-emerald-500" />
                </div>
                <h2 class="text-2xl font-bold mb-2">Welcome to Korus</h2>
                <p class="text-zinc-400 mb-6">
                    Let's get you set up with a few quick steps.
                </p>
            </div>
        {:else if step === 1}
            <div>
                <div class="w-12 h-12 bg-emerald-500/20 rounded-full flex items-center justify-center mb-4">
                    <Settings size={24} class="text-emerald-500" />
                </div>
                <h2 class="text-xl font-bold mb-2">Streaming Quality</h2>
                <p class="text-zinc-400 text-sm mb-4">
                    Choose your preferred audio quality. You can change this later in settings.
                </p>
                <div class="space-y-2 max-h-64 overflow-y-auto">
                    {#each presets as preset}
                        <button
                            onclick={() => settings.setPreset(preset.value)}
                            class="w-full p-3 rounded-lg border text-left transition-colors {settings.preset === preset.value
                                ? 'border-emerald-500 bg-emerald-500/10'
                                : 'border-zinc-700 hover:border-zinc-600'}"
                        >
                            <div class="font-medium">{preset.label}</div>
                            <div class="text-xs text-zinc-400">{preset.desc}</div>
                        </button>
                    {/each}
                </div>
            </div>
        {:else if step === 2}
            <div>
                <div class="w-12 h-12 bg-emerald-500/20 rounded-full flex items-center justify-center mb-4">
                    <FolderSync size={24} class="text-emerald-500" />
                </div>
                <h2 class="text-xl font-bold mb-2">Scan Your Library</h2>
                <p class="text-zinc-400 text-sm mb-4">
                    Scan your music folder to import your library. This may take a while depending on size.
                </p>
                <button
                    onclick={startScan}
                    disabled={scanning}
                    class="w-full py-3 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 text-black font-medium rounded-lg transition-colors"
                >
                    {scanning ? "Scanning..." : "Start Scan"}
                </button>
                {#if scanStatus}
                    <p class="text-sm text-zinc-400 mt-3 text-center">{scanStatus}</p>
                {/if}
                <p class="text-xs text-zinc-500 mt-4 text-center">
                    You can skip this and scan later from settings.
                </p>
            </div>
        {/if}

        <div class="flex justify-between mt-6 pt-4 border-t border-zinc-800">
            <button
                onclick={prev}
                disabled={step === 0}
                class="flex items-center gap-1 px-4 py-2 text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:cursor-not-allowed"
            >
                <ChevronLeft size={18} />
                Back
            </button>
            <button
                onclick={next}
                class="flex items-center gap-1 px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-black font-medium rounded-lg"
            >
                {step === 2 ? "Finish" : "Next"}
                {#if step < 2}
                    <ChevronRight size={18} />
                {/if}
            </button>
        </div>
    </div>
</div>
