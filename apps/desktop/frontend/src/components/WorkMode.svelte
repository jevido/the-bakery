<script lang="ts">
  import WorkCanvas from './WorkCanvas.svelte'
  import SupervisorChat from './SupervisorChat.svelte'
  import { SupervisorChat as Chat } from '../lib/supervisor.svelte'
  import TimeControls from './TimeControls.svelte'
  import type { Colony } from '../lib/colony.svelte'
  import type { Letters } from '../lib/letters.svelte'

  // Work mode: one board at a time, its agents on the canvas at the left
  // and the supervisor chat at the right. Boards is where the work is
  // planned; this is where it is watched and steered.
  // onsettings opens the board's settings, which live on the Boards screen.
  // onopenrun shows a run in the run panel, on the Boards screen.
  let { colony, letters, onsettings, onopenrun }: { colony: Colony; letters: Letters; onsettings: () => void; onopenrun: (id: string) => void } = $props()

  const chat = new Chat()
  $effect(() => {
    chat.load(colony.boardId)
  })
</script>

<section class="work-mode">
  <header>
    <label class="picker">
      <span>Board</span>
      <select
        value={colony.boardId ?? ''}
        onchange={(e) => colony.openBoard(Number(e.currentTarget.value))}
        aria-label="Board to work on"
      >
        {#each colony.boards as b (b.id)}
          <option value={b.id}>{b.name}</option>
        {/each}
      </select>
    </label>
    <TimeControls {colony} />
    <span class="spacer"></span>
    {#if colony.settings && !colony.settings.linked}
      <button class="linked" title="Agents cannot run on this board on this machine until it is linked to a repository" onclick={onsettings}
        >Not linked</button
      >
    {/if}
  </header>
  <div class="split">
    <div class="canvas">
      {#if colony.view}
        {#key colony.boardId}<WorkCanvas {colony} {letters} {onopenrun} />{/key}
      {:else}
        <p class="dim">Pick a board.</p>
      {/if}
    </div>
    <div class="side">
      <SupervisorChat {chat} {colony} />
    </div>
  </div>
</section>

<style>
  .work-mode {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: var(--gap);
  }

  header {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  .picker {
    display: flex;
    gap: 6px;
    align-items: center;
    font-size: 12px;
    color: var(--text-dim);
  }

  select {
    font: inherit;
    font-size: 13px;
    color: var(--text);
    padding: 3px 6px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .spacer {
    flex: 1;
  }

  .linked {
    font: inherit;
    font-size: 11px;
    padding: 1px 8px;
    color: var(--rust-bright);
    background: none;
    border: 1px solid var(--rust);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .split {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 4fr) minmax(280px, 1fr);
    gap: var(--gap);
  }

  .side {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .side > :global(.panel) {
    flex: 1;
    min-height: 0;
  }

  .canvas {
    min-width: 0;
    min-height: 0;
  }

  .dim {
    color: var(--text-dim);
  }
</style>
