<script lang="ts">
  import type { Snippet } from 'svelte'
  import type { HTMLButtonAttributes } from 'svelte/elements'

  type Variant = 'plain' | 'confirm' | 'danger'

  let {
    variant = 'plain',
    children,
    ...rest
  }: HTMLButtonAttributes & { variant?: Variant; children: Snippet } = $props()
</script>

<button class={['button', variant]} {...rest}>
  {@render children()}
</button>

<style>
  .button {
    font-family: var(--font-display);
    font-size: 14px;
    color: var(--text);
    padding: 4px 12px;
    background: var(--panel-raised);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    box-shadow: inset 0 1px 0 #ffffff14, var(--shadow-outer);
    cursor: pointer;
  }

  .button:hover:not(:disabled) {
    background: var(--panel-title);
    border-color: var(--text-dim);
  }

  .button:active:not(:disabled) {
    box-shadow: var(--shadow-inner);
  }

  .button:disabled {
    color: var(--text-faint);
    border-color: var(--frame-dim);
    cursor: default;
  }

  .confirm {
    background: var(--olive);
  }

  .confirm:hover:not(:disabled) {
    background: var(--olive-bright);
  }

  .danger {
    background: var(--rust);
  }

  .danger:hover:not(:disabled) {
    background: var(--rust-bright);
  }
</style>
