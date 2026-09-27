<script lang="ts">
  import { Panel } from '@bakery/ui'
  import DownloadButton from '../components/DownloadButton.svelte'
</script>

<svelte:head>
  <title>The Bakery — a workbench for your guild</title>
  <meta name="description" content="A desktop workbench where your guild plans the work on shared boards, and soon hands it to Claude." />
</svelte:head>

<section class="hero">
  <h1>The Bakery</h1>
  <p class="lede">A workbench where your guild plans the work and, soon, Claude helps do it.</p>
  <div class="actions">
    <DownloadButton />
    <a class="signup" href="/signup">Sign up to found a guild</a>
  </div>
</section>

<div class="panels">
  <Panel title="Guilds">
    <p>A guild is the people who work together. Found one here on the website, send invite links, and everyone in it can do everything. No org charts, no permission screens.</p>
  </Panel>
  <Panel title="Boards">
    <p>Plan the work as tasks on boards: Backlog, To do, Doing, Done. Drag them along; the whole guild sees the same board.</p>
  </Panel>
  <Panel title="Agents · coming">
    <p>Claude agents you set up like colonists: skills, priorities, a task to work on. Not built yet.</p>
  </Panel>
</div>

<p class="note">
  Early days: the colony has guilds, boards and not much else yet. See
  <a href="/how-it-works">how it works</a>, or <a href="/contributors">help build it</a>.
</p>

<style>
  .hero {
    padding: 24px 0 28px;
  }

  h1 {
    margin: 0 0 8px;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: clamp(36px, 8vw, 56px);
    line-height: 1.1;
  }

  .lede {
    margin: 0 0 20px;
    max-width: 36em;
    font-size: 18px;
    color: var(--text-dim);
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px 20px;
  }

  .signup {
    font-family: var(--font-display);
  }

  .panels {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: var(--gap);
  }

  .panels p {
    margin: 0;
    color: var(--text-dim);
  }

  .note {
    margin: 24px 0 0;
    color: var(--text-faint);
  }
</style>
