<script lang="ts">
  import { Button, Panel } from '@bakery/ui'
  import { messageOf } from '../lib/bindings'
  import { preload, play } from '../lib/sound'
  import { saveSettings, settings } from '../lib/settings.svelte'

  // The desktop's own settings. They stay on this machine.
  let error = $state('')
  let saved = $state(false)

  async function save() {
    error = ''
    saved = false
    try {
      await saveSettings()
      saved = true
    } catch (err) {
      error = messageOf(err)
    }
  }

  function toggleSound() {
    settings.sound = !settings.sound
    if (settings.sound) {
      preload()
      play('task-done')
    }
    save()
  }
</script>

<Panel title="Settings">
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <section>
    <label class="switch">
      <input
        type="checkbox"
        checked={settings.quiet}
        onchange={(e) => {
          settings.quiet = e.currentTarget.checked
          save()
        }}
      />
      <strong>Quiet colony</strong>
    </label>
    <p class="help">
      Turns the flavor off: moods, event letters, the colony's turns of phrase, sounds and idle antics. The board
      reads plainly. Portraits stay, so people and agents can still be told apart, and so do alerts and letters
      that ask you something: they are work, not flavor.
    </p>
  </section>
  <section>
    <label class="switch">
      <input type="checkbox" checked={settings.sound} disabled={settings.quiet} onchange={toggleSound} />
      <strong>Sound</strong>
    </label>
    <p class="help">Short cues when a task reaches Done, a letter arrives, or a run here finishes or fails. Off until you turn it on.</p>
    <label class="volume">
      Volume
      <input
        type="range"
        min="0"
        max="1"
        step="0.05"
        bind:value={settings.volume}
        disabled={!settings.sound || settings.quiet}
        onchange={() => {
          save()
          play('letter')
        }}
      />
    </label>
    <Button disabled={!settings.sound || settings.quiet} onclick={() => play('run-finished')}>Play a cue</Button>
  </section>
  <section>
    <label class="volume">
      <strong>Supervisor model</strong>
      <select bind:value={settings.supervisorModel} onchange={save} aria-label="Supervisor model">
        <option value="sonnet">sonnet</option>
        <option value="opus">opus</option>
        <option value="haiku">haiku</option>
        {#if !['sonnet', 'opus', 'haiku'].includes(settings.supervisorModel)}<option value={settings.supervisorModel}>{settings.supervisorModel}</option>{/if}
      </select>
    </label>
    <p class="help">
      The model the supervisor in Work mode answers with. Every message to the supervisor is one short Claude call on
      a snapshot of the board, and costs what that model costs; the canvas costs nothing.
    </p>
  </section>
  {#if saved}<p class="dim" role="status">Saved on this machine.</p>{/if}
</Panel>

<style>
  section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px 0;
    border-bottom: 1px solid var(--frame-dim);
  }

  .switch {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .help {
    margin: 0;
    max-width: 60ch;
    font-size: 12px;
    color: var(--text-dim);
  }

  .volume {
    display: flex;
    gap: 8px;
    align-items: center;
    font-size: 12px;
  }

  select {
    font: inherit;
    font-size: 12px;
    color: var(--text);
    padding: 2px 6px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .dim {
    color: var(--text-faint);
    font-size: 12px;
  }

  .error {
    margin: 0;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
