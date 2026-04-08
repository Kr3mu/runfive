<script lang="ts">
    import "./app.css";
    import { QueryClientProvider } from "@tanstack/svelte-query";
    import { QueryClient } from "@tanstack/query-core";
    import { Router } from "sv-router";
    import "$lib/theme.svelte";
    import "sv-router/generated";

    const queryClient = new QueryClient({
        defaultOptions: {
            queries: {
                staleTime: 1000 * 5,
                retry: 1,
                refetchOnWindowFocus: true,
            },
        },
    });
</script>

<QueryClientProvider client={queryClient}>
    <Router />
</QueryClientProvider>
