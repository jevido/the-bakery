<script lang="ts">
  import Panel from './components/Panel.svelte'
</script>

<main>
  <Panel title="First Colony — Colony log">
    <p>Day 1 of Aprimay. The colonists wake beside their crashed pod, with little more than a plan.</p>
  </Panel>
</main>

<style>
  main {
    height: 100%;
    padding: 16px;
    display: grid;
    place-items: start center;
  }

  main :global(.panel) {
    width: min(640px, 100%);
  }

  p {
    margin: 0;
    color: var(--text-dim);
  }
</style>
