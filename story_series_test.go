package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func seriesFixture(t *testing.T, id string) StoryChallenge {
	t.Helper()
	_, c, err := findStoryChallenge(id)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestSeriesCampaignFormats(t *testing.T) {
	counts := map[int]int{}
	for _, city := range storyCities() {
		for _, c := range city.Challenges {
			counts[storyBestOf(c)]++
		}
	}
	if counts[1] != 18 || counts[3] != 5 || counts[5] != 1 {
		t.Fatal("Wrong series distribution", counts)
	}
	if storyBestOf(seriesFixture(t, "CHAMPIONSIAN-4")) != 5 {
		t.Fatal("Grand final must be BO5")
	}
}

func TestBO3RewardsAndUnlockOnlyOnSeriesVictory(t *testing.T) {
	s := newStorySave()
	c := seriesFixture(t, "INTRO-4")
	for _, id := range []string{"INTRO-1", "INTRO-2", "INTRO-3"} {
		s.Completed[id] = true
	}
	storyRecordGame(&s, c, 3, 0, 0, 0, 0, 10)
	if s.Credits != 0 || s.Completed[c.ID] || s.Series[c.ID].Wins != 1 || !storyHasOpenSeries(s) {
		t.Fatal("First game must only save series score", s)
	}
	if slices.Contains(unlockedStoryCities(s, storyCities()), "TRYHARD") {
		t.Fatal("Chapter unlocked before series victory")
	}
	storyRecordGame(&s, c, 1, 4, 0, 0, 0, 10)
	if s.Credits != 0 || s.Series[c.ID] != (StorySeries{1, 1}) {
		t.Fatal("Middle game must not award credits")
	}
	message := storyRecordGame(&s, c, 4, 1, 0, 0, 0, 10)
	if s.Credits != 100 || !s.Completed[c.ID] || s.Series[c.ID] != (StorySeries{2, 1}) || storyHasOpenSeries(s) || !strings.Contains(message, "won 2–1") {
		t.Fatal("Series victory not recorded correctly", message, s)
	}
	if !slices.Contains(unlockedStoryCities(s, storyCities()), "TRYHARD") {
		t.Fatal("Next chapter still locked")
	}
	if len(s.Attempts) != 3 || s.Attempts[0].Completed || !s.Attempts[2].Completed || s.Attempts[2].SeriesBestOf != 3 {
		t.Fatal("Round history lost series context")
	}
}

func TestBO5LossRetryAndReplay(t *testing.T) {
	s := newStorySave()
	c := seriesFixture(t, "CHAMPIONSIAN-4")
	for i := 0; i < 2; i++ {
		storyRecordGame(&s, c, 0, 3, 0, 0, 0, 10)
	}
	if s.Credits != 0 || !storyHasOpenSeries(s) {
		t.Fatal("BO5 must continue at 0–2")
	}
	storyRecordGame(&s, c, 0, 3, 0, 0, 0, 10)
	if s.Credits != 25 || s.Completed[c.ID] || storyHasOpenSeries(s) {
		t.Fatal("Lost series reward or completion incorrect")
	}
	storyRecordGame(&s, c, 3, 0, 0, 0, 0, 10)
	if s.Series[c.ID] != (StorySeries{1, 0}) || s.Credits != 25 {
		t.Fatal("Retry failed to start from zero")
	}
	storyRecordGame(&s, c, 3, 0, 0, 0, 0, 10)
	if s.Completed[c.ID] {
		t.Fatal("BO5 completed at two wins")
	}
	storyRecordGame(&s, c, 3, 0, 0, 0, 0, 10)
	if !s.Completed[c.ID] || s.Credits != 125 {
		t.Fatal("BO5 victory reward incorrect")
	}
	for i := 0; i < 3; i++ {
		storyRecordGame(&s, c, 3, 0, 0, 0, 0, 10)
	}
	if s.Credits != 150 || !s.Completed[c.ID] {
		t.Fatal("Replay should grant one return reward for entire series")
	}
}

func TestSeriesPersistsAndCanResetWithoutLosingCareer(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	s := newStorySave()
	c := seriesFixture(t, "INTRO-4")
	s.Completed["INTRO-1"] = true
	s.Credits = 85
	storyRecordGame(&s, c, 3, 0, 0, 0, 0, 10)
	if err := a.writeStory(s); err != nil {
		t.Fatal(err)
	}
	loaded, err := a.loadStory()
	if err != nil || loaded.Series[c.ID] != (StorySeries{1, 0}) {
		t.Fatal("Series did not survive reload", err)
	}
	if err = a.SetStoryProfile("solo", map[string]int{"1v1": 10, "2v2": 10, "3v3": 10}); err == nil {
		t.Fatal("Profile changed halfway through a series")
	}
	a.storyStatus.Active = true
	if err = a.ResetStorySeries(c.ID); err == nil {
		t.Fatal("Reset allowed during active game")
	}
	a.storyStatus.Active = false
	if err = a.ResetStorySeries(c.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err = a.loadStory()
	if err != nil || storyHasOpenSeries(loaded) || loaded.Credits != 85 || !loaded.Completed["INTRO-1"] {
		t.Fatal("Reset changed career history", err)
	}
}

func TestExistingSingleGamesAndCompletedFinalsRemainCompatible(t *testing.T) {
	s := newStorySave()
	c := seriesFixture(t, "INTRO-1")
	storyRecordGame(&s, c, 3, 0, 0, 0, 0, 10)
	if s.Credits != 100 || !s.Completed[c.ID] || len(s.Series) != 0 {
		t.Fatal("Single-game behavior changed")
	}
	s.Completed["INTRO-4"] = true
	s.Series = nil
	normalizeStorySave(&s)
	if !s.Completed["INTRO-4"] || s.Series == nil {
		t.Fatal("Older completed final was reset")
	}
}
