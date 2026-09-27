import { describe, expect, test } from 'bun:test'
import { detectOS, osOf, pickRelease } from './releases'

const asset = (name: string) => ({ name, browser_download_url: `https://example.test/${name}`, size: 1 })

const fixture = [
  {
    tag_name: 'desktop-v0.2.0',
    draft: true,
    prerelease: false,
    published_at: null,
    body: 'draft',
    html_url: 'https://github.com/jevido/the-bakery/releases/tag/desktop-v0.2.0',
    assets: [asset('the-bakery-0.2.0.AppImage')],
  },
  {
    tag_name: 'desktop-v0.1.1',
    draft: false,
    prerelease: false,
    published_at: '2026-10-02T10:00:00Z',
    body: 'Fixes',
    html_url: 'https://github.com/jevido/the-bakery/releases/tag/desktop-v0.1.1',
    assets: [
      asset('the-bakery_0.1.1_amd64.deb'),
      asset('the-bakery-x86_64.AppImage'),
      asset('the-bakery-amd64-installer.exe'),
      asset('the-bakery-universal.dmg'),
      asset('update.json'),
      asset('the-bakery-x86_64.AppImage.sig'),
    ],
  },
  {
    tag_name: 'desktop-v0.1.0',
    draft: false,
    prerelease: false,
    published_at: '2026-10-01T10:00:00Z',
    body: '',
    html_url: 'https://github.com/jevido/the-bakery/releases/tag/desktop-v0.1.0',
    assets: [asset('the-bakery-old.AppImage')],
  },
  {
    tag_name: 'api-v9.9.9',
    draft: false,
    prerelease: false,
    published_at: '2027-01-01T00:00:00Z',
    body: '',
    html_url: 'https://github.com/jevido/the-bakery/releases/tag/api-v9.9.9',
    assets: [asset('ignored.AppImage')],
  },
]

describe('pickRelease', () => {
  test('newest published desktop release, files sorted by OS', () => {
    const r = pickRelease(fixture)!
    expect(r.version).toBe('0.1.1')
    expect(r.assets.linux.map((a) => a.name)).toEqual(['the-bakery-x86_64.AppImage', 'the-bakery_0.1.1_amd64.deb'])
    expect(r.assets.windows.map((a) => a.name)).toEqual(['the-bakery-amd64-installer.exe'])
    expect(r.assets.macos.map((a) => a.name)).toEqual(['the-bakery-universal.dmg'])
  })

  test('no desktop release', () => {
    expect(pickRelease([])).toBeNull()
    expect(pickRelease([fixture[0], fixture[3]])).toBeNull()
  })
})

test('osOf ignores manifests and signatures', () => {
  expect(osOf('update.json')).toBeNull()
  expect(osOf('the-bakery-x86_64.AppImage.sig')).toBeNull()
  expect(osOf('The-Bakery-mac.zip')).toBe('macos')
  expect(osOf('sources.zip')).toBeNull()
})

test('detectOS', () => {
  expect(detectOS('Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36')).toBe('linux')
  expect(detectOS('Mozilla/5.0 (Windows NT 10.0; Win64; x64)')).toBe('windows')
  expect(detectOS('Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0)')).toBe('macos')
  expect(detectOS('Mozilla/5.0 (Linux; Android 14)')).toBeNull()
})
