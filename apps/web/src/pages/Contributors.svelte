<script lang="ts">
  import { Panel } from '@bakery/ui'
</script>

<svelte:head>
  <title>For contributors — The Bakery</title>
  <meta name="description" content="The Bakery is open source. How to run it, how work is planned, and the house rules." />
</svelte:head>

<h1>For contributors</h1>
<p class="lede">The Bakery is open source, and the colony takes volunteers.</p>

<div class="panels">
  <Panel title="Get the code running">
    <p>The code is at <a href="https://github.com/jevido/the-bakery">github.com/jevido/the-bakery</a>. With Go, bun, the Wails CLI, Podman and Task installed:</p>
    <pre><code>task dev
task seed</code></pre>
    <p>That starts Postgres, the API, the website and the desktop app. Sign in as <code>ada@bakery.test</code> with password <code>password</code>.</p>
  </Panel>

  <Panel title="How work is planned">
    <p>In phases: each one a short list of tasks, done in order, each checked before the next starts. What exists and what comes next is in the <a href="https://jevidocs.jevido.app/p/bakery">docs</a>. For anything bigger than a fix, open an issue first so we can agree where it fits.</p>
  </Panel>

  <Panel title="House rules">
    <ul>
      <li>The words in the glossary are the words in the code. A guild is a guild, never a team or a workspace.</li>
      <li>The API's parts (identity, guilds, boards) never reach into each other; each uses what the other offers.</li>
      <li>Docs change with the code, in the same change.</li>
      <li>Commits say what changed and where: <code>feat(web): …</code>, <code>fix(api): …</code>.</li>
    </ul>
  </Panel>
</div>

<p class="more">
  The full guide is <a href="https://github.com/jevido/the-bakery/blob/main/CONTRIBUTING.md">CONTRIBUTING.md</a>.
  Questions go to <a href="https://github.com/jevido/the-bakery/issues">GitHub issues</a>.
</p>

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

  .panels {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: var(--gap);
    align-items: start;
  }

  p {
    margin: 0 0 8px;
    color: var(--text-dim);
  }

  pre {
    margin: 0 0 8px;
    padding: 8px;
    overflow-x: auto;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  code {
    font-size: 13px;
    overflow-wrap: anywhere;
  }

  ul {
    margin: 0;
    padding-left: 18px;
    color: var(--text-dim);
  }

  li {
    margin-bottom: 6px;
  }

  .more {
    margin-top: 16px;
  }
</style>
