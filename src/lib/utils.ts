import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

/** e.g. `1536` -> `"1.5 KB"`. Used for host-reported volume/disk sizes. */
export function formatBytes(bytes: number): string {
	if (bytes <= 0) return '0 B';
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	const exp = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
	const value = bytes / 1024 ** exp;
	return `${exp === 0 ? value : value.toFixed(1)} ${units[exp]}`;
}

/** e.g. `"Jane Doe"` -> `"JD"`. Used anywhere a user's real name needs a compact avatar fallback. */
export function initialsFrom(name: string): string {
	return name
		.trim()
		.split(/\s+/)
		.slice(0, 2)
		.map((part) => part[0])
		.join('')
		.toUpperCase();
}

/** e.g. `754_000` -> `"12m 34s"`. Used for build durations and deployment uptime. */
export function formatDuration(ms: number): string {
	const seconds = Math.max(0, Math.floor(ms / 1000));
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	const remSeconds = seconds % 60;
	if (minutes < 60) return remSeconds > 0 ? `${minutes}m ${remSeconds}s` : `${minutes}m`;
	const hours = Math.floor(minutes / 60);
	const remMinutes = minutes % 60;
	if (hours < 24) return remMinutes > 0 ? `${hours}h ${remMinutes}m` : `${hours}h`;
	const days = Math.floor(hours / 24);
	const remHours = hours % 24;
	return remHours > 0 ? `${days}d ${remHours}h` : `${days}d`;
}

/** e.g. a Date 2 minutes ago -> `"2m ago"`. Used by the overview dashboard's activity feed. */
export function formatRelativeTime(date: Date): string {
	const seconds = Math.max(0, Math.floor((Date.now() - date.getTime()) / 1000));
	if (seconds < 60) return 'just now';
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m ago`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.floor(hours / 24);
	return `${days}d ago`;
}

// Mirrors the `deployment_status` Postgres enum (bakery.schema.ts) -- kept as
// a local literal union rather than importing the schema's own type, since
// this file is loaded client-side and `src/lib/server/*` is server-only.
export type DeploymentStatus =
	| 'starting_new'
	| 'health_checking'
	| 'flipping_proxy'
	| 'stopping_old'
	| 'running'
	| 'failed'
	| 'rolled_back'
	| 'stopped';

/**
 * Reduces every deployment status to exactly one of three colors: green only
 * for `running`, red only for `failed`, grey for everything else -- an
 * operator only needs to know "is it live," "did it crash," or neither, not
 * which of the four in-progress rollout steps it's currently on.
 * `pulse: true` for the four non-terminal statuses keeps an in-progress
 * deploy visually distinct from a merely `stopped` one despite both being
 * grey.
 */
export function deploymentStatusColor(status: DeploymentStatus): {
	color: string;
	pulse: boolean;
} {
	if (status === 'running') return { color: '#52cc96', pulse: false };
	if (status === 'failed') return { color: '#f0836b', pulse: false };
	const pulse =
		status === 'starting_new' ||
		status === 'health_checking' ||
		status === 'flipping_proxy' ||
		status === 'stopping_old';
	return { color: 'var(--tx-3)', pulse };
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChild<T> = T extends { child?: any } ? Omit<T, 'child'> : T;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChildren<T> = T extends { children?: any } ? Omit<T, 'children'> : T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };
