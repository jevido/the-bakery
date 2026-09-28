<script lang="ts">
  import { Panel, Button, TextField, Portrait, NeedBar } from '@bakery/ui'
  import { needs, NEED_HINTS } from '../lib/needs.svelte'
  import Confirm from './Confirm.svelte'
  import type { Roster } from '../lib/roster.svelte'
  import type { Guild, SkillEntry } from '../lib/bindings'

  // The RimWorld-style card for one agent. Edits stay in the draft until
  // Save writes agent.toml; the sync sends it.
  let { roster, guilds }: { roster: Roster; guilds: Guild[] } = $props()

  let detail = $derived(roster.detail)
  let draft = $derived(roster.draft)
  let mine = $derived(detail ? needs.bySlug[detail.slug] : undefined)
  let saving = $state(false)
  let confirmingDelete = $state(false)
  let importing = $state<SkillEntry[] | null>(null)
  let picked = $state<string[]>([])

  const MODELS = ['sonnet', 'opus', 'haiku']
  const MODE_LABELS: Record<string, string> = {
    manual: 'Ask before acting (manual)',
    acceptEdits: 'Accept file edits',
    plan: 'Plan only',
    dontAsk: 'Only allowed tools, never ask',
    auto: 'Auto',
  }

  function blockedBy(key: string): string | null {
    const trait = roster.traits.find((t) => t.key === key)
    const clash = trait?.conflicts_with?.find((c) => draft?.traits.includes(c))
    return clash ? (roster.traits.find((t) => t.key === clash)?.label ?? clash) : null
  }

  function toggleTrait(key: string) {
    if (!draft) return
    draft.traits = draft.traits.includes(key) ? draft.traits.filter((t) => t !== key) : [...draft.traits, key]
  }

  const MOOD_WORDS = { content: 'Content', okay: 'Okay', stressed: 'Stressed', breaking: 'Breaking' }

  // reroll draws a new face; like every edit it is kept once saved.
  function reroll() {
    if (!draft) return
    draft.portrait_seed = Array.from(crypto.getRandomValues(new Uint8Array(8)), (b) => b.toString(16).padStart(2, '0')).join('')
  }

  async function save() {
    saving = true
    await roster.save()
    saving = false
  }

  async function openImport() {
    picked = []
    importing = await roster.claudeSkills()
  }

  async function doImport() {
    if (await roster.importSkills(picked)) importing = null
  }
</script>

{#if detail && draft}
  <Panel>
    {#snippet header()}
      <span class="head-name">{detail.manifest.name || detail.slug}</span>
      {#if detail.unsynced}<span class="badge">not synced yet</span>{/if}
      {#if detail.conflict}<span class="badge warn">changed in two places</span>{/if}
    {/snippet}
    {#snippet actions()}
      <Button onclick={() => roster.openFolder()}>Open folder</Button>
    {/snippet}

    {#if roster.error}<p class="error" role="alert">{roster.error}</p>{/if}
    {#if detail.problem}<p class="error">Not sent: {detail.problem}</p>{/if}

    <div class="card">
      <div class="identity">
        <div class="face">
          <Portrait seed={draft.portrait_seed || detail.slug} size={88} alt={`${draft.name || detail.slug}'s portrait`} />
          <button class="reroll" onclick={reroll}>Re-roll</button>
        </div>
        <div class="names">
          <TextField label="Name" maxlength={40} bind:value={draft.name} />
          <TextField label="Title" maxlength={60} placeholder="Backend engineer" bind:value={draft.title} />
        </div>
      </div>

      {#if mine}
        <section class="needs" aria-label="Needs">
          <p class="mood">
            <strong class={['word', mine.mood]}>{MOOD_WORDS[mine.mood]}</strong>{#if mine.reason}: {mine.reason}{/if}
          </p>
          <NeedBar label="Budget" value={mine.needs.budget} hint={NEED_HINTS.budget} />
          <NeedBar label="Focus" value={mine.needs.focus} hint={NEED_HINTS.focus} />
          <NeedBar label="Morale" value={mine.needs.morale} hint={NEED_HINTS.morale} />
          <NeedBar label="Rest" value={mine.needs.rest} hint={NEED_HINTS.rest} />
        </section>
      {/if}

      {#if detail.recruited}
        <p class="origin">
          Recruited{detail.origin_gone ? '; the original is gone' : ''}.
          {#if detail.origin_newer}
            The original has changed.
            <Button onclick={() => roster.pullOrigin()}>Update from original</Button>
          {/if}
        </p>
      {/if}

      <label class="field">
        <span>Backstory</span>
        <textarea maxlength={2000} placeholder="Where they came from, what they're good at." bind:value={draft.backstory}
        ></textarea>
      </label>

      <section>
        <h3>Traits</h3>
        <div class="chips">
          {#each roster.traits as t (t.key)}
            {@const on = draft.traits.includes(t.key)}
            {@const clash = on ? null : blockedBy(t.key)}
            <button
              class={['chip', { on }]}
              disabled={!!clash}
              title={clash ? `Cannot go with ${clash}` : t.description}
              aria-pressed={on}
              onclick={() => toggleTrait(t.key)}>{t.label}</button
            >
          {/each}
        </div>
      </section>

      <div class="row2">
        <label class="field">
          <span>Model</span>
          <select bind:value={draft.model}>
            {#each MODELS as m (m)}<option value={m}>{m}</option>{/each}
            {#if !MODELS.includes(draft.model)}<option value={draft.model}>{draft.model}</option>{/if}
          </select>
        </label>
        <label class="field">
          <span>Permission mode</span>
          <select bind:value={draft.permission_mode}>
            {#each roster.permissionModes as m (m)}<option value={m}>{MODE_LABELS[m] ?? m}</option>{/each}
          </select>
        </label>
      </div>

      <label class="field">
        <span>Allowed tools, one per line</span>
        <textarea class="mono" placeholder={'Bash(go test:*)\nRead'} bind:value={draft.allowed_tools}></textarea>
      </label>

      <div class="save-row">
        <Button variant="confirm" disabled={saving || !roster.dirty || !draft.name.trim()} onclick={save}>
          {saving ? 'Saving…' : 'Save'}
        </Button>
        {#if roster.dirty}<span class="dim">Unsaved changes</span>{/if}
      </div>

      <section>
        <h3>Skills · {detail.skills?.length ?? 0}</h3>
        {#if !detail.skills?.length}
          <p class="dim">No skills yet. Import some, or add a folder with a SKILL.md under skills/.</p>
        {/if}
        <ul class="skills">
          {#each detail.skills ?? [] as s (s.name)}
            <li>
              <div>
                <strong>{s.name}</strong>
                <span class="dim">{s.description}</span>
              </div>
              <button class="link danger" onclick={() => roster.removeSkill(s.name)}>Remove</button>
            </li>
          {/each}
        </ul>
        <div class="row">
          <Button onclick={openImport}>From ~/.claude/skills</Button>
          <Button onclick={() => roster.importFolder()}>From folder…</Button>
        </div>
        {#if importing}
          <div class="import">
            {#if importing.length === 0}
              <p class="dim">No skills in ~/.claude/skills.</p>
            {/if}
            {#each importing as s (s.dir)}
              <label class="pick">
                <input type="checkbox" value={s.dir} bind:group={picked} />
                <span><strong>{s.name}</strong> <span class="dim">{s.description}</span></span>
              </label>
            {/each}
            <div class="row">
              <Button variant="confirm" disabled={picked.length === 0} onclick={doImport}>Import {picked.length || ''}</Button>
              <Button onclick={() => (importing = null)}>Cancel</Button>
            </div>
          </div>
        {/if}
      </section>

      <section>
        <h3>Sharing</h3>
        {#if !detail.agent_id}
          <p class="dim">Share once the agent has synced.</p>
        {:else}
          {#each guilds as g (g.id)}
            <label class="pick">
              <input
                type="checkbox"
                checked={detail.shared_with?.includes(g.id)}
                onchange={(e) => roster.share(g.id, (e.currentTarget as HTMLInputElement).checked)}
              />
              <span>Shared with {g.name}</span>
            </label>
          {/each}
          <p class="dim small">Members of a guild can recruit their own copy. Your agent stays yours.</p>
        {/if}
      </section>

      <section>
        {#if confirmingDelete}
          <Confirm
            question={`Send ${detail.manifest.name || detail.slug} away for good? Their folder moves to .trash.`}
            action="Delete"
            onconfirm={() => roster.remove(detail.slug)}
            oncancel={() => (confirmingDelete = false)}
          />
        {:else}
          <Button variant="danger" onclick={() => (confirmingDelete = true)}>Delete agent</Button>
        {/if}
      </section>
    </div>
  </Panel>
{/if}

<style>
  .card {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .head-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .badge {
    padding: 0 6px;
    font-family: var(--font-body);
    font-size: 11px;
    font-weight: 400;
    letter-spacing: 0;
    color: var(--text-dim);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .badge.warn {
    color: var(--rust-bright);
    border-color: var(--rust);
  }

  .identity {
    display: flex;
    gap: 12px;
    align-items: flex-start;
  }

  .names {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .origin {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    color: var(--text-dim);
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

  textarea,
  select {
    font: inherit;
    color: var(--text);
    padding: 5px 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    user-select: text;
  }

  textarea {
    min-height: 64px;
    resize: vertical;
  }

  .mono {
    font-family: ui-monospace, monospace;
    font-size: 12px;
  }

  .row2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  h3 {
    margin: 0 0 6px;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-dim);
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .chip {
    font: inherit;
    font-size: 12px;
    padding: 3px 9px;
    color: var(--text-dim);
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: 999px;
    cursor: pointer;
  }

  .chip.on {
    color: var(--bg-deep);
    background: var(--olive);
    border-color: var(--olive-bright);
  }

  .chip:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .save-row,
  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
  }

  .skills {
    list-style: none;
    margin: 0 0 8px;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .skills li {
    display: flex;
    justify-content: space-between;
    gap: 8px;
  }

  .skills li div {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .import {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 8px;
    padding: 8px;
    max-height: 260px;
    overflow: auto;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .pick {
    display: flex;
    gap: 6px;
    align-items: flex-start;
  }

  input[type='checkbox'] {
    accent-color: var(--olive);
  }

  .link {
    font: inherit;
    font-size: 12px;
    padding: 0;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
  }

  .link.danger {
    color: var(--rust-bright);
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .small {
    font-size: 12px;
    margin-top: 4px;
  }

  .error {
    margin-bottom: 8px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  .face {
    display: flex;
    flex-direction: column;
    gap: 4px;
    align-items: center;
  }

  .reroll {
    font: inherit;
    font-size: 11px;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
    text-decoration: underline;
  }

  .needs {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 6px 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .mood {
    margin: 0 0 3px;
    font-size: 12px;
    color: var(--text-dim);
  }

  .word.content {
    color: var(--olive-bright);
  }

  .word.okay {
    color: var(--text);
  }

  .word.stressed {
    color: var(--amber);
  }

  .word.breaking {
    color: var(--rust-bright);
  }
</style>
