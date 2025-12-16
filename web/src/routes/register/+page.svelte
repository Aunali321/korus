<script lang="ts">
    import { goto } from "$app/navigation";
    import { auth } from "$lib/stores/auth.svelte";
    import { toast } from "$lib/stores/toast.svelte";
    import { setApiUrl } from "$lib/api";
    import { ChevronDown, Server } from "lucide-svelte";

    let username = $state("");
    let email = $state("");
    let password = $state("");
    let confirmPassword = $state("");
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

        if (!username || !email || !password) {
            error = "Please fill in all fields";
            return;
        }

        if (password !== confirmPassword) {
            error = "Passwords do not match";
            return;
        }

        if (password.length < 8) {
            error = "Password must be at least 8 characters";
            return;
        }

        loading = true;
        error = "";

        try {
            await auth.register(username, email, password);
            toast.success("Account created successfully!");
            goto("/");
        } catch (err) {
            error = err instanceof Error ? err.message : "Registration failed";
        } finally {
            loading = false;
        }
    }
</script>

<div class="min-h-screen flex items-center justify-center p-4">
    <div class="w-full max-w-md">
        <div class="text-center mb-8">
            <h1
                class="text-4xl font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent"
            >
                Korus
            </h1>
            <p class="text-zinc-500 mt-2">Self-hosted Music</p>
        </div>

        <form
            onsubmit={handleSubmit}
            class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4"
        >
            <h2 class="text-xl font-semibold text-center mb-4">
                Create Account
            </h2>

            {#if error}
                <div
                    class="bg-red-500/10 border border-red-500/50 text-red-400 px-4 py-2 rounded-lg text-sm"
                >
                    {error}
                </div>
            {/if}

            <div>
                <label
                    for="username"
                    class="block text-sm font-medium text-zinc-400 mb-1"
                    >Username</label
                >
                <input
                    id="username"
                    type="text"
                    bind:value={username}
                    class="w-full px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    placeholder="Choose a username"
                />
            </div>

            <div>
                <label
                    for="email"
                    class="block text-sm font-medium text-zinc-400 mb-1"
                    >Email</label
                >
                <input
                    id="email"
                    type="email"
                    bind:value={email}
                    class="w-full px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    placeholder="Enter your email"
                />
            </div>

            <div>
                <label
                    for="password"
                    class="block text-sm font-medium text-zinc-400 mb-1"
                    >Password</label
                >
                <input
                    id="password"
                    type="password"
                    bind:value={password}
                    class="w-full px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    placeholder="Choose a password (min 8 chars)"
                />
            </div>

            <div>
                <label
                    for="confirmPassword"
                    class="block text-sm font-medium text-zinc-400 mb-1"
                    >Confirm Password</label
                >
                <input
                    id="confirmPassword"
                    type="password"
                    bind:value={confirmPassword}
                    class="w-full px-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                    placeholder="Confirm your password"
                />
            </div>

            <button
                type="submit"
                disabled={loading}
                class="w-full py-3 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 disabled:cursor-not-allowed text-black font-semibold rounded-lg transition-colors"
            >
                {loading ? "Creating account..." : "Create Account"}
            </button>

            <p class="text-center text-sm text-zinc-500">
                Already have an account?
                <a href="/login" class="text-emerald-400 hover:underline"
                    >Sign in</a
                >
            </p>

            <div class="border-t border-zinc-800 pt-4">
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
