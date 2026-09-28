<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import { WorkshopService, messageOf, type BoardConfig } from '../lib/bindings'
  import type { Colony } from '../lib/colony.svelte'

  // How this machine works the open board: which repo runs happen in, where
  // their worktrees go, how many at once and at what cost. Saved in the
  // board's config folder; nothing here reaches the API.
  let { colony, onclose }: { colony: Colony; onclose: () => void } = $props()

  const settings = $derived(colony.settings)
  let draft = $state<BoardConfig | null>(null)
  let problem = $state<{ field: string; message: string } | null>(null)
  let saved = $state(false)
  let saving = $state(false)

  // A fresh draft whenever the board (or its saved config) changes.
  $effect(() => {
    draft = settings ? { ...settings.config, agents: [...(settings.config.agents ?? [])] } : null
    problem = null
  })

  async function pickRepo() {
    if (!draft) return
    try {
      const dir = await WorkshopService.PickRepo()
      if (dir) draft.repo = dir
    } catch (err) {
      problem = { field: '', message: messageOf(err) }
    }
  }

  async function save(event: SubmitEvent) {
    event.preventDefault()
    if (!draft || colony.boardId === null) return
    saving = true
    saved = false
    try {
      const refused = await WorkshopService.SaveBoardConfig(colony.boardId, {
        ...draft,
        repo: draft.repo.trim(),
        base_branch: draft.base_branch.trim(),
        worktree_root: draft.worktree_root.trim(),
        max_concurrent_runs: Number(draft.max_concurrent_runs),
        max_concurrent_runs_fast: Number(draft.max_concurrent_runs_fast),
        max_budget_usd: Number(draft.max_budget_usd),
        agents: draft.agents ?? [],
      })
      problem = refused ?? null
      if (!refused) {
        await colony.reloadSettings()
        saved = true
      }
    } catch (err) {
      problem = { field: '', message: messageOf(err) }
    }
    saving = false
  }

  function openFolder() {
    if (colony.boardId !== null) WorkshopService.OpenBoardConfigFolder(colony.boardId).catch((err) => (problem = { field: '', message: messageOf(err) }))
  }
</script>

<aside class="settings">
  <Panel title="Board settings">
    {#snippet actions()}
      <button class="close" aria-label="Close" onclick={onclose}>×</button>
    {/snippet}
    {#if settings && draft}
      <p class="dim">
        How this machine works <strong>{colony.view?.board.name}</strong>. These settings stay on this machine.
      </p>
      {#if settings.problem && !problem}<p class="error">{settings.problem}</p>{/if}
      {#if problem}<p class="error" role="alert">{problem.message}</p>{/if}
      <p class={['state', { linked: settings.linked }]}>
        {settings.linked ? `Linked to ${settings.config.repo}` : 'Not linked: agents cannot run on this board here yet.'}
      </p>

      <form onsubmit={save}>
        <div class="repo">
          <TextField label="Repository" placeholder="/home/you/Projects/colony-site" bind:value={draft.repo} aria-invalid={problem?.field === 'repo'} />
          <Button type="button" onclick={pickRepo}>Choose…</Button>
        </div>
        <TextField label="Base branch" placeholder="main" bind:value={draft.base_branch} aria-invalid={problem?.field === 'base_branch'} />
        <TextField
          label="Worktree folder"
          placeholder="next to the repository, in .bakery-worktrees"
          bind:value={draft.worktree_root}
          aria-invalid={problem?.field === 'worktree_root'}
        />
        <div class="row2">
          <label class="field">
            <span>Runs at once</span>
            <input type="number" min="1" max="8" bind:value={draft.max_concurrent_runs} aria-invalid={problem?.field === 'max_concurrent_runs'} />
          </label>
          <label class="field">
            <span>Runs at once, fast</span>
            <input type="number" min="1" max="8" bind:value={draft.max_concurrent_runs_fast} aria-invalid={problem?.field === 'max_concurrent_runs_fast'} />
          </label>
        </div>
        <div class="row2">
          <label class="field">
            <span>Budget per run ($)</span>
            <input type="number" min="0.01" step="0.01" bind:value={draft.max_budget_usd} aria-invalid={problem?.field === 'max_budget_usd'} />
          </label>
        </div>
        <label class="field">
          <span>When a run succeeds, move the task to</span>
          <select bind:value={draft.finish_column}>
            <option value="">Leave it where it is</option>
            {#each colony.view?.columns ?? [] as c (c.id)}<option value={c.name}>{c.name}</option>{/each}
            {#if draft.finish_column && !colony.view?.columns.some((c) => c.name === draft?.finish_column)}
              <option value={draft.finish_column}>{draft.finish_column} (not on this board)</option>
            {/if}
          </select>
        </label>
        <fieldset class="colony">
          <legend>The colony</legend>
          <label class="field">
            <span>Agents take work from</span>
            <select bind:value={draft.ready_column}>
              {#each colony.view?.columns ?? [] as c (c.id)}<option value={c.name}>{c.name}</option>{/each}
              {#if draft.ready_column && !colony.view?.columns.some((c) => c.name === draft?.ready_column)}
                <option value={draft.ready_column}>{draft.ready_column} (not on this board)</option>
              {/if}
            </select>
          </label>
          <span class="label">Agents who pick up work here on their own</span>
          {#each colony.agents as a (a.slug)}
            <label class="check">
              <input type="checkbox" value={a.slug} bind:group={draft.agents} />
              <span>{a.name}{a.title ? ` · ${a.title}` : ''}</span>
            </label>
          {:else}
            <span class="dim small">No agents yet; make them on the Agents screen.</span>
          {/each}
          <span class="dim small">They start when the board is not paused (⏸ ▶ ▶▶ in the header).</span>
        </fieldset>
        <label class="check">
          <input type="checkbox" bind:checked={draft.isolate_user_settings} />
          <span>Leave out my own Claude settings (<code>~/.claude</code> skills, hooks and plugins)</span>
        </label>
        <div class="actions">
          <Button type="submit" variant="confirm" disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
          <Button type="button" onclick={openFolder}>Open config folder</Button>
          {#if saved}<span class="dim">Saved</span>{/if}
        </div>
      </form>
      <p class="dim small">Folder: <code>{settings.dir}</code></p>
    {:else}
      <p class="dim">Loading…</p>
    {/if}
  </Panel>
</aside>

<style>
  .settings {
    min-height: 0;
    overflow: auto;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 10px;
  }

  .repo {
    display: flex;
    gap: 6px;
    align-items: flex-end;
  }

  .repo :global(.field) {
    flex: 1;
  }

  .row2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }

  .field > span {
    font-size: 12px;
    color: var(--text-dim);
  }

  input[type='number'],
  select {
    font: inherit;
    color: var(--text);
    padding: 5px 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    user-select: text;
  }

  :global([aria-invalid='true']) {
    border-color: var(--rust) !important;
  }

  .colony {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 0;
    padding: 8px;
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  legend {
    padding: 0 4px;
    font-size: 12px;
    color: var(--text-dim);
  }

  .label {
    font-size: 12px;
    color: var(--text-dim);
  }

  .check {
    display: flex;
    gap: 6px;
    align-items: flex-start;
    font-size: 13px;
  }

  input[type='checkbox'] {
    accent-color: var(--olive);
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .state {
    margin-top: 8px;
    padding: 5px 8px;
    color: var(--text-dim);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    overflow-wrap: anywhere;
  }

  .state.linked {
    color: var(--olive-bright);
    border-color: var(--olive);
  }

  .close {
    font: inherit;
    font-size: 16px;
    padding: 0 4px;
    color: var(--text-dim);
    background: none;
    border: none;
    cursor: pointer;
  }

  p {
    margin: 0;
  }

  code {
    font-size: 12px;
    overflow-wrap: anywhere;
  }

  .dim {
    color: var(--text-dim);
  }

  .small {
    margin-top: 10px;
    font-size: 12px;
  }

  .error {
    margin-top: 8px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
