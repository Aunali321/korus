<script lang="ts">
    import { goto } from "$app/navigation";
    import { auth } from "$lib/stores/auth.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import { setApiUrl } from "$lib/api";
    import { ChevronDown, Server } from "lucide-svelte";

    let username = $state("");
    let password = $state("");
    let loading = $state(false);
    let error = $state("");
    let showServerConfig = $state(false);
    let apiUrl = $state("");

    $effect(() => {
        if (typeof localStorage !== "undefined") {
            apiUrl = localStorage.getItem("korus_api_url") || "/api";
        }
    });

    function saveApiUrl() {
        if (!apiUrl.trim()) return;
        setApiUrl(apiUrl);
        toast.success("Server URL updated");
    }

    async function handleSubmit(e: Event) {
        e.preventDefault();
        if (!username || !password) {
            error = "Please fill in all fields";
            return;
        }

        loading = true;
        error = "";

        try {
            await auth.login(username, password);
            toast.success("Welcome back!");
            goto("/");
        } catch (err) {
            error = err instanceof Error ? err.message : "Login failed";
        } finally {
            loading = false;
        }
    }
</script>

<div class="min-h-screen flex items-center justify-center p-6 bg-grid">
    <div class="w-full max-w-sm">
        <div class="text-center mb-10">
            <h1 class="text-5xl font-bold text-emerald-400 tracking-tight">
                Korus
            </h1>
            <p class="text-zinc-500 mt-3 text-sm tracking-wide">Self-hosted Music</p>
        </div>

        <form
            onsubmit={handleSubmit}
            class="bg-zinc-900 border border-zinc-800 rounded-lg p-8 space-y-6"
        >
            {#if error}
                <div
                    class="bg-red-500/10 border border-red-500/50 text-red-400 px-4 py-3 rounded text-sm"
                >
                    {error}
                </div>
            {/if}

            <div class="space-y-2">
                <label
                    for="username"
                    class="block text-sm font-medium text-zinc-300"
                    >Username</label
                >
                <input
                    id="username"
                    type="text"
                    bind:value={username}
                    class="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    placeholder="Enter your username"
                />
            </div>

            <div class="space-y-2">
                <label
                    for="password"
                    class="block text-sm font-medium text-zinc-300"
                    >Password</label
                >
                <input
                    id="password"
                    type="password"
                    bind:value={password}
                    class="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    placeholder="Enter your password"
                />
            </div>

            <button
                type="submit"
                disabled={loading}
                class="w-full py-3 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 disabled:cursor-not-allowed text-black font-semibold rounded-lg transition-colors mt-2"
            >
                {loading ? "Signing in..." : "Sign In"}
            </button>

            <p class="text-center text-sm text-zinc-500 pt-2">
                Don't have an account?
                <a href="/register" class="text-emerald-400 hover:underline"
                    >Sign up</a
                >
            </p>

            <div class="border-t border-zinc-800 pt-6 mt-6">
                <button
                    type="button"
                    onclick={() => (showServerConfig = !showServerConfig)}
                    class="flex items-center gap-2 text-sm text-zinc-400 hover:text-zinc-300 w-full"
                >
                    <Server size={16} />
                    <span>Server Configuration</span>
                    <ChevronDown
                        size={16}
                        class="ml-auto transition-transform {showServerConfig ? 'rotate-180' : ''}"
                    />
                </button>

                {#if showServerConfig}
                    <div class="mt-3 space-y-2">
                        <label
                            for="apiUrl"
                            class="block text-sm font-medium text-zinc-400"
                            >API Base URL</label
                        >
                        <div class="flex gap-2">
                            <input
                                id="apiUrl"
                                type="text"
                                bind:value={apiUrl}
                                class="flex-1 px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 text-sm"
                                placeholder="/api or https://your-server.com/api"
                            />
                            <button
                                type="button"
                                onclick={saveApiUrl}
                                class="px-3 py-2 bg-zinc-700 hover:bg-zinc-600 text-sm rounded-lg"
                            >
                                Save
                            </button>
                        </div>
                        <p class="text-xs text-zinc-500">
                            Default: /api. Change if connecting to a different server.
                        </p>
                    </div>
                {/if}
            </div>
        </form>
    </div>
</div>
