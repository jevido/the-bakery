<script lang="ts">
  import { Panel, Button, TextField, Tabs } from '@bakery/ui'
  import type { OpenTask } from '../lib/task.svelte'
  import { COLUMN_TITLES } from '../lib/colony.svelte'
  import { renderMarkdown, openLinksOutside } from '../lib/markdown'
  import { activityLine, timeAgo, fullTime } from '../lib/activity'

  // me is the signed-in member's id: only their own comments can be changed.
  let { open, me, onclose }: { open: OpenTask; me: number; onclose: () => void } = $props()

  const TABS = [
    { id: 'comments', label: 'Comments' },
    { id: 'activity', label: 'Activity' },
  ]
  let tab = $state('comments')

  let newComment = $state('')
  let sending = $state(false)
  let editingComment = $state<number | null>(null)
  let commentDraft = $state('')
  let deletingComment = $state<number | null>(null)
  // Re-render relative times now and then.
  let now = $state(Date.now())
  $effect(() => {
    const timer = setInterval(() => (now = Date.now()), 30_000)
    return () => clearInterval(timer)
  })

  async function sendComment() {
    const body = newComment.trim()
    if (!body || sending) return
    sending = true
    if (await open.addComment(body)) newComment = ''
    sending = false
  }

  function commentKeys(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      event.preventDefault()
      sendComment()
    }
  }

  function startEditComment(id: number, body: string) {
    deletingComment = null
    editingComment = id
    commentDraft = body
  }

  async function saveComment() {
    const body = commentDraft.trim()
    if (editingComment === null || !body) return
    if (await open.editComment(editingComment, body)) editingComment = null
  }

  function editKeys(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      event.preventDefault()
      saveComment()
    }
  }

  let task = $derived(open.task)

  // Title and description are edited as drafts and saved on blur, Enter
  // (title) or Ctrl+S (description).
  let titleDraft = $state('')
  let descriptionDraft = $state('')
  let editing = $state(false)
  let saved = $state(false)
  let newSubtask = $state('')

  // Reset the drafts whenever another task opens (or the same one reloads
  // while nobody is typing in it).
  let draftsFor = 0
  $effect.pre(() => {
    if (!task) return
    if (task.id !== draftsFor) {
      draftsFor = task.id
      editing = task.description === ''
    }
    if (document.activeElement?.closest('.title-field') == null) titleDraft = task.title
    if (!editing) descriptionDraft = task.description
  })

  let done = $derived(open.subtasks.filter((s) => s.done).length)
  let preview = $derived(renderMarkdown(task?.description ?? ''))

  async function saveTitle() {
    const title = titleDraft.trim()
    if (!task || !title || title === task.title) {
      if (task) titleDraft = task.title
      return
    }
    await open.rename(title)
  }

  async function saveDescription() {
    if (!task || descriptionDraft === task.description) return
    if (await open.describe(descriptionDraft)) {
      saved = true
      setTimeout(() => (saved = false), 1200)
    }
  }

  function descriptionKeys(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.key === 's') {
      event.preventDefault()
      saveDescription()
    }
  }

  async function finishEditing() {
    await saveDescription()
    editing = false
  }

  async function addSubtask(event: KeyboardEvent) {
    if (event.key !== 'Enter') return
    const title = newSubtask.trim()
    if (title && (await open.addSubtask(title))) newSubtask = ''
  }

  function renameSubtask(id: number, current: string, event: Event) {
    const input = event.currentTarget as HTMLInputElement
    const title = input.value.trim()
    if (!title) input.value = current
    else if (title !== current) open.renameSubtask(id, title)
  }

  function subtaskKeys(event: KeyboardEvent) {
    if (event.key === 'Enter') (event.currentTarget as HTMLInputElement).blur()
  }

  // Dragging subtasks works like dragging cards on the board: the drop index
  // counts the other rows whose middle is above the pointer.
  let draggedId = $state<number | null>(null)
  let dropIndex = $state<number | null>(null)

  function dragStart(event: DragEvent, id: number) {
    draggedId = id
    event.dataTransfer?.setData('text/plain', String(id))
    if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
  }

  function dragEnd() {
    draggedId = null
    dropIndex = null
  }

  function dragOver(event: DragEvent) {
    if (draggedId === null) return
    event.preventDefault()
    const rows = [...(event.currentTarget as HTMLElement).querySelectorAll<HTMLElement>('[data-subtask-id]')].filter(
      (el) => Number(el.dataset.subtaskId) !== draggedId,
    )
    dropIndex = rows.filter((el) => {
      const r = el.getBoundingClientRect()
      return r.top + r.height / 2 < event.clientY
    }).length
  }

  function drop(event: DragEvent) {
    event.preventDefault()
    const id = draggedId
    const index = dropIndex
    dragEnd()
    if (id !== null && index !== null) open.moveSubtask(id, index)
  }

  function indicatorAt(): number {
    if (dropIndex === null) return -1
    let seen = 0
    for (let i = 0; i < open.subtasks.length; i++) {
      if (open.subtasks[i].id === draggedId) continue
      if (seen === dropIndex) return i
      seen++
    }
    return open.subtasks.length
  }

  function closeOnEscape(event: KeyboardEvent) {
    if (event.key !== 'Escape' || !task) return
    // Esc first leaves an open comment edit or delete question.
    if (editingComment !== null || deletingComment !== null) {
      editingComment = deletingComment = null
      return
    }
    onclose()
  }
</script>

<svelte:window onkeydown={closeOnEscape} />

<aside class="task-panel" aria-label="Task">
  <Panel title={task ? (COLUMN_TITLES[task.column] ?? 'Task') : 'Task'}>
    {#snippet actions()}
      <button class="close" aria-label="Close" onclick={onclose}>×</button>
    {/snippet}

    {#if open.error}<p class="error" role="alert">{open.error}</p>{/if}

    {#if !task}
      <p class="dim">Fetching the task…</p>
    {:else}
      <div class="title-field">
        <input
          class="title"
          aria-label="Title"
          maxlength={200}
          bind:value={titleDraft}
          onblur={saveTitle}
          onkeydown={(e) => e.key === 'Enter' && (e.currentTarget as HTMLInputElement).blur()}
        />
      </div>

      <section>
        <header>
          <h3>Description</h3>
          <span class="row">
            {#if saved}<span class="saved">saved</span>{/if}
            {#if editing}
              <Button onclick={finishEditing}>Preview</Button>
            {:else}
              <Button onclick={() => (editing = true)}>Edit</Button>
            {/if}
          </span>
        </header>
        {#if editing}
          <textarea
            aria-label="Description"
            placeholder="What needs doing, in markdown. Ctrl+S saves."
            bind:value={descriptionDraft}
            onblur={saveDescription}
            onkeydown={descriptionKeys}
          ></textarea>
        {:else if task.description}
          <!-- The HTML is markdown rendered with raw HTML escaped, then sanitised. -->
          <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
          <div class="markdown" onclick={openLinksOutside}>{@html preview}</div>
        {:else}
          <p class="dim">No description.</p>
        {/if}
      </section>

      <section>
        <header>
          <h3>Subtasks{#if open.subtasks.length} · {done}/{open.subtasks.length}{/if}</h3>
        </header>
        {#if open.subtasks.length}
          {@const at = indicatorAt()}
          <ul class="subtasks" ondragover={dragOver} ondrop={drop}>
            {#each open.subtasks as st, i (st.id)}
              {#if at === i}<li class="indicator" aria-hidden="true"></li>{/if}
              <li class={['subtask', { dragging: draggedId === st.id, done: st.done }]} data-subtask-id={st.id}>
                <span
                  class="handle"
                  draggable="true"
                  role="button"
                  tabindex="-1"
                  aria-label="Drag to reorder"
                  ondragstart={(e) => dragStart(e, st.id)}
                  ondragend={dragEnd}>⋮⋮</span
                >
                <input type="checkbox" aria-label="Done" checked={st.done} onchange={() => open.toggle(st.id)} />
                <input
                  class="subtask-title"
                  aria-label="Subtask title"
                  maxlength={200}
                  value={st.title}
                  onchange={(e) => renameSubtask(st.id, st.title, e)}
                  onkeydown={subtaskKeys}
                />
                <button class="delete" aria-label="Delete subtask" onclick={() => open.deleteSubtask(st.id)}>×</button>
              </li>
            {/each}
            {#if at === open.subtasks.length}<li class="indicator" aria-hidden="true"></li>{/if}
          </ul>
        {/if}
        <TextField placeholder="Add a subtask, then Enter" maxlength={200} bind:value={newSubtask} onkeydown={addSubtask} />
      </section>

      <Tabs label="Talk and history" tabs={TABS} bind:active={tab}>
        {#snippet children(active)}
          {#if active === 'comments'}
            {#if open.comments.length === 0}
              <p class="dim">No comments yet.</p>
            {/if}
            <ul class="comments">
              {#each open.comments as c (c.id)}
                <li class="comment">
                  <div class="meta">
                    <strong>{c.author_name}</strong>
                    <span title={fullTime(c.created_at)}>{timeAgo(c.created_at, now)}</span>
                    {#if c.edited_at}<span class="dim" title={fullTime(c.edited_at)}>edited</span>{/if}
                    {#if c.author_id === me && editingComment !== c.id}
                      <span class="own">
                        <button class="link" onclick={() => startEditComment(c.id, c.body)}>Edit</button>
                        <button class="link danger" onclick={() => ((deletingComment = c.id), (editingComment = null))}>Delete</button>
                      </span>
                    {/if}
                  </div>
                  {#if editingComment === c.id}
                    <textarea aria-label="Edit comment" bind:value={commentDraft} onkeydown={editKeys}></textarea>
                    <span class="row">
                      <Button variant="confirm" onclick={saveComment} disabled={!commentDraft.trim()}>Save</Button>
                      <Button onclick={() => (editingComment = null)}>Cancel</Button>
                    </span>
                  {:else}
                    <!-- Rendered like the description: raw HTML escaped, then sanitised. -->
                    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
                    <div class="markdown" onclick={openLinksOutside}>{@html renderMarkdown(c.body)}</div>
                  {/if}
                  {#if deletingComment === c.id}
                    <div class="confirm" role="alertdialog" aria-label="Delete this comment?">
                      <span>Delete this comment?</span>
                      <span class="row">
                        <Button variant="danger" onclick={() => open.deleteComment(c.id)}>Delete</Button>
                        <Button onclick={() => (deletingComment = null)}>Keep</Button>
                      </span>
                    </div>
                  {/if}
                </li>
              {/each}
            </ul>
            <div class="write">
              <textarea
                aria-label="New comment"
                placeholder="Write a comment, in markdown. Ctrl+Enter sends."
                bind:value={newComment}
                onkeydown={commentKeys}
              ></textarea>
              <Button variant="confirm" onclick={sendComment} disabled={sending || !newComment.trim()}>Comment</Button>
            </div>
          {:else}
            {#if open.activity.length === 0}
              <p class="dim">Nothing has happened yet.</p>
            {/if}
            <ul class="activity">
              {#each open.activity as e (e.id)}
                <li>
                  <span>{activityLine(e)}</span>
                  <span class="dim when" title={fullTime(e.at)}>{timeAgo(e.at, now)}</span>
                </li>
              {/each}
            </ul>
            {#if open.moreActivity}
              <Button onclick={() => open.loadMoreActivity()}>Load more</Button>
            {/if}
          {/if}
        {/snippet}
      </Tabs>
    {/if}
  </Panel>
</aside>

<style>
  .task-panel {
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
  }

  .task-panel :global(.panel) {
    flex: 1;
  }

  .task-panel :global(.panel > .body) {
    display: flex;
    flex-direction: column;
    gap: 12px;
    overflow-y: auto;
  }

  .close {
    padding: 0 4px;
    font-size: 16px;
    line-height: 1;
    color: var(--text-dim);
    background: none;
    border: none;
    cursor: pointer;
  }

  .close:hover {
    color: var(--text);
  }

  .title {
    width: 100%;
    box-sizing: border-box;
    font: inherit;
    font-size: 16px;
    font-weight: 600;
    color: var(--text);
    padding: 4px 6px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--radius);
    user-select: text;
  }

  .title:hover,
  .title:focus {
    background: var(--panel-inset);
    border-color: var(--frame-dim);
  }

  section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  section header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
  }

  h3 {
    margin: 0;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-dim);
  }

  .row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .saved {
    font-size: 12px;
    color: var(--olive-bright);
  }

  textarea {
    min-height: 140px;
    resize: vertical;
    font: inherit;
    color: var(--text);
    padding: 6px 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    user-select: text;
  }

  textarea:focus {
    border-color: var(--frame);
    outline: none;
  }

  .markdown {
    overflow-wrap: anywhere;
    user-select: text;
    line-height: 1.45;
  }

  .markdown :global(:first-child) {
    margin-top: 0;
  }

  .markdown :global(:last-child) {
    margin-bottom: 0;
  }

  .markdown :global(p),
  .markdown :global(ul),
  .markdown :global(ol) {
    margin: 0 0 6px;
  }

  .markdown :global(ul),
  .markdown :global(ol) {
    padding-left: 20px;
  }

  .markdown :global(a) {
    color: var(--steel-bright);
  }

  .markdown :global(code) {
    padding: 0 3px;
    font-size: 13px;
    background: var(--panel-inset);
    border-radius: var(--radius);
  }

  .markdown :global(pre) {
    padding: 6px 8px;
    overflow-x: auto;
    background: var(--panel-inset);
    border-radius: var(--radius);
  }

  .comments,
  .activity {
    list-style: none;
    margin: 0 0 8px;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .comment {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .meta {
    display: flex;
    align-items: baseline;
    gap: 6px;
    font-size: 12px;
    color: var(--text-dim);
  }

  .meta strong {
    color: var(--text);
    font-weight: 600;
  }

  .own {
    margin-left: auto;
    display: flex;
    gap: 8px;
  }

  .link {
    font: inherit;
    padding: 0;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
  }

  .link.danger {
    color: var(--rust-bright);
  }

  .confirm {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
    padding: 6px;
    background: var(--panel-inset);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  .write {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 6px;
  }

  .write textarea,
  .comment textarea {
    width: 100%;
    box-sizing: border-box;
    min-height: 64px;
  }

  .activity {
    gap: 6px;
  }

  .activity li {
    display: flex;
    justify-content: space-between;
    gap: 8px;
  }

  .when {
    flex-shrink: 0;
    font-size: 12px;
  }

  .subtasks {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .subtask {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 2px 4px;
    border-radius: var(--radius);
  }

  .subtask:hover {
    background: var(--panel-raised);
  }

  .subtask.dragging {
    opacity: 0.4;
  }

  .handle {
    color: var(--text-faint);
    font-size: 11px;
    letter-spacing: -2px;
    cursor: grab;
  }

  input[type='checkbox'] {
    accent-color: var(--olive);
  }

  .subtask-title {
    flex: 1;
    min-width: 0;
    font: inherit;
    color: var(--text);
    padding: 2px 4px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--radius);
    user-select: text;
  }

  .subtask-title:focus {
    background: var(--panel-inset);
    border-color: var(--frame-dim);
    outline: none;
  }

  .done .subtask-title {
    color: var(--text-faint);
    text-decoration: line-through;
  }

  .delete {
    visibility: hidden;
    padding: 0 4px;
    font-size: 15px;
    line-height: 1.2;
    color: var(--text-faint);
    background: none;
    border: none;
    cursor: pointer;
  }

  .subtask:hover .delete,
  .delete:focus-visible {
    visibility: visible;
  }

  .delete:hover {
    color: var(--rust-bright);
  }

  .indicator {
    height: 2px;
    margin: -1px 0;
    background: var(--steel-bright);
    box-shadow: 0 0 4px var(--steel);
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .error {
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
