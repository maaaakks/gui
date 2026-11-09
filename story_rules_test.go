package main

import (
	"github.com/RLBot/go-interface/flat"
	"path/filepath"
	"testing"
)

func TestStoryThreeGoalLead(t *testing.T) {
	for _, tc := range []struct {
		blue, orange int
		want         bool
	}{{3, 0, true}, {4, 1, true}, {5, 2, true}, {0, 3, true}, {1, 4, true}, {3, 2, false}, {2, 3, false}, {2, 0, false}, {0, 0, false}} {
		if storyEarlyFinish(true, tc.blue, tc.orange) != tc.want {
			t.Fatalf("Wrong result at %d–%d", tc.blue, tc.orange)
		}
		if storyEarlyFinish(false, tc.blue, tc.orange) {
			t.Fatal("Disabled rule triggered")
		}
	}
	c := StoryChallenge{MaxScore: "3 Goals"}
	if storyMatchMutators(c, true).MaxScore != flat.MaxScoreMutatorUnlimited {
		t.Fatal("3–2 must not end because of a first-to-three ceiling")
	}
	if storyMatchMutators(c, false).MaxScore != flat.MaxScoreMutatorThreeGoals {
		t.Fatal("Original cap should remain when option disabled")
	}
	c.DisabledBoost = true
	if storyMatchMutators(c, true).BoostAmount != flat.BoostAmountMutatorNoBoost {
		t.Fatal("Boost rules should be unchanged")
	}
	// Ending early does not bypass additional objectives.
	c.Conditions = StoryConditions{DemoAchievedCount: 2}
	if storyCompleted(c, 3, 0, 0, 1, 0) {
		t.Fatal("Missing objective must prevent rewards")
	}
	if storyCompleted(c, 0, 3, 0, 3, 0) {
		t.Fatal("Loss must not grant rewards")
	}
}
func TestStoryThreeGoalPreference(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	if !storyThreeGoalLead(newStorySave()) {
		t.Fatal("Default should be on, including existing saves")
	}
	for _, enabled := range []bool{false, true} {
		if err := a.SetStoryThreeGoalLead(enabled); err != nil {
			t.Fatal(err)
		}
		save, err := a.loadStory()
		if err != nil {
			t.Fatal(err)
		}
		if storyThreeGoalLead(save) != enabled {
			t.Fatal("Preference did not persist")
		}
	}
	a.storyStatus.Active = true
	if err := a.SetStoryThreeGoalLead(false); err == nil {
		t.Fatal("Rule cannot change midway through a match")
	}
}
