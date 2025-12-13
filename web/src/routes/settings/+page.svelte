<script lang="ts">
    import { Settings, LogOut, Server } from "lucide-svelte";
    import { auth } from "$lib/stores/auth.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import { setApiUrl } from "$lib/api";
    import { goto } from "$app/navigation";

    let apiUrl = $state("");

    $effect(() => {
        if (typeof localStorage !== "undefined") {
            apiUrl = localStorage.getItem("korus_api_url") || "/api";
        }
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
