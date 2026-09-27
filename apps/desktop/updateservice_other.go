//go:build !linux

package main

import "errors"

// replaceAppImage only exists on Linux; elsewhere APPIMAGE is never set.
func (s *UpdateService) replaceAppImage(target, staged string) error {
	return errors.New("AppImage updates are Linux only")
}
