// The desktop's own settings (settings.toml in the config folder, through
// SettingsService). quiet turns every flavor element off: flavor lines,
// mood, event letters, sounds and idle animations. Portraits, alerts and
// letters that ask something stay.

import { SettingsService } from './bindings'

export const settings = $state({ quiet: false, sound: false, volume: 0.6, loaded: false })

export async function loadSettings() {
  try {
    const s = await SettingsService.Get()
    settings.quiet = s.quiet_colony
    settings.sound = s.sound
    settings.volume = s.volume
  } catch {
    // Unreadable settings: the defaults stand.
  }
  settings.loaded = true
}

export async function saveSettings() {
  await SettingsService.Save({ quiet_colony: settings.quiet, sound: settings.sound, volume: settings.volume })
}
