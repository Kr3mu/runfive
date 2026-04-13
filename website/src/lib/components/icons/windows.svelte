<script lang="ts">
    interface Props {
        class?: string;
    }

    let { class: className = "" }: Props = $props();
</script>

<svg viewBox="0 0 24 24" fill="currentColor" class={className}>
    <path d="M0,0H11.377V11.372H0ZM12.623,0H24V11.372H12.623ZM0,12.623H11.377V24H0Zm12.623,0H24V24H12.623" />
</svg>
