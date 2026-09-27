// Desktop releases come from GitHub: tags `desktop-vX.Y.Z` on
// jevido/the-bakery. This module finds the newest one and sorts its files by
// operating system.

export type OS = 'linux' | 'windows' | 'macos'

export type Asset = { name: string; url: string; size: number }

export type DesktopRelease = {
  version: string
  tag: string
  publishedAt: string
  notes: string
  url: string
  assets: Record<OS, Asset[]>
}

const RELEASES_URL = 'https://api.github.com/repos/jevido/the-bakery/releases?per_page=20'
const TAG_PREFIX = 'desktop-v'

type GitHubRelease = {
  tag_name: string
  draft: boolean
  prerelease: boolean
  published_at: string | null
  body: string | null
  html_url: string
  assets: { name: string; browser_download_url: string; size: number }[]
}

// osOf says which OS an installer file is for, by its name. Update
// manifests, checksums and signatures belong to no OS and are left out.
export function osOf(name: string): OS | null {
  const n = name.toLowerCase()
  if (n.endsWith('.appimage') || n.endsWith('.deb') || n.endsWith('.rpm')) return 'linux'
  if (n.endsWith('.exe') || n.endsWith('.msi')) return 'windows'
  if (n.endsWith('.dmg') || (n.endsWith('.zip') && /mac|darwin|\.app\.zip$/.test(n))) return 'macos'
  return null
}

// pickRelease returns the newest published desktop release, or null.
export function pickRelease(releases: GitHubRelease[]): DesktopRelease | null {
  const r = releases
    .filter((r) => !r.draft && r.tag_name.startsWith(TAG_PREFIX) && r.published_at)
    .sort((a, b) => (b.published_at ?? '').localeCompare(a.published_at ?? ''))[0]
  if (!r) return null
  const assets: Record<OS, Asset[]> = { linux: [], windows: [], macos: [] }
  for (const a of r.assets) {
    const os = osOf(a.name)
    if (os) assets[os].push({ name: a.name, url: a.browser_download_url, size: a.size })
  }
  // AppImage first on Linux: it runs on any distribution.
  assets.linux.sort((a, b) => Number(b.name.toLowerCase().endsWith('.appimage')) - Number(a.name.toLowerCase().endsWith('.appimage')))
  return {
    version: r.tag_name.slice(TAG_PREFIX.length),
    tag: r.tag_name,
    publishedAt: r.published_at!,
    notes: r.body ?? '',
    url: r.html_url,
    assets,
  }
}

export function detectOS(userAgent: string): OS | null {
  const ua = userAgent.toLowerCase()
  if (ua.includes('windows')) return 'windows'
  if (ua.includes('mac os') || ua.includes('macintosh')) return 'macos'
  if (ua.includes('linux') && !ua.includes('android')) return 'linux'
  return null
}

let cached: Promise<DesktopRelease | null> | undefined

// latestRelease fetches once per page view. A failed fetch counts as "no
// release" so the page falls back to running from source.
export function latestRelease(fetchFn: typeof fetch = fetch): Promise<DesktopRelease | null> {
  cached ??= fetchFn(RELEASES_URL, { headers: { Accept: 'application/vnd.github+json' } })
    .then((res) => (res.ok ? res.json() : []))
    .then((releases: GitHubRelease[]) => pickRelease(releases))
    .catch(() => null)
  return cached
}

export const OS_NAMES: Record<OS, string> = { linux: 'Linux', windows: 'Windows', macos: 'macOS' }
