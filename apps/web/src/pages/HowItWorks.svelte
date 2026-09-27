<script lang="ts">
  import { Panel } from '@bakery/ui'
</script>

<svelte:head>
  <title>How it works — The Bakery</title>
  <meta name="description" content="The Bakery in three parts: the desktop app for the work, the website for running the guild, the API that keeps it all." />
</svelte:head>

<h1>How it works</h1>
<p class="lede">Three parts, one colony.</p>

<figure class="diagram">
  <svg viewBox="0 0 640 220" role="img" aria-labelledby="diagram-title">
    <title id="diagram-title">The desktop app and the website both talk to the API, which keeps every guild, board and task.</title>
    <g class="box">
      <rect x="10" y="20" width="200" height="70" rx="2" />
      <text x="110" y="50">Desktop app</text>
      <text x="110" y="72" class="sub">doing the work</text>
    </g>
    <g class="box">
      <rect x="430" y="20" width="200" height="70" rx="2" />
      <text x="535" y="50">Website</text>
      <text x="535" y="72" class="sub">running the guild</text>
    </g>
    <g class="box store">
      <rect x="220" y="140" width="200" height="70" rx="2" />
      <text x="320" y="170">API</text>
      <text x="320" y="192" class="sub">keeps guilds, boards, tasks</text>
    </g>
    <path class="line" d="M110 90 L110 115 L300 115 L300 140" />
    <path class="line" d="M530 90 L530 115 L340 115 L340 140" />
  </svg>
</figure>

<div class="panels">
  <Panel title="The desktop app: doing">
    <p>Where the work happens. Pick a guild, open a board, add tasks and drag them from Backlog to Done. It looks like a colony sim, remembers you, and installs new releases itself. <a href="/desktop">Get it</a>.</p>
  </Panel>
  <Panel title="The website: running the guild">
    <p>Found a guild, rename or archive it, make invite links and see who is in it. Anyone in a guild can do all of it; there are no ranks.</p>
  </Panel>
  <Panel title="The API: the colony's memory">
    <p>Every guild, board and task lives there, so everyone in a guild sees the same board. The desktop app and the website both work through it.</p>
  </Panel>
</div>

<Panel title="Coming later">
  <ul>
    <li><strong>MCP:</strong> Claude, in any client, reading and working a guild's boards.</li>
    <li><strong>Agents:</strong> your own roster of Claude agents, each with skills and work priorities, like colonists.</li>
    <li><strong>The Claude runner:</strong> put an agent on a task and watch it work in its own copy of the code.</li>
    <li><strong>The colony:</strong> idle agents picking up the next task by priority, several at once.</li>
  </ul>
  <p class="dim">None of this is built yet. The full list is on <a href="https://jevidocs.jevido.app/p/bakery/unfinished">Not built yet</a>; the details of what exists are in the <a href="https://jevidocs.jevido.app/p/bakery">docs</a>.</p>
</Panel>

<style>
  h1 {
    margin: 0 0 4px;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 32px;
  }

  .lede {
    margin: 0 0 16px;
    color: var(--text-dim);
  }

  .diagram {
    margin: 0 0 16px;
  }

  svg {
    width: 100%;
    max-width: 640px;
    height: auto;
    display: block;
    margin: 0 auto;
  }

  .box rect {
    fill: var(--panel);
    stroke: var(--frame);
  }

  .store rect {
    fill: var(--panel-title);
  }

  text {
    fill: var(--text);
    font-family: var(--font-display);
    font-size: 17px;
    text-anchor: middle;
  }

  text.sub {
    fill: var(--text-dim);
    font-family: var(--font-body);
    font-size: 12px;
  }

  .line {
    fill: none;
    stroke: var(--steel-bright);
    stroke-width: 2;
  }

  .panels {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: var(--gap);
    margin-bottom: var(--gap);
  }

  p {
    margin: 0;
    color: var(--text-dim);
  }

  ul {
    margin: 0 0 10px;
    padding-left: 18px;
    color: var(--text-dim);
  }

  li {
    margin-bottom: 4px;
  }

  strong {
    color: var(--text);
    font-weight: normal;
    font-family: var(--font-display);
  }

  .dim {
    color: var(--text-faint);
  }
</style>
