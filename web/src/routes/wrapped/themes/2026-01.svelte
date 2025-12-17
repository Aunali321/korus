<script lang="ts">
    import { fade, fly, blur, scale } from "svelte/transition";
    import { quintOut, backOut, expoOut } from "svelte/easing";

    interface WrappedData {
        total_minutes: number;
        top_song?: { id: number; title: string; artist?: { name: string } };
        top_artist?: { id: number; name: string };
        top_album?: { id: number; title: string; artist?: { name: string } };
        total_plays: number;
        days_listened: number;
        avg_plays_per_day: number;
        unique_songs: number;
        unique_artists: number;
        personality?: string;
        milestones: string[];
    }

    let { wrapped }: { wrapped: WrappedData } = $props();

    function formatHours(mins: number): number {
        if (!mins || isNaN(mins)) return 0;
        return Math.floor(mins / 60);
    }

    function splitNumber(num: number): string[] {
        return num.toLocaleString().split("");
    }
</script>

<div class="wrapped">
    <!-- Layered background -->
    <div class="bg-layer bg-base"></div>
    <div class="bg-layer bg-mesh"></div>
    <div class="bg-layer bg-grain"></div>
    <div class="bg-layer bg-vignette"></div>
    
    <!-- Diagonal slice accent -->
    <div class="slice slice-top" in:fly={{ x: -200, duration: 1200, delay: 100, easing: expoOut }}></div>
    <div class="slice slice-bottom" in:fly={{ x: 200, duration: 1200, delay: 200, easing: expoOut }}></div>

    <!-- Content wrapper -->
    <div class="frame">
        <!-- Header badge -->
        <header class="header" in:fade={{ duration: 600, delay: 300 }}>
            <div class="header-line"></div>
            <span class="header-text">WRAPPED</span>
            <span class="header-year">JAN 2026</span>
            <div class="header-line"></div>
        </header>

        <!-- Main hero - massive stacked typography -->
        <section class="hero">
            <div class="hero-intro" in:blur={{ duration: 800, delay: 400, amount: 10 }}>
                YOU SPENT
            </div>
            <div class="hero-stack">
                {#each splitNumber(formatHours(wrapped.total_minutes)) as char, i}
                    <span 
                        class="hero-char" 
                        class:hero-comma={char === ","}
                        in:fly={{ y: 120, duration: 900, delay: 500 + i * 80, easing: backOut }}
                    >
                        {char}
                    </span>
                {/each}
            </div>
            <div class="hero-label" in:fly={{ y: 30, duration: 600, delay: 900, easing: quintOut }}>
                <span class="hero-label-text">HOURS</span>
                <span class="hero-label-sub">IMMERSED IN SOUND</span>
            </div>
        </section>

        <!-- Horizontal divider with stats -->
        <section class="divider-section" in:scale={{ start: 0.8, duration: 800, delay: 1000, easing: quintOut }}>
            <div class="divider-stat">
                <span class="divider-value">{wrapped.total_plays.toLocaleString()}</span>
                <span class="divider-label">PLAYS</span>
            </div>
            <div class="divider-line"></div>
            <div class="divider-stat">
                <span class="divider-value">{wrapped.days_listened}</span>
                <span class="divider-label">DAYS</span>
            </div>
            <div class="divider-line"></div>
            <div class="divider-stat">
                <span class="divider-value">{wrapped.unique_songs}</span>
                <span class="divider-label">TRACKS</span>
            </div>
            <div class="divider-line"></div>
            <div class="divider-stat">
                <span class="divider-value">{wrapped.unique_artists}</span>
                <span class="divider-label">ARTISTS</span>
            </div>
        </section>

        <!-- Top picks - editorial magazine style -->
        <section class="picks">
            {#if wrapped.top_song}
                <article class="pick" in:fly={{ x: -60, duration: 700, delay: 1200, easing: quintOut }}>
                    <div class="pick-index">01</div>
                    <div class="pick-body">
                        <span class="pick-category">TOP TRACK</span>
                        <h2 class="pick-title">{wrapped.top_song.title}</h2>
                        <span class="pick-artist">{wrapped.top_song.artist?.name || "Unknown Artist"}</span>
                    </div>
                    <div class="pick-accent"></div>
                </article>
            {/if}

            {#if wrapped.top_artist}
                <article class="pick" in:fly={{ x: -60, duration: 700, delay: 1350, easing: quintOut }}>
                    <div class="pick-index">02</div>
                    <div class="pick-body">
                        <span class="pick-category">TOP ARTIST</span>
                        <h2 class="pick-title">{wrapped.top_artist.name}</h2>
                    </div>
                    <div class="pick-accent"></div>
                </article>
            {/if}

            {#if wrapped.top_album}
                <article class="pick" in:fly={{ x: -60, duration: 700, delay: 1500, easing: quintOut }}>
                    <div class="pick-index">03</div>
                    <div class="pick-body">
                        <span class="pick-category">TOP ALBUM</span>
                        <h2 class="pick-title">{wrapped.top_album.title}</h2>
                        <span class="pick-artist">{wrapped.top_album.artist?.name || "Unknown Artist"}</span>
                    </div>
                    <div class="pick-accent"></div>
                </article>
            {/if}
        </section>

        <!-- Personality footer -->
        {#if wrapped.personality}
            <footer class="footer" in:fly={{ y: 40, duration: 600, delay: 1700, easing: quintOut }}>
                <div class="personality-wrapper">
                    <span class="personality-pre">YOUR SONIC IDENTITY</span>
                    <div class="personality-main">{wrapped.personality}</div>
                </div>
            </footer>
        {/if}
    </div>

    <!-- Corner accents -->
    <div class="corner corner-tl" in:scale={{ start: 0, duration: 800, delay: 200 }}></div>
    <div class="corner corner-tr" in:scale={{ start: 0, duration: 800, delay: 300 }}></div>
    <div class="corner corner-bl" in:scale={{ start: 0, duration: 800, delay: 400 }}></div>
    <div class="corner corner-br" in:scale={{ start: 0, duration: 800, delay: 500 }}></div>
</div>

<style>
    @import url('https://fonts.googleapis.com/css2?family=Playfair+Display:wght@400;700;900&family=Bebas+Neue&family=Libre+Baskerville:ital@0;1&display=swap');

    :root {
        --clr-bg: #0d0d0d;
        --clr-surface: #1a1a1a;
        --clr-accent: #c9a227;
        --clr-accent-dim: rgba(201, 162, 39, 0.15);
        --clr-text: #f5f0e6;
        --clr-text-muted: rgba(245, 240, 230, 0.5);
        --clr-text-dim: rgba(245, 240, 230, 0.25);
        --font-display: 'Bebas Neue', sans-serif;
        --font-serif: 'Playfair Display', serif;
        --font-body: 'Libre Baskerville', serif;
    }

    .wrapped {
        min-height: 100vh;
        background: var(--clr-bg);
        position: relative;
        overflow: hidden;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    /* Layered backgrounds */
    .bg-layer {
        position: absolute;
        inset: 0;
        pointer-events: none;
    }

    .bg-base {
        background: radial-gradient(ellipse 120% 80% at 50% 120%, #1f1a12 0%, var(--clr-bg) 70%);
    }

    .bg-mesh {
        background: 
            radial-gradient(circle at 20% 30%, rgba(201, 162, 39, 0.08) 0%, transparent 40%),
            radial-gradient(circle at 80% 70%, rgba(201, 162, 39, 0.05) 0%, transparent 35%);
    }

    .bg-grain {
        background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 512 512' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.8' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E");
        opacity: 0.035;
        mix-blend-mode: overlay;
    }

    .bg-vignette {
        background: radial-gradient(ellipse 70% 60% at 50% 50%, transparent 30%, rgba(0,0,0,0.6) 100%);
    }

    /* Diagonal slices */
    .slice {
        position: absolute;
        pointer-events: none;
    }

    .slice-top {
        top: 0;
        left: -10%;
        width: 50%;
        height: 35%;
        background: linear-gradient(135deg, var(--clr-accent-dim) 0%, transparent 60%);
        clip-path: polygon(0 0, 100% 0, 70% 100%, 0 80%);
    }

    .slice-bottom {
        bottom: 0;
        right: -10%;
        width: 45%;
        height: 40%;
        background: linear-gradient(-45deg, var(--clr-accent-dim) 0%, transparent 50%);
        clip-path: polygon(30% 0, 100% 20%, 100% 100%, 0 100%);
    }

    /* Frame wrapper */
    .frame {
        position: relative;
        z-index: 1;
        width: 100%;
        max-width: 1100px;
        padding: 4rem 3rem;
        display: flex;
        flex-direction: column;
        gap: 3.5rem;
    }

    /* Header */
    .header {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 1.5rem;
    }

    .header-line {
        flex: 1;
        max-width: 120px;
        height: 1px;
        background: linear-gradient(90deg, transparent, var(--clr-accent), transparent);
    }

    .header-text {
        font-family: var(--font-display);
        font-size: 0.875rem;
        letter-spacing: 0.5em;
        color: var(--clr-accent);
    }

    .header-year {
        font-family: var(--font-body);
        font-size: 0.75rem;
        font-style: italic;
        letter-spacing: 0.1em;
        color: var(--clr-text-muted);
    }

    /* Hero section */
    .hero {
        text-align: center;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5rem;
    }

    .hero-intro {
        font-family: var(--font-body);
        font-size: 0.875rem;
        font-style: italic;
        letter-spacing: 0.3em;
        color: var(--clr-text-muted);
    }

    .hero-stack {
        display: flex;
        align-items: baseline;
        justify-content: center;
        line-height: 0.85;
        overflow: hidden;
        padding: 0.5rem 0;
    }

    .hero-char {
        font-family: var(--font-serif);
        font-size: clamp(8rem, 22vw, 18rem);
        font-weight: 900;
        color: var(--clr-text);
        text-shadow: 
            0 0 80px rgba(201, 162, 39, 0.3),
            0 4px 0 rgba(0,0,0,0.3);
        display: inline-block;
    }

    .hero-comma {
        font-size: clamp(4rem, 12vw, 10rem);
        color: var(--clr-accent);
        margin: 0 0.1em;
    }

    .hero-label {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.25rem;
        margin-top: 0.5rem;
    }

    .hero-label-text {
        font-family: var(--font-display);
        font-size: 2rem;
        letter-spacing: 0.5em;
        color: var(--clr-accent);
        margin-left: 0.5em;
    }

    .hero-label-sub {
        font-family: var(--font-body);
        font-size: 0.75rem;
        font-style: italic;
        letter-spacing: 0.2em;
        color: var(--clr-text-dim);
    }

    /* Divider section with stats */
    .divider-section {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 2rem;
        padding: 2rem 0;
        border-top: 1px solid rgba(201, 162, 39, 0.2);
        border-bottom: 1px solid rgba(201, 162, 39, 0.2);
    }

    .divider-stat {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.25rem;
        min-width: 80px;
    }

    .divider-value {
        font-family: var(--font-serif);
        font-size: 1.75rem;
        font-weight: 700;
        color: var(--clr-text);
    }

    .divider-label {
        font-family: var(--font-display);
        font-size: 0.625rem;
        letter-spacing: 0.3em;
        color: var(--clr-text-muted);
    }

    .divider-line {
        width: 40px;
        height: 1px;
        background: linear-gradient(90deg, transparent, var(--clr-accent), transparent);
    }

    /* Picks section */
    .picks {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
    }

    .pick {
        display: flex;
        align-items: flex-start;
        gap: 1.5rem;
        padding: 1.5rem 2rem;
        background: linear-gradient(135deg, rgba(26, 26, 26, 0.9) 0%, rgba(26, 26, 26, 0.4) 100%);
        border: 1px solid rgba(201, 162, 39, 0.1);
        position: relative;
        overflow: hidden;
        transition: all 0.4s ease;
    }

    .pick:hover {
        border-color: rgba(201, 162, 39, 0.4);
        transform: translateX(8px);
    }

    .pick:hover .pick-accent {
        transform: scaleY(1);
    }

    .pick-index {
        font-family: var(--font-display);
        font-size: 3rem;
        line-height: 1;
        color: rgba(201, 162, 39, 0.25);
        min-width: 70px;
    }

    .pick-body {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .pick-category {
        font-family: var(--font-display);
        font-size: 0.625rem;
        letter-spacing: 0.4em;
        color: var(--clr-accent);
    }

    .pick-title {
        font-family: var(--font-serif);
        font-size: 1.5rem;
        font-weight: 700;
        color: var(--clr-text);
        margin: 0;
        line-height: 1.2;
    }

    .pick-artist {
        font-family: var(--font-body);
        font-size: 0.875rem;
        font-style: italic;
        color: var(--clr-text-muted);
    }

    .pick-accent {
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        width: 3px;
        background: var(--clr-accent);
        transform: scaleY(0);
        transform-origin: top;
        transition: transform 0.4s ease;
    }

    /* Footer personality */
    .footer {
        display: flex;
        justify-content: center;
        padding-top: 1rem;
    }

    .personality-wrapper {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.75rem;
        padding: 2rem 4rem;
        background: rgba(201, 162, 39, 0.08);
        border: 1px solid rgba(201, 162, 39, 0.25);
        position: relative;
    }

    .personality-wrapper::before,
    .personality-wrapper::after {
        content: '';
        position: absolute;
        width: 20px;
        height: 20px;
        border-color: var(--clr-accent);
        border-style: solid;
    }

    .personality-wrapper::before {
        top: -1px;
        left: -1px;
        border-width: 2px 0 0 2px;
    }

    .personality-wrapper::after {
        bottom: -1px;
        right: -1px;
        border-width: 0 2px 2px 0;
    }

    .personality-pre {
        font-family: var(--font-display);
        font-size: 0.625rem;
        letter-spacing: 0.4em;
        color: var(--clr-text-muted);
    }

    .personality-main {
        font-family: var(--font-serif);
        font-size: 1.5rem;
        font-weight: 700;
        color: var(--clr-accent);
        text-transform: uppercase;
        letter-spacing: 0.1em;
        text-align: center;
    }

    /* Corner accents */
    .corner {
        position: absolute;
        width: 60px;
        height: 60px;
        border-color: rgba(201, 162, 39, 0.3);
        border-style: solid;
        pointer-events: none;
        z-index: 2;
    }

    .corner-tl {
        top: 2rem;
        left: 2rem;
        border-width: 1px 0 0 1px;
    }

    .corner-tr {
        top: 2rem;
        right: 2rem;
        border-width: 1px 1px 0 0;
    }

    .corner-bl {
        bottom: 2rem;
        left: 2rem;
        border-width: 0 0 1px 1px;
    }

    .corner-br {
        bottom: 2rem;
        right: 2rem;
        border-width: 0 1px 1px 0;
    }

    /* Responsive */
    @media (max-width: 768px) {
        .frame {
            padding: 3rem 1.5rem;
            gap: 2.5rem;
        }

        .header {
            flex-wrap: wrap;
            gap: 0.75rem;
        }

        .header-line {
            display: none;
        }

        .hero-char {
            font-size: clamp(5rem, 18vw, 10rem);
        }

        .hero-comma {
            font-size: clamp(3rem, 10vw, 6rem);
        }

        .divider-section {
            flex-wrap: wrap;
            gap: 1.5rem;
        }

        .divider-line {
            display: none;
        }

        .divider-stat {
            min-width: 70px;
        }

        .pick {
            padding: 1.25rem 1.5rem;
            gap: 1rem;
        }

        .pick-index {
            font-size: 2rem;
            min-width: 45px;
        }

        .pick-title {
            font-size: 1.25rem;
        }

        .personality-wrapper {
            padding: 1.5rem 2rem;
        }

        .personality-main {
            font-size: 1.25rem;
        }

        .corner {
            width: 40px;
            height: 40px;
        }

        .corner-tl, .corner-bl {
            left: 1rem;
        }

        .corner-tr, .corner-br {
            right: 1rem;
        }

        .corner-tl, .corner-tr {
            top: 1rem;
        }

        .corner-bl, .corner-br {
            bottom: 1rem;
        }

        .slice-top, .slice-bottom {
            opacity: 0.5;
        }
    }

    @media (max-width: 480px) {
        .frame {
            padding: 2.5rem 1rem;
            gap: 2rem;
        }

        .hero-intro {
            font-size: 0.75rem;
            letter-spacing: 0.2em;
        }

        .hero-label-text {
            font-size: 1.5rem;
            letter-spacing: 0.3em;
        }

        .divider-value {
            font-size: 1.5rem;
        }

        .pick-index {
            display: none;
        }

        .personality-main {
            font-size: 1.125rem;
        }
    }
</style>
