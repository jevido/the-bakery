<script lang="ts">
  import { Panel } from '@bakery/ui'
  import { detectOS, latestRelease, OS_NAMES, type DesktopRelease, type OS } from '../lib/releases'

  let release = $state<DesktopRelease | null | undefined>(undefined)
  latestRelease().then((r) => (release = r))

  const detected = detectOS(navigator.userAgent)
  // The visitor's OS first, then the others.
  const order: OS[] = [detected, ...(['linux', 'windows', 'macos'] as OS[]).filter((o) => o !== detected)].filter(
    (o): o is OS => o !== null,
  )

  function size(bytes: number): string {
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  }

  function date(iso: string): string {
    return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
  }
</script>

<svelte:head>
  <title>Desktop app — The Bakery</title>
  <meta name="description" content="Download The Bakery's desktop app for Linux, Windows or macOS." />
</svelte:head>

<h1>The desktop app</h1>

{#if release === undefined}
  <p class="dim">Checking the supply depot…</p>
{:else if release === null}
  <Panel title="No builds yet — run from source">
    <p>The first release has not landed. Until it does, run The Bakery from source: clone the repository and run <code>task dev</code>.</p>
    <p><a href="https://jevidocs.jevido.app/p/bakery/getting-started">Getting started</a></p>
  </Panel>
{:else}
  <p class="dim">Version {release.version}, released {date(release.publishedAt)}.</p>
  <div class="downloads">
    {#each order as os (os)}
      <Panel title={os === detected ? `${OS_NAMES[os]} · your system` : OS_NAMES[os]}>
        {#if release.assets[os].length === 0}
          <p class="dim">No build for {OS_NAMES[os]} in this release.</p>
        {:else}
          <ul>
            {#each release.assets[os] as asset (asset.name)}
              <li><a href={asset.url}>{asset.name}</a> <span class="dim">{size(asset.size)}</span></li>
            {/each}
          </ul>
        {/if}
        {#if os === 'linux' && release.assets.linux.length > 0}
          <p class="dim small">
            Needs WebKitGTK 6.0 from your distribution: <code>webkitgtk-6.0</code> on Arch and Fedora,
            <code>libwebkitgtk-6.0-4</code> on Debian and Ubuntu. The AppImage updates itself.
          </p>
        {/if}
        {#if os === 'macos' && release.assets.macos.length > 0}
          <p class="dim small">The macOS build is not signed yet: open it with right-click → Open the first time.</p>
        {/if}
      </Panel>
    {/each}
  </div>
  {#if release.notes}
    <Panel title="What changed">
      <pre class="notes">{release.notes}</pre>
      <p><a href={release.url}>Release on GitHub</a></p>
    </Panel>
  {/if}
{/if}

<style>
  h1 {
    margin: 0 0 8px;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 32px;
  }

  .downloads {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: var(--gap);
    margin: 16px 0;
  }

  ul {
    margin: 0;
    padding-left: 18px;
  }

  li {
    overflow-wrap: anywhere;
    margin-bottom: 4px;
  }

  p {
    margin: 0 0 8px;
  }

  .dim {
    color: var(--text-dim);
  }

  .small {
    font-size: 12px;
    margin-top: 8px;
  }

  code {
    padding: 0 3px;
    background: var(--panel-inset);
    border-radius: var(--radius);
  }

  .notes {
    margin: 0 0 8px;
    white-space: pre-wrap;
    font: inherit;
    color: var(--text-dim);
  }
</style>
