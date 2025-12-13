<script lang="ts">
    import { goto } from "$app/navigation";
    import { auth } from "$lib/stores/auth.svelte";
    import { toast } from "$lib/stores/toast.svelte";

    let username = $state("");
    let password = $state("");
    let loading = $state(false);
    let error = $state("");

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
            <h2 class="text-xl font-semibold text-center mb-4">Sign In</h2>

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
                    placeholder="Enter your username"
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
                    placeholder="Enter your password"
                />
            </div>

            <button
                type="submit"
                disabled={loading}
                class="w-full py-3 bg-emerald-500 hover:bg-emerald-600 disabled:opacity-50 disabled:cursor-not-allowed text-black font-semibold rounded-lg transition-colors"
            >
                {loading ? "Signing in..." : "Sign In"}
            </button>

            <p class="text-center text-sm text-zinc-500">
                Don't have an account?
                <a href="/register" class="text-emerald-400 hover:underline"
                    >Sign up</a
                >
            </p>
        </form>
    </div>
</div>
