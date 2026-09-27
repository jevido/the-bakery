<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements'

  let {
    label,
    value = $bindable(''),
    ...rest
  }: HTMLInputAttributes & { label?: string; value?: string } = $props()
</script>

<label class="field">
  {#if label}<span>{label}</span>{/if}
  <input bind:value {...rest} />
</label>

<style>
  .field {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  span {
    font-size: 12px;
    color: var(--text-dim);
  }

  input {
    width: 100%;
    min-width: 0;
    font: inherit;
    color: var(--text);
    padding: 5px 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    box-shadow: var(--shadow-inner);
    user-select: text;
  }

  input:focus {
    outline: none;
    border-color: var(--frame);
  }

  input::placeholder {
    color: var(--text-faint);
  }
</style>
