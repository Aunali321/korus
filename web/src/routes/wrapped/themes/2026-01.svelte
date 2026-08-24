<script lang="ts">
    import { fly, fade, slide } from "svelte/transition";
    import { cubicOut, expoOut } from "svelte/easing";

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

    function formatMinutes(mins: number): string {
        if (!mins || isNaN(mins)) return "0";
        const hours = Math.floor(mins / 60);
        return hours > 0 ? hours.toLocaleString() : String(mins);
    }

    function getUnit(mins: number): string {
        const hours = Math.floor(mins / 60);
        return hours > 0 ? "hours" : "minutes";
    }

    function splitChars(text: string): string[] {
        return text.split("");
    }
</script>

<div class="wrapped">
    <div class="grain"></div>
    <div class="scanlines"></div>
    
    <!-- Horizontal accent lines -->
    <div class="line-accent line-1"></div>
    <div class="line-accent line-2"></div>
    <div class="line-accent line-3"></div>

    <!-- Opening Title -->
    <section class="section section-intro">
        <div class="intro-content" in:fade={{ duration: 1200, delay: 300 }}>
            <span class="eyebrow">A Korus Production</span>
            <div class="year-display">
                <span class="year-prefix">20</span>
                <span class="year-main">24</span>
            </div>
            <span class="presents">presents</span>
        </div>
    </section>

    <!-- Hero: The Big Number -->
    <section class="section section-hero">
        <div class="hero-split">
            <div class="hero-left" in:fly={{ x: -100, duration: 1000, delay: 600, easing: expoOut }}>
                <div class="stat-vertical">
                    <span class="stat-num">{wrapped.days_listened}</span>
                    <span class="stat-word">days</span>
                </div>
                <div class="divider"></div>
                <div class="stat-vertical">
                    <span class="stat-num">{wrapped.total_plays.toLocaleString()}</span>
                    <span class="stat-word">plays</span>
                </div>
            </div>
            <div class="hero-right" in:fly={{ x: 100, duration: 1000, delay: 600, easing: expoOut }}>
                <div class="time-block">
                    <div class="time-number">
                        {#each splitChars(formatMinutes(wrapped.total_minutes)) as char, i}
                            <span 
                                class="char" 
                                in:fly={{ y: 80, duration: 800, delay: 800 + (i * 80), easing: cubicOut }}
                            >{char}</span>
                        {/each}
                    </div>
                    <div class="time-label" in:slide={{ duration: 600, delay: 1400 }}>
                        {getUnit(wrapped.total_minutes)} of sound
                    </div>
                </div>
            </div>
        </div>
    </section>

    <!-- Featured: Top Song -->
    {#if wrapped.top_song}
        <section class="section section-feature">
            <div class="feature-layout">
                <div class="feature-label" in:fly={{ x: -50, duration: 800, delay: 200, easing: expoOut }}>
                    <span class="label-line"></span>
                    <span class="label-text">Most Played Track</span>
                </div>
                <div class="feature-content" in:fade={{ duration: 1000, delay: 400 }}>
                    <h2 class="feature-title">{wrapped.top_song.title}</h2>
                    <p class="feature-subtitle">{wrapped.top_song.artist?.name || "Unknown Artist"}</p>
                </div>
            </div>
        </section>
    {/if}

    <!-- Artist & Album Grid -->
    <section class="section section-grid">
        <div class="grid-container">
            {#if wrapped.top_artist}
                <div class="grid-item grid-artist" in:fly={{ y: 60, duration: 800, delay: 300, easing: expoOut }}>
                    <div class="grid-header">
                        <span class="grid-number">01</span>
                        <span class="grid-label">Artist</span>
                    </div>
                    <h3 class="grid-value">{wrapped.top_artist.name}</h3>
                </div>
            {/if}
            {#if wrapped.top_album}
                <div class="grid-item grid-album" in:fly={{ y: 60, duration: 800, delay: 450, easing: expoOut }}>
                    <div class="grid-header">
                        <span class="grid-number">02</span>
                        <span class="grid-label">Album</span>
                    </div>
                    <h3 class="grid-value">{wrapped.top_album.title}</h3>
                    <p class="grid-sub">{wrapped.top_album.artist?.name || ""}</p>
                </div>
            {/if}
        </div>
    </section>

    <!-- Stats Ticker -->
    <section class="section section-stats">
        <div class="ticker" in:slide={{ duration: 800, delay: 200, axis: 'x' }}>
            <div class="ticker-item">
                <span class="ticker-value">{wrapped.unique_songs}</span>
                <span class="ticker-label">unique tracks</span>
            </div>
            <span class="ticker-sep">/</span>
            <div class="ticker-item">
                <span class="ticker-value">{wrapped.unique_artists}</span>
                <span class="ticker-label">artists discovered</span>
            </div>
            <span class="ticker-sep">/</span>
            <div class="ticker-item">
                <span class="ticker-value">{wrapped.avg_plays_per_day.toFixed(1)}</span>
                <span class="ticker-label">plays per day</span>
            </div>
        </div>
    </section>

    <!-- Personality Card -->
    {#if wrapped.personality}
        <section class="section section-personality">
            <div class="personality-card" in:fly={{ y: 40, duration: 800, delay: 300, easing: expoOut }}>
                <div class="personality-badge">Your Listening Identity</div>
                <div class="personality-text">{wrapped.personality}</div>
            </div>
        </section>
    {/if}

    <!-- Closing Credits -->
    <section class="section section-credits">
        <div class="credits-content" in:fade={{ duration: 1000, delay: 200 }}>
            <div class="credits-line"></div>
            <span class="credits-text">Your Year in Sound</span>
            <span class="credits-year">2024</span>
        </div>
    </section>
</div>

<style>
    @import url('https://fonts.googleapis.com/css2?family=Playfair+Display:wght@400;700;900&family=Space+Mono:wght@400;700&display=swap');

    :root {
        --color-bg: #0d0d0d;
        --color-cream: #f5f0e8;
        --color-gold: #c9a227;
        --color-muted: #6b6b6b;
        --font-display: 'Playfair Display', Georgia, serif;
        --font-mono: 'Space Mono', monospace;
    }

    .wrapped {
        min-height: 100vh;
        background: var(--color-bg);
        color: var(--color-cream);
        position: relative;
        overflow: hidden;
        font-family: var(--font-mono);
    }

    /* Film grain overlay */
    .grain {
        position: fixed;
        inset: 0;
        pointer-events: none;
        z-index: 100;
        opacity: 0.04;
        background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E");
    }

    /* Scanlines */
    .scanlines {
        position: fixed;
        inset: 0;
        pointer-events: none;
        z-index: 99;
        background: repeating-linear-gradient(
            0deg,
            transparent,
            transparent 2px,
            rgba(0, 0, 0, 0.03) 2px,
            rgba(0, 0, 0, 0.03) 4px
        );
    }

    /* Horizontal accent lines */
    .line-accent {
        position: fixed;
        left: 0;
        right: 0;
        height: 1px;
        background: linear-gradient(90deg, transparent, var(--color-gold), transparent);
        opacity: 0.2;
        z-index: 50;
    }

    .line-1 { top: 15%; }
    .line-2 { top: 50%; }
    .line-3 { top: 85%; }

    .section {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 4rem 2rem;
        position: relative;
    }

    /* Intro Section */
    .section-intro {
        flex-direction: column;
        gap: 1rem;
    }

    .intro-content {
        text-align: center;
    }

    .eyebrow {
        display: block;
        font-size: 0.7rem;
        letter-spacing: 0.4em;
        text-transform: uppercase;
        color: var(--color-muted);
        margin-bottom: 2rem;
    }

    .year-display {
        display: flex;
        justify-content: center;
        align-items: baseline;
        gap: 0.5rem;
    }

    .year-prefix {
        font-family: var(--font-display);
        font-size: 4rem;
        font-weight: 400;
        color: var(--color-muted);
    }

    .year-main {
        font-family: var(--font-display);
        font-size: 12rem;
        font-weight: 900;
        line-height: 0.85;
        background: linear-gradient(180deg, var(--color-cream) 0%, var(--color-gold) 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .presents {
        display: block;
        font-size: 0.9rem;
        letter-spacing: 0.3em;
        text-transform: uppercase;
        color: var(--color-gold);
        margin-top: 2rem;
    }

    /* Hero Section - Split Layout */
    .section-hero {
        padding: 0;
    }

    .hero-split {
        display: grid;
        grid-template-columns: 1fr 2fr;
        width: 100%;
        min-height: 100vh;
    }

    .hero-left {
        background: linear-gradient(135deg, #1a1a1a 0%, #0d0d0d 100%);
        display: flex;
        flex-direction: column;
        justify-content: center;
        padding: 4rem;
        border-right: 1px solid rgba(201, 162, 39, 0.2);
    }

    .stat-vertical {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .stat-num {
        font-size: 3.5rem;
        font-weight: 700;
        font-family: var(--font-display);
        color: var(--color-cream);
    }

    .stat-word {
        font-size: 0.7rem;
        letter-spacing: 0.3em;
        text-transform: uppercase;
        color: var(--color-muted);
    }

    .divider {
        width: 40px;
        height: 1px;
        background: var(--color-gold);
        margin: 2rem 0;
    }

    .hero-right {
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 4rem;
        background: radial-gradient(ellipse at center, rgba(201, 162, 39, 0.03) 0%, transparent 70%);
    }

    .time-block {
        text-align: center;
    }

    .time-number {
        display: flex;
        justify-content: center;
        overflow: hidden;
    }

    .char {
        font-family: var(--font-display);
        font-size: clamp(8rem, 22vw, 18rem);
        font-weight: 900;
        line-height: 0.9;
        color: var(--color-cream);
        display: inline-block;
    }

    .time-label {
        font-size: 1.1rem;
        letter-spacing: 0.2em;
        text-transform: uppercase;
        color: var(--color-gold);
        margin-top: 2rem;
    }

    /* Feature Section */
    .section-feature {
        background: linear-gradient(180deg, var(--color-bg) 0%, #111 50%, var(--color-bg) 100%);
    }

    .feature-layout {
        max-width: 900px;
        width: 100%;
    }

    .feature-label {
        display: flex;
        align-items: center;
        gap: 1rem;
        margin-bottom: 2rem;
    }

    .label-line {
        width: 60px;
        height: 1px;
        background: var(--color-gold);
    }

    .label-text {
        font-size: 0.7rem;
        letter-spacing: 0.3em;
        text-transform: uppercase;
        color: var(--color-gold);
    }

    .feature-content {
        padding-left: calc(60px + 1rem);
    }

    .feature-title {
        font-family: var(--font-display);
        font-size: clamp(3rem, 8vw, 6rem);
        font-weight: 700;
        line-height: 1.1;
        margin: 0;
        color: var(--color-cream);
    }

    .feature-subtitle {
        font-size: 1.2rem;
        color: var(--color-muted);
        margin-top: 1rem;
        letter-spacing: 0.1em;
    }

    /* Grid Section */
    .section-grid {
        min-height: 70vh;
    }

    .grid-container {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 1px;
        background: rgba(201, 162, 39, 0.2);
        max-width: 1000px;
        width: 100%;
    }

    .grid-item {
        background: var(--color-bg);
        padding: 4rem;
    }

    .grid-header {
        display: flex;
        align-items: baseline;
        gap: 1rem;
        margin-bottom: 1.5rem;
    }

    .grid-number {
        font-size: 0.7rem;
        color: var(--color-gold);
    }

    .grid-label {
        font-size: 0.65rem;
        letter-spacing: 0.3em;
        text-transform: uppercase;
        color: var(--color-muted);
    }

    .grid-value {
        font-family: var(--font-display);
        font-size: 2.5rem;
        font-weight: 700;
        margin: 0;
        line-height: 1.2;
    }

    .grid-sub {
        font-size: 0.9rem;
        color: var(--color-muted);
        margin-top: 0.5rem;
    }

    /* Stats Ticker */
    .section-stats {
        min-height: 30vh;
        background: #0a0a0a;
        border-top: 1px solid rgba(201, 162, 39, 0.1);
        border-bottom: 1px solid rgba(201, 162, 39, 0.1);
    }

    .ticker {
        display: flex;
        align-items: center;
        gap: 3rem;
        flex-wrap: wrap;
        justify-content: center;
    }

    .ticker-item {
        text-align: center;
    }

    .ticker-value {
        display: block;
        font-family: var(--font-display);
        font-size: 3rem;
        font-weight: 700;
    }

    .ticker-label {
        display: block;
        font-size: 0.65rem;
        letter-spacing: 0.2em;
        text-transform: uppercase;
        color: var(--color-muted);
        margin-top: 0.5rem;
    }

    .ticker-sep {
        font-size: 2rem;
        color: var(--color-gold);
        opacity: 0.5;
    }

    /* Personality Section */
    .section-personality {
        min-height: 60vh;
    }

    .personality-card {
        text-align: center;
        max-width: 600px;
        padding: 4rem;
        border: 1px solid rgba(201, 162, 39, 0.3);
        position: relative;
    }

    .personality-card::before,
    .personality-card::after {
        content: '';
        position: absolute;
        width: 20px;
        height: 20px;
        border: 1px solid var(--color-gold);
    }

    .personality-card::before {
        top: -1px;
        left: -1px;
        border-right: none;
        border-bottom: none;
    }

    .personality-card::after {
        bottom: -1px;
        right: -1px;
        border-left: none;
        border-top: none;
    }

    .personality-badge {
        font-size: 0.6rem;
        letter-spacing: 0.4em;
        text-transform: uppercase;
        color: var(--color-gold);
        margin-bottom: 2rem;
    }

    .personality-text {
        font-family: var(--font-display);
        font-size: 2rem;
        font-weight: 400;
        font-style: italic;
        line-height: 1.4;
    }

    /* Credits Section */
    .section-credits {
        min-height: 50vh;
    }

    .credits-content {
        text-align: center;
    }

    .credits-line {
        width: 1px;
        height: 80px;
        background: linear-gradient(180deg, transparent, var(--color-gold), transparent);
        margin: 0 auto 2rem;
    }

    .credits-text {
        display: block;
        font-size: 0.7rem;
        letter-spacing: 0.4em;
        text-transform: uppercase;
        color: var(--color-muted);
    }

    .credits-year {
        display: block;
        font-family: var(--font-display);
        font-size: 1.5rem;
        font-weight: 700;
        color: var(--color-gold);
        margin-top: 1rem;
    }

    /* Responsive */
    @media (max-width: 768px) {
        .hero-split {
            grid-template-columns: 1fr;
        }

        .hero-left {
            min-height: 40vh;
            border-right: none;
            border-bottom: 1px solid rgba(201, 162, 39, 0.2);
            flex-direction: row;
            gap: 2rem;
            align-items: center;
        }

        .divider {
            width: 1px;
            height: 60px;
            margin: 0;
        }

        .hero-right {
            min-height: 60vh;
        }

        .grid-container {
            grid-template-columns: 1fr;
        }

        .ticker {
            flex-direction: column;
            gap: 2rem;
        }

        .ticker-sep {
            display: none;
        }

        .feature-content {
            padding-left: 0;
        }

        .year-main {
            font-size: 8rem;
        }
    }
</style>
