<script lang="ts">
    import { theme } from "$lib/theme.svelte";

    const logoSrc = $derived(
        theme.value === "dark" ? "/logo.webp" : "/logo-light.webp",
    );
</script>

<main class="relative flex min-h-svh items-center justify-center overflow-hidden bg-background">
    <div class="speed-line speed-line-1"></div>
    <div class="speed-line speed-line-2"></div>
    <div class="speed-line speed-line-3"></div>
    <div class="speed-line speed-line-4"></div>
    <div class="speed-line speed-line-5"></div>

    <div class="relative z-10 flex flex-col items-center">
        <img
            src={logoSrc}
            alt="runfive"
            class="logo-entrance mb-8 w-72 md:w-96"
        />

        <div class="mb-6">
            <h1
                class="four-oh-four select-none font-heading text-[10rem] leading-none font-black tracking-tighter text-primary/10 md:text-[14rem]"
            >
                <span class="inline-block">4</span><span class="inline-block mx-2 md:mx-3">0</span><span class="inline-block">4</span>
            </h1>
        </div>

        <p
            class="overlay-text mb-5 font-heading text-lg font-semibold tracking-widest text-muted-foreground uppercase"
        >
            Wrong turn
        </p>

        <a
            href="/"
            class="go-back group flex items-center gap-2 rounded-md border border-border bg-card px-5 py-2.5 text-sm font-medium text-foreground transition-all hover:border-primary hover:text-primary"
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
            Head back
        </a>
    </div>
</main>

<style>
    .speed-line {
        position: absolute;
        height: 1px;
        background: linear-gradient(90deg, transparent, var(--primary), transparent);
        opacity: 0;
        animation: streak 3s ease-in-out infinite;
    }

    .speed-line-1 {
        top: 20%;
        width: 40%;
        left: -10%;
        animation-delay: 0s;
        opacity: 0.15;
    }

    .speed-line-2 {
        top: 35%;
        width: 60%;
        right: -20%;
        left: auto;
        animation-delay: 0.8s;
        opacity: 0.1;
    }

    .speed-line-3 {
        top: 55%;
        width: 35%;
        left: 5%;
        animation-delay: 1.6s;
        opacity: 0.12;
    }

    .speed-line-4 {
        top: 70%;
        width: 50%;
        right: -15%;
        left: auto;
        animation-delay: 2.2s;
        opacity: 0.08;
    }

    .speed-line-5 {
        top: 45%;
        width: 25%;
        left: 30%;
        animation-delay: 0.4s;
        opacity: 0.1;
    }

    @keyframes streak {
        0% {
            transform: translateX(-100%);
            opacity: 0;
        }
        30% {
            opacity: var(--streak-opacity, 0.12);
        }
        100% {
            transform: translateX(200vw);
            opacity: 0;
        }
    }

    .logo-entrance {
        animation: slide-in 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
    }

    .four-oh-four {
        animation: scale-in 0.5s cubic-bezier(0.16, 1, 0.3, 1) 0.15s both;
    }

    .overlay-text {
        animation: fade-up 0.4s ease-out 0.35s both;
    }

    .go-back {
        animation: fade-up 0.4s ease-out 0.5s both;
    }

    @keyframes slide-in {
        from {
            opacity: 0;
            transform: translateX(-30px);
        }
        to {
            opacity: 1;
            transform: translateX(0);
        }
    }

    @keyframes scale-in {
        from {
            opacity: 0;
            transform: scale(0.8);
        }
        to {
            opacity: 1;
            transform: scale(1);
        }
    }

    @keyframes fade-up {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
    }

    .go-back {
        animation-name: btn-fade;
        animation-fill-mode: both;
    }

    @keyframes btn-fade {
        from {
            opacity: 0;
            transform: translateY(10px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
</style>
