<script lang="ts">
  import { Button } from '@bakery/ui'

  // An in-page confirmation: the question and two buttons, no browser dialog.
  let {
    question,
    action,
    onconfirm,
    oncancel,
    busy = false,
  }: { question: string; action: string; onconfirm: () => void; oncancel: () => void; busy?: boolean } = $props()
</script>

<div class="confirm" role="alertdialog" aria-label={question}>
  <span>{question}</span>
  <div class="row">
    <Button variant="danger" onclick={onconfirm} disabled={busy}>{action}</Button>
    <Button onclick={oncancel} disabled={busy}>Keep</Button>
  </div>
</div>

<style>
  .confirm {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px;
    background: var(--panel-inset);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  .row {
    display: flex;
    gap: 6px;
  }
</style>
