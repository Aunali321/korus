<script lang="ts">
    import { X, CheckCircle, AlertCircle, Info } from "lucide-svelte";
    import { toast, type Toast } from "$lib/stores/toast.svelte";

    const icons = {
        success: CheckCircle,
        error: AlertCircle,
        info: Info,
    };

    const colors = {
        success: "bg-emerald-500/20 border-emerald-500/50 text-emerald-400",
        error: "bg-red-500/20 border-red-500/50 text-red-400",
        info: "bg-blue-500/20 border-blue-500/50 text-blue-400",
    };
</script>

<div class="fixed bottom-28 right-4 z-50 flex flex-col gap-2">
    {#each toast.toasts as t (t.id)}
        {@const Icon = icons[t.type]}
        <div
            class="flex items-center gap-3 px-4 py-3 rounded-lg border backdrop-blur-sm {colors[
                t.type
            ]}"
        >
            <Icon size={18} />
            <span class="text-sm">{t.message}</span>
            <button
                onclick={() => toast.remove(t.id)}
                class="p-1 hover:opacity-70"
            >
                <X size={14} />
            </button>
        </div>
    {/each}
</div>
