<script lang="ts">
  import { Events } from '@wailsio/runtime'
  import { Panel, Button } from '@bakery/ui'
  import { UpdateService, messageOf } from '../lib/bindings'

  // The Wails updater announces what it does as wails:updater:* events;
  // UpdateService only starts the check and the install.
  type Release = { version: string; name?: string; notes?: string }
  type Progress = { written: number; total: number }

  let release = $state<Release | null>(null)
  let dismissed = $state(false)
  let installing = $state(false)
  let progress = $state<Progress | null>(null)
  let stage = $state('')
  let error = $state('')

  Events.On('wails:updater:update-available', (e) => {
    release = e.data as Release
    dismissed = false
    error = ''
  })
  Events.On('wails:updater:download-progress', (e) => (progress = e.data as Progress))
  Events.On('wails:updater:verifying', () => (stage = 'Checking the crate for tampering…'))
  Events.On('wails:updater:installing', () => (stage = 'Unpacking…'))
  Events.On('wails:updater:update-ready', () => (stage = 'Restarting…'))
  Events.On('wails:updater:error', (e) => {
    const info = e.data as { message?: string }
    // Failed checks and downloads leave the running version as it is.
    error = `The supply drop was refused and nothing was installed (${info?.message ?? 'unknown error'}).`
    installing = false
  })

  let percent = $derived(progress && progress.total > 0 ? Math.round((progress.written / progress.total) * 100) : 0)

  async function install() {
    installing = true
    error = ''
    stage = 'Hauling it in…'
    try {
      await UpdateService.Install()
    } catch (err) {
      error = messageOf(err)
      installing = false
    }
  }
</script>

{#if release && !dismissed}
  <div class="drop">
    <Panel title={`A supply drop has arrived: v${release.version}`}>
      {#if release.notes}
        <pre class="notes">{release.notes}</pre>
      {:else}
        <p class="dim">A new version of The Bakery is ready to install.</p>
      {/if}

      {#if installing}
        <div class="bar" role="progressbar" aria-valuenow={percent} aria-valuemin={0} aria-valuemax={100}>
          <div class="fill" style:width={`${percent}%`}></div>
        </div>
        <p class="dim">{stage}</p>
      {/if}
      {#if error}<p class="error" role="alert">{error}</p>{/if}

      <div class="row">
        <Button onclick={() => (dismissed = true)} disabled={installing}>Later</Button>
        <Button variant="confirm" onclick={install} disabled={installing}>Install now</Button>
      </div>
    </Panel>
  </div>
{/if}

<style>
  .drop {
    position: fixed;
    right: 12px;
    bottom: 12px;
    width: min(360px, calc(100vw - 24px));
    z-index: 10;
  }

  .notes {
    margin: 0 0 10px;
    max-height: 160px;
    overflow: auto;
    white-space: pre-wrap;
    font: inherit;
    color: var(--text-dim);
  }

  p {
    margin: 0 0 8px;
  }

  .dim {
    color: var(--text-dim);
  }

  .bar {
    height: 8px;
    margin-bottom: 6px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .fill {
    height: 100%;
    background: var(--olive-bright);
  }

  .error {
    color: var(--rust-bright);
  }

  .row {
    display: flex;
    justify-content: flex-end;
    gap: 6px;
  }
</style>
