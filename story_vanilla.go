package main

import "errors"

// Vanilla keeps career progression and loadouts, while suspending gameplay modifiers.
func (a *App) SetStoryVanilla(enabled bool) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Change Vanilla mode between games")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	save.Vanilla = enabled
	return a.writeStory(save)
}
