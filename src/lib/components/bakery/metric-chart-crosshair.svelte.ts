import { getContext, setContext } from 'svelte';

const KEY = Symbol('metric-chart-crosshair');

/**
 * Shared "which point index is hovered" state for every `MetricChart` on one
 * page -- provided once (e.g. Host Detail's Overview tab) so hovering any
 * one chart highlights the same timestamp across every other chart sharing
 * this context, without a heavier state library. Scoped per-page via
 * Svelte context, not a module-level global: unrelated charts on different
 * pages must never affect each other.
 */
export interface MetricChartCrosshair {
	hoveredIndex: number | null;
}

export function provideMetricChartCrosshair(): MetricChartCrosshair {
	const state = $state<MetricChartCrosshair>({ hoveredIndex: null });
	setContext(KEY, state);
	return state;
}

/** Returns `undefined` if no provider is above this component in the tree -- `MetricChart` falls back to purely local (unsynced) behavior in that case. */
export function useMetricChartCrosshair(): MetricChartCrosshair | undefined {
	return getContext<MetricChartCrosshair | undefined>(KEY);
}
