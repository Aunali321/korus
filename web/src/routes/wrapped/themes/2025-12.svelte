<script lang="ts">
    import { fly, fade, slide } from "svelte/transition";
    import { cubicOut, elasticOut, backOut } from "svelte/easing";

    interface WrappedSong {
        id: number;
        title: string;
        artist?: { id: number; name: string };
        plays: number;
    }

    interface WrappedArtist {
        id: number;
        name: string;
        plays: number;
    }

    interface WrappedAlbum {
        id: number;
        title: string;
        artist?: { id: number; name: string };
        plays: number;
    }

    interface WrappedData {
        period: string;
        top_songs: WrappedSong[];
        top_artists: WrappedArtist[];
        top_albums: WrappedAlbum[];
        total_minutes: number;
        total_plays: number;
        days_listened: number;
        avg_plays_per_day: number;
        unique_songs: number;
        unique_artists: number;
        personality?: string;
        milestones: string[];
    }

    let { wrapped }: { wrapped: WrappedData } = $props();

    function formatMinutes(mins: number): string {
        if (!mins || isNaN(mins)) return "0";
        const hours = Math.floor(mins / 60);
        if (hours > 0) return hours.toLocaleString();
        return String(mins);
    }

    function getUnit(mins: number): string {
        const hours = Math.floor(mins / 60);
        return hours > 0 ? "hours" : "minutes";
    }

    function splitNumber(num: number): string[] {
        return num.toLocaleString().split("");
    }
</script>

<div class="wrapped">
    <!-- Cinematic scanlines overlay -->
    <div class="scanlines"></div>

    <!-- Animated mesh gradient background -->
    <div class="mesh-bg">
        <div class="mesh-layer mesh-1"></div>
        <div class="mesh-layer mesh-2"></div>
        <div class="mesh-layer mesh-3"></div>
    </div>

    <!-- Noise texture overlay -->
    <div class="noise"></div>

    <div class="viewport">
        <!-- Left vertical strip - branding -->
        <aside
            class="sidebar"
            in:slide={{ axis: "x", duration: 800, easing: cubicOut }}
        >
            <div class="brand">
                <span class="brand-year">2025</span>
                <span class="brand-month">DEC</span>
            </div>
            <div class="brand-line"></div>
            <span class="brand-label">WRAPPED</span>
        </aside>

        <!-- Main content area -->
        <main class="main">
            <!-- Hero section - massive typography -->
            <section class="hero" in:fade={{ duration: 600, delay: 200 }}>
                <div class="hero-stat">
                    <div
                        class="hero-digits"
                        in:fly={{
                            y: 100,
                            duration: 1000,
                            delay: 400,
                            easing: backOut,
                        }}
                    >
                        {#each splitNumber(parseInt(formatMinutes(wrapped.total_minutes))) as digit, i}
                            <span
                                class="digit"
                                style="animation-delay: {i * 0.1}s"
                                >{digit}</span
                            >
                        {/each}
                    </div>
                    <div
                        class="hero-label"
                        in:fly={{ x: -50, duration: 600, delay: 800 }}
                    >
                        <span class="hero-unit"
                            >{getUnit(wrapped.total_minutes)}</span
                        >
                        <span class="hero-desc">of pure sound</span>
                    </div>
                </div>
            </section>

            <!-- Horizontal stats ribbon -->
            <section class="ribbon" in:slide={{ duration: 700, delay: 600 }}>
                <div class="ribbon-item">
                    <span class="ribbon-value"
                        >{wrapped.total_plays.toLocaleString()}</span
                    >
                    <span class="ribbon-label">PLAYS</span>
                </div>
                <div class="ribbon-divider"></div>
                <div class="ribbon-item">
                    <span class="ribbon-value">{wrapped.days_listened}</span>
                    <span class="ribbon-label">DAYS</span>
                </div>
                <div class="ribbon-divider"></div>
                <div class="ribbon-item">
                    <span class="ribbon-value">{wrapped.unique_songs}</span>
                    <span class="ribbon-label">TRACKS</span>
                </div>
                <div class="ribbon-divider"></div>
                <div class="ribbon-item">
                    <span class="ribbon-value">{wrapped.unique_artists}</span>
                    <span class="ribbon-label">ARTISTS</span>
                </div>
            </section>

            <!-- Stacked horizontal cards for top items -->
            <section class="cards-section">
                {#if wrapped.top_songs?.length > 0}
                    <div
                        class="card-group"
                        in:fly={{
                            x: -100,
                            duration: 800,
                            delay: 900,
                            easing: cubicOut,
                        }}
                    >
                        <div class="card-header">
                            <span class="card-index">01</span>
                            <span class="card-title">TOP TRACKS</span>
                        </div>
                        <div class="card-stack">
                            {#each wrapped.top_songs.slice(0, 5) as song, i}
                                <div
                                    class="card"
                                    style="--i: {i}"
                                    in:fly={{
                                        y: 30,
                                        duration: 500,
                                        delay: 1000 + i * 80,
                                    }}
                                >
                                    <span class="card-rank">#{i + 1}</span>
                                    <div class="card-content">
                                        <span class="card-name"
                                            >{song.title}</span
                                        >
                                        <span class="card-sub"
                                            >{song.artist?.name ||
                                                "Unknown Artist"}</span
                                        >
                                    </div>
                                    <span class="card-plays">{song.plays}</span>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}

                {#if wrapped.top_artists?.length > 0}
                    <div
                        class="card-group"
                        in:fly={{
                            x: -100,
                            duration: 800,
                            delay: 1100,
                            easing: cubicOut,
                        }}
                    >
                        <div class="card-header">
                            <span class="card-index">02</span>
                            <span class="card-title">TOP ARTISTS</span>
                        </div>
                        <div class="card-stack">
                            {#each wrapped.top_artists.slice(0, 5) as artist, i}
                                <div
                                    class="card card--artist"
                                    style="--i: {i}"
                                    in:fly={{
                                        y: 30,
                                        duration: 500,
                                        delay: 1200 + i * 80,
                                    }}
                                >
                                    <span class="card-rank">#{i + 1}</span>
                                    <div class="card-content">
                                        <span class="card-name"
                                            >{artist.name}</span
                                        >
                                    </div>
                                    <span class="card-plays"
                                        >{artist.plays}</span
                                    >
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}

                {#if wrapped.top_albums?.length > 0}
                    <div
                        class="card-group"
                        in:fly={{
                            x: -100,
                            duration: 800,
                            delay: 1300,
                            easing: cubicOut,
                        }}
                    >
                        <div class="card-header">
                            <span class="card-index">03</span>
                            <span class="card-title">TOP ALBUMS</span>
                        </div>
                        <div class="card-stack">
                            {#each wrapped.top_albums.slice(0, 5) as album, i}
                                <div
                                    class="card"
                                    style="--i: {i}"
                                    in:fly={{
                                        y: 30,
                                        duration: 500,
                                        delay: 1400 + i * 80,
                                    }}
                                >
                                    <span class="card-rank">#{i + 1}</span>
                                    <div class="card-content">
                                        <span class="card-name"
                                            >{album.title}</span
                                        >
                                        <span class="card-sub"
                                            >{album.artist?.name ||
                                                "Unknown Artist"}</span
                                        >
                                    </div>
                                    <span class="card-plays">{album.plays}</span
                                    >
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}
            </section>

            <!-- Personality badge -->
            {#if wrapped.personality}
                <div
                    class="personality-badge"
                    in:fly={{
                        y: 40,
                        duration: 700,
                        delay: 1600,
                        easing: elasticOut,
                    }}
                >
                    <span class="personality-icon">&#9835;</span>
                    <span class="personality-text">{wrapped.personality}</span>
                </div>
            {/if}
        </main>

        <!-- Right decorative edge -->
        <aside class="edge" in:fade={{ duration: 1000, delay: 400 }}>
            <div class="edge-pattern">
                {#each Array(12) as _, i}
                    <div class="edge-bar" style="--delay: {i * 0.1}s"></div>
                {/each}
            </div>
        </aside>
    </div>
</div>

<style>
    @import url("https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=Syne:wght@700;800&display=swap");

    .wrapped {
        min-height: 100vh;
        background: #050505;
        position: relative;
        overflow: hidden;
        font-family: "Space Grotesk", system-ui, sans-serif;
    }

    /* Scanlines effect */
    .scanlines {
        position: absolute;
        inset: 0;
        background: repeating-linear-gradient(
            0deg,
            transparent,
            transparent 2px,
            rgba(0, 0, 0, 0.15) 2px,
            rgba(0, 0, 0, 0.15) 4px
        );
        pointer-events: none;
        z-index: 100;
    }

    /* Mesh gradient background */
    .mesh-bg {
        position: absolute;
        inset: 0;
        overflow: hidden;
    }

    .mesh-layer {
        position: absolute;
        width: 150%;
        height: 150%;
        background-size: 100% 100%;
        animation: meshMove 20s ease-in-out infinite;
    }

    .mesh-1 {
        background: radial-gradient(
            ellipse at 20% 30%,
            #ff2d55 0%,
            transparent 50%
        );
        opacity: 0.4;
        animation-delay: 0s;
    }

    .mesh-2 {
        background: radial-gradient(
            ellipse at 80% 60%,
            #5856d6 0%,
            transparent 50%
        );
        opacity: 0.5;
        animation-delay: -7s;
    }

    .mesh-3 {
        background: radial-gradient(
            ellipse at 50% 90%,
            #ff9500 0%,
            transparent 40%
        );
        opacity: 0.3;
        animation-delay: -14s;
    }

    @keyframes meshMove {
        0%,
        100% {
            transform: translate(0, 0) rotate(0deg);
        }
        25% {
            transform: translate(-5%, 3%) rotate(1deg);
        }
        50% {
            transform: translate(3%, -5%) rotate(-1deg);
        }
        75% {
            transform: translate(-3%, -3%) rotate(0.5deg);
        }
    }

    /* Noise texture */
    .noise {
        position: absolute;
        inset: 0;
        background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E");
        opacity: 0.03;
        pointer-events: none;
        z-index: 50;
    }

    /* Viewport layout */
    .viewport {
        position: relative;
        z-index: 10;
        min-height: 100vh;
        display: grid;
        grid-template-columns: 80px 1fr 60px;
    }

    /* Left sidebar */
    .sidebar {
        background: linear-gradient(
            180deg,
            rgba(255, 45, 85, 0.15) 0%,
            rgba(88, 86, 214, 0.1) 100%
        );
        border-right: 1px solid rgba(255, 255, 255, 0.08);
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 2rem 0;
        gap: 1.5rem;
    }

    .brand {
        writing-mode: vertical-rl;
        text-orientation: mixed;
        display: flex;
        gap: 0.5rem;
    }

    .brand-year {
        font-family: "Syne", sans-serif;
        font-size: 1.5rem;
        font-weight: 800;
        color: #fff;
        letter-spacing: -0.02em;
    }

    .brand-month {
        font-size: 0.875rem;
        font-weight: 500;
        color: #ff2d55;
        letter-spacing: 0.1em;
    }

    .brand-line {
        width: 2px;
        height: 60px;
        background: linear-gradient(180deg, #ff2d55, #5856d6);
    }

    .brand-label {
        writing-mode: vertical-rl;
        font-size: 0.625rem;
        letter-spacing: 0.3em;
        color: rgba(255, 255, 255, 0.4);
        text-transform: uppercase;
    }

    /* Main content */
    .main {
        padding: 3rem 4rem;
        display: flex;
        flex-direction: column;
        gap: 3rem;
    }

    /* Hero section */
    .hero {
        display: flex;
        align-items: flex-start;
    }

    .hero-stat {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .hero-digits {
        display: flex;
        line-height: 1;
    }

    .digit {
        font-family: "Syne", sans-serif;
        font-size: clamp(5rem, 18vw, 14rem);
        font-weight: 800;
        color: transparent;
        background: linear-gradient(
            135deg,
            #ff2d55 0%,
            #ff9500 40%,
            #5856d6 100%
        );
        -webkit-background-clip: text;
        background-clip: text;
        animation: digitPulse 3s ease-in-out infinite;
        animation-delay: var(--delay, 0s);
    }

    @keyframes digitPulse {
        0%,
        100% {
            filter: brightness(1);
        }
        50% {
            filter: brightness(1.2);
        }
    }

    .hero-label {
        display: flex;
        align-items: baseline;
        gap: 1rem;
        padding-left: 0.5rem;
    }

    .hero-unit {
        font-size: 1.5rem;
        font-weight: 700;
        color: #fff;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .hero-desc {
        font-size: 1rem;
        color: rgba(255, 255, 255, 0.5);
        font-style: italic;
    }

    /* Stats ribbon */
    .ribbon {
        display: flex;
        align-items: center;
        gap: 2rem;
        padding: 1.5rem 2rem;
        background: rgba(255, 255, 255, 0.03);
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 2px;
        backdrop-filter: blur(10px);
    }

    .ribbon-item {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .ribbon-value {
        font-family: "Syne", sans-serif;
        font-size: 1.75rem;
        font-weight: 700;
        color: #fff;
    }

    .ribbon-label {
        font-size: 0.625rem;
        letter-spacing: 0.2em;
        color: rgba(255, 255, 255, 0.4);
    }

    .ribbon-divider {
        width: 1px;
        height: 40px;
        background: linear-gradient(
            180deg,
            transparent,
            rgba(255, 255, 255, 0.2),
            transparent
        );
    }

    /* Cards section */
    .cards-section {
        display: flex;
        flex-direction: column;
        gap: 2.5rem;
    }

    .card-group {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .card-header {
        display: flex;
        align-items: center;
        gap: 1rem;
    }

    .card-index {
        font-family: "Syne", sans-serif;
        font-size: 0.875rem;
        font-weight: 700;
        color: #ff2d55;
    }

    .card-title {
        font-size: 0.75rem;
        letter-spacing: 0.15em;
        color: rgba(255, 255, 255, 0.6);
    }

    .card-stack {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    .card {
        display: flex;
        align-items: center;
        gap: 1.25rem;
        padding: 1rem 1.5rem;
        background: linear-gradient(
            90deg,
            rgba(255, 45, 85, calc(0.08 - var(--i) * 0.01)) 0%,
            rgba(88, 86, 214, calc(0.05 - var(--i) * 0.008)) 100%
        );
        border-left: 3px solid;
        border-image: linear-gradient(180deg, #ff2d55, #5856d6) 1;
        transition:
            transform 0.3s ease,
            background 0.3s ease;
    }

    .card:hover {
        transform: translateX(8px);
        background: linear-gradient(
            90deg,
            rgba(255, 45, 85, 0.15) 0%,
            rgba(88, 86, 214, 0.1) 100%
        );
    }

    .card-rank {
        font-family: "Syne", sans-serif;
        font-size: 0.875rem;
        font-weight: 700;
        color: rgba(255, 255, 255, 0.3);
        min-width: 2.5rem;
    }

    .card-content {
        flex: 1;
        min-width: 0;
        display: flex;
        flex-direction: column;
        gap: 0.125rem;
    }

    .card-name {
        font-size: 1rem;
        font-weight: 600;
        color: #fff;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .card-sub {
        font-size: 0.8125rem;
        color: rgba(255, 255, 255, 0.45);
    }

    .card-plays {
        font-family: "Syne", sans-serif;
        font-size: 0.875rem;
        font-weight: 500;
        color: rgba(255, 255, 255, 0.5);
        padding: 0.25rem 0.75rem;
        background: rgba(255, 255, 255, 0.05);
        border-radius: 2px;
    }

    /* Personality badge */
    .personality-badge {
        display: inline-flex;
        align-items: center;
        gap: 0.75rem;
        padding: 1rem 1.75rem;
        background: linear-gradient(
            135deg,
            rgba(255, 45, 85, 0.2),
            rgba(88, 86, 214, 0.2)
        );
        border: 1px solid rgba(255, 255, 255, 0.12);
        border-radius: 100px;
        align-self: flex-start;
    }

    .personality-icon {
        font-size: 1.25rem;
        color: #ff9500;
    }

    .personality-text {
        font-size: 1rem;
        font-weight: 500;
        color: #fff;
        letter-spacing: 0.02em;
    }

    /* Right edge decoration */
    .edge {
        display: flex;
        align-items: center;
        justify-content: center;
        border-left: 1px solid rgba(255, 255, 255, 0.05);
    }

    .edge-pattern {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .edge-bar {
        width: 20px;
        height: 3px;
        background: linear-gradient(90deg, #ff2d55, transparent);
        opacity: 0.4;
        animation: edgePulse 2s ease-in-out infinite;
        animation-delay: var(--delay);
    }

    @keyframes edgePulse {
        0%,
        100% {
            opacity: 0.2;
            transform: scaleX(0.6);
        }
        50% {
            opacity: 0.6;
            transform: scaleX(1);
        }
    }

    /* Responsive */
    @media (max-width: 900px) {
        .viewport {
            grid-template-columns: 60px 1fr 0;
        }

        .main {
            padding: 2rem;
        }

        .edge {
            display: none;
        }

        .ribbon {
            flex-wrap: wrap;
            gap: 1.5rem;
        }

        .ribbon-divider {
            display: none;
        }
    }

    @media (max-width: 600px) {
        .viewport {
            grid-template-columns: 1fr;
        }

        .sidebar {
            flex-direction: row;
            writing-mode: horizontal-tb;
            border-right: none;
            border-bottom: 1px solid rgba(255, 255, 255, 0.08);
            padding: 1rem 2rem;
        }

        .brand {
            writing-mode: horizontal-tb;
            flex-direction: row;
        }

        .brand-line {
            width: 40px;
            height: 2px;
        }

        .brand-label {
            writing-mode: horizontal-tb;
        }

        .main {
            padding: 1.5rem;
        }

        .card {
            padding: 0.875rem 1rem;
        }
    }
</style>
