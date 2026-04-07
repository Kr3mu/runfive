<script lang="ts">
    interface Props {
        class?: string;
    }

    let { class: className = "" }: Props = $props();
</script>

<svg viewBox="0 0 95 64" fill="currentColor" class={className}>
    <path d="M64.4992 0H52.0201L53.054 12.8535H41.6446L42.6785 0H30.3064L0 64H37.4908L38.7209 48.7933H55.9599L57.19 64H94.7877L64.4813 0H64.4992ZM39.5944 38.2039L41.2167 18.2017H53.4997L55.122 38.2039H39.5944Z"/>
</svg>
