<script lang="ts">
  // One need as a labelled bar, 0 to 100: green when full, amber when low,
  // rust when nearly empty, like a colonist's needs.
  let { label, value, hint = '' }: { label: string; value: number; hint?: string } = $props()

  let level = $derived(Math.max(0, Math.min(100, Math.round(value))))
  let tone = $derived(level >= 60 ? 'full' : level >= 30 ? 'low' : 'empty')
</script>

<div class="need" title={hint || undefined}>
  <span class="label">{label}</span>
  <span
    class={['bar', tone]}
    role="meter"
    aria-label={label}
    aria-valuemin={0}
    aria-valuemax={100}
    aria-valuenow={level}
    style:--level="{level}%"
  ></span>
  <span class="value">{level}</span>
</div>

<style>
  .need {
    display: grid;
    grid-template-columns: 56px 1fr 26px;
    gap: 6px;
    align-items: center;
    font-size: 12px;
  }

  .label {
    color: var(--text-dim);
  }

  .value {
    text-align: right;
    color: var(--text-faint);
    font-variant-numeric: tabular-nums;
  }

  .bar {
    position: relative;
    height: 9px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .bar::after {
    content: '';
    position: absolute;
    inset: 0;
    width: var(--level);
    background: var(--tone);
  }

  .full {
    --tone: var(--olive);
  }

  .low {
    --tone: var(--amber);
  }

  .empty {
    --tone: var(--rust);
  }
</style>
