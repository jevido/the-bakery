<script lang="ts">
  import { portrait } from '../portrait/portrait'

  // A member's or an agent's generated pixel portrait. The same seed always
  // draws the same face. With alt it is an image; without, decoration.
  let { seed, size = 48, alt = '' }: { seed: string; size?: number; alt?: string } = $props()

  let svg = $derived(portrait(seed, size))
</script>

{#if alt}
  <span class="portrait" style:width="{size}px" style:height="{size}px" role="img" aria-label={alt}>{@html svg}</span>
{:else}
  <span class="portrait" style:width="{size}px" style:height="{size}px" aria-hidden="true">{@html svg}</span>
{/if}

<style>
  .portrait {
    display: inline-block;
    flex-shrink: 0;
    line-height: 0;
    overflow: hidden;
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    image-rendering: pixelated;
  }

  .portrait :global(svg) {
    width: 100%;
    height: 100%;
  }
</style>
