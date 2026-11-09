package main

import (
	"path/filepath"
	"testing"
)

func TestVanillaSuspendsAllGameplayModifiers(t *testing.T) {
	s := newStorySave()
	storyTestWins(&s, 8)
	s.Modules = map[string]int{"reserve": 3, "saves": 3, "shots": 3, "demos": 3}
	s.Equipped = []string{"reserve", "saves"}
	s.Vanilla = true
	r := newStoryRuntime(s, StoryChallenge{OpponentBoostCap: 80})
	r.Charge = 100
	r.activeTime = 30
	p := runtimePacket(1, 33)
	p.Players[0].ScoreInfo.Saves = 1
	p.Players[0].ScoreInfo.Shots = 1
	p.Players[0].ScoreInfo.Demolitions = 1
	if state := r.step(p, 0, true); state != nil {
		t.Fatal("Vanilla changed native boost, opponent boost, or Flow", state)
	}
	if r.Extra != 0 || r.capacity() != 100 || storyModuleValue(s, "saves") != 0 || r.Uses != 0 {
		t.Fatal("Equipment or Flow effect remained enabled")
	}
	p.MatchInfo.SecondsElapsed = 1.1
	p.Players[0].ScoreInfo.Saves++
	if state := r.step(p, 0, true); state != nil {
		t.Fatal("Later packet bypassed Vanilla guard")
	}
}
func TestVanillaSwitchPersistsAndPreservesLoadout(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	s.Credits = 85
	a.writeStory(s)
	if s.Vanilla {
		t.Fatal("Old campaigns must retain upgrades by default")
	}
	if err := a.SetStoryVanilla(true); err != nil {
		t.Fatal(err)
	}
	loaded, err := a.loadStory()
	if err != nil || !loaded.Vanilla || loaded.Credits != 85 || loaded.Modules["reserve"] != 1 || len(loaded.Equipped) != 1 {
		t.Fatal("Toggle lost career data", err)
	}
	a.storyStatus.Active = true
	a.storyStatus.FlowCharge = 100
	if err = a.SetStoryVanilla(false); err == nil {
		t.Fatal("Mode changed mid-game")
	}
	if err = a.TriggerStoryFlow(); err == nil {
		t.Fatal("Manual Flow bypassed Vanilla")
	}
	a.storyStatus.Active = false
	if err = a.SetStoryVanilla(false); err != nil {
		t.Fatal(err)
	}
	loaded, err = a.loadStory()
	if err != nil || loaded.Vanilla || storyModuleValue(loaded, "reserve") != 25 {
		t.Fatal("Original loadout was not restored", err)
	}
}
