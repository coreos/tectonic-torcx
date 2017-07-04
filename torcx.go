package main

import "fmt"

func (a *App) InstallDocker(version string) error {
	return fmt.Errorf("not implemented")
}

// FetchTorcxAddon fetches a given torcx addon in to the user store
func (a *App) FetchTorcxAddon(name string, reference string) error {
	// XXX implement
	return nil
}

// UseAddon selects the addon for installation on next boot
func (a *App) UseAddon(name string, reference string) error {
	// XXX implement
	return nil
}

// torcxCmd executes a torcx command
func (a *App) torcxCmd(args []string) error {
	return nil
}
