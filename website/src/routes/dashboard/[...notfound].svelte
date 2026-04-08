<script lang="ts">
    import Logo from '$lib/components/logo.svelte';
</script>

<div class="flex h-full items-center justify-center">
    <div class="relative z-10 flex flex-col items-center">
        <Logo class="logo-entrance mb-8 w-48" />

        <div class="mb-4">
            <h1
                class="four-oh-four select-none font-heading text-[8rem] leading-none font-black tracking-tighter text-primary/10"
            >
                <span class="inline-block">4</span>
                <span class="mx-2 inline-block">0</span>
                <span class="inline-block">4</span>
            </h1>
        </div>

        <p
            class="overlay-text mb-5 font-heading text-sm font-semibold tracking-widest text-muted-foreground uppercase"
        >
            Page not found
        </p>

        <a
            href="/dashboard"
            data-view-transition
            class="go-back group flex items-center gap-2 rounded-md border border-border bg-card px-4 py-2 text-sm font-medium text-foreground transition-all hover:border-primary hover:text-primary"
        >
            <svg
                class="h-4 w-4 transition-transform group-hover:-translate-x-1"
                viewBox="0 0 16 16"
                fill="none"
            >
                <path
                    d="M10 12L6 8L10 4"
                    stroke="currentColor"
                    stroke-width="1.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                />
            </svg>
            Back to Dashboard
        </a>
    </div>
</div>

<style>
    :global(.logo-entrance) {
        animation: slide-in 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
    }

    .four-oh-four {
        animation: scale-in 0.5s cubic-bezier(0.16, 1, 0.3, 1) 0.15s both;
    }

    .overlay-text {
        animation: fade-up 0.4s ease-out 0.35s both;
    }

    .go-back {
        animation: btn-fade 0.4s ease-out 0.5s both;
    }

    @keyframes slide-in {
        from { opacity: 0; transform: translateX(-30px); }
        to { opacity: 1; transform: translateX(0); }
    }

    @keyframes scale-in {
        from { opacity: 0; transform: scale(0.8); }
        to { opacity: 1; transform: scale(1); }
    }

    @keyframes fade-up {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    @keyframes btn-fade {
        from { opacity: 0; transform: translateY(10px); }
        to { opacity: 1; transform: translateY(0); }
    }
</style>
