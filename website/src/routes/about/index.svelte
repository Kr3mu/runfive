<script lang="ts">
    import Logo from "$lib/components/logo.svelte";
    import Github from "$lib/components/icons/github.svelte";
    import Discord from "$lib/components/icons/discord.svelte";
</script>

<main class="flex flex-1 flex-col bg-background">
    <div class="mx-auto w-full max-w-2xl px-6 py-20 md:py-32">

        <Logo class="about-reveal mx-auto mb-16 w-56" />

        <blockquote class="about-reveal mb-16 text-center delay-1">
            <p class="font-heading text-3xl leading-snug font-medium tracking-tight text-foreground md:text-4xl">
                Your server, your rules.
            </p>
            <p class="mt-2 font-heading text-3xl leading-snug font-medium tracking-tight text-primary md:text-4xl">
                No black boxes.
            </p>
        </blockquote>

        <div class="about-reveal mb-12 space-y-5 delay-2">
            <p class="text-base leading-relaxed text-muted-foreground">
                We got tired of managing FiveM servers with tools we couldn't
                read, couldn't fix, and couldn't trust. Closed source panels
                that break on updates, phone home without asking, and lock you
                into workflows that don't fit how you actually work.
            </p>
            <p class="text-base leading-relaxed text-muted-foreground">
                runfive is the alternative we wished existed. One binary.
                Everything included — deployment, monitoring, management.
                Open source from the first commit. No telemetry, no premium
                tiers, no surprises.
            </p>
        </div>

        <div class="about-reveal mx-auto mb-12 h-px w-12 bg-primary/30 delay-3"></div>

        <section class="about-reveal mb-12 delay-3">
            <p class="mb-5 text-center text-xs tracking-widest text-muted-foreground/40 uppercase">
                Built by
            </p>
            <div class="flex items-center justify-center gap-8">
                <a
                    href="https://github.com/Kr3mu"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="group flex items-center gap-2 text-foreground transition-colors hover:text-primary"
                >
                    <Github class="h-4 w-4 opacity-40 transition-opacity group-hover:opacity-100" />
                    <span class="font-heading text-sm font-medium">Kr3mu</span>
                </a>
                <a
                    href="https://github.com/simpleC0de"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="group flex items-center gap-2 text-foreground transition-colors hover:text-primary"
                >
                    <Github class="h-4 w-4 opacity-40 transition-opacity group-hover:opacity-100" />
                    <span class="font-heading text-sm font-medium">lananal</span>
                </a>
            </div>
        </section>

        <div class="about-reveal flex items-center justify-center gap-5 delay-4">
            <a
                href="https://github.com/Kr3mu/runfive"
                target="_blank"
                rel="noopener noreferrer"
                class="text-muted-foreground/40 transition-colors hover:text-foreground"
                aria-label="GitHub"
            >
                <Github class="h-5 w-5" />
            </a>
            <!-- TODO: replace with actual Discord invite link -->
            <a
                href="https://discord.gg"
                target="_blank"
                rel="noopener noreferrer"
                class="text-muted-foreground/40 transition-colors hover:text-foreground"
                aria-label="Discord"
            >
                <Discord class="h-5 w-5" />
            </a>
        </div>

    </div>
</main>

<style>
    .about-reveal {
        animation: reveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
    }

    .delay-1 { animation-delay: 0.1s; }
    .delay-2 { animation-delay: 0.2s; }
    .delay-3 { animation-delay: 0.3s; }
    .delay-4 { animation-delay: 0.4s; }

    @keyframes reveal {
        from {
            opacity: 0;
            transform: translateY(12px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
</style>
