<script lang="ts">
  import { detectOS, latestRelease, OS_NAMES, type DesktopRelease } from '../lib/releases'

  // The download for the visitor's OS, or a way to run from source.
  let release = $state<DesktopRelease | null | undefined>(undefined)
  latestRelease().then((r) => (release = r))

  const os = detectOS(navigator.userAgent)
  let asset = $derived(release && os ? release.assets[os][0] : undefined)
</script>

{#if release === undefined}
  <span class="button disabled">Checking the supply depot…</span>
{:else if release}
  {#if asset && os}
    <a class="button" href={asset.url}>Download for {OS_NAMES[os]} · v{release.version}</a>
  {:else}
    <a class="button" href="/desktop">Get the desktop app</a>
  {/if}
  <a class="alt" href="https://jevidocs.jevido.app/p/bakery/getting-started">Run from source</a>
{:else}
  <a class="button" href="https://jevidocs.jevido.app/p/bakery/getting-started">Run it from source</a>
{/if}

<style>
  .button {
    display: inline-block;
    font-family: var(--font-display);
    font-size: 16px;
    color: var(--text);
    text-decoration: none;
    padding: 8px 18px;
    background: var(--olive);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    box-shadow: inset 0 1px 0 #ffffff14, var(--shadow-outer);
  }

  .button:hover {
    background: var(--olive-bright);
    color: var(--text);
  }

  .alt {
    margin-left: 20px;
  }

  .disabled {
    background: var(--panel-raised);
    color: var(--text-faint);
  }
</style>
