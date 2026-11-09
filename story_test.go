package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestStoryCampaignAndProgression(t *testing.T) {
	cities := storyCities()
	count := 0
	ids := map[string]bool{}
	for _, city := range cities {
		for _, c := range city.Challenges {
			count++
			if ids[c.ID] {
				t.Fatal("Duplicate fixture")
			}
			ids[c.ID] = true
			if storyMapPackages[c.Map] == "" {
				t.Fatal("Unknown map", c.Map)
			}
			if c.DisabledBoost {
				t.Fatal("Season must start with normal boost")
			}
		}
	}
	if len(cities) != 7 || count != 24 {
		t.Fatalf("Expected 7 chapters and 24 fixtures, got %d and %d", len(cities), count)
	}
	save := newStorySave()
	if len(unlockedStoryCities(save, cities)) != 1 {
		t.Fatal("Only the first chapter should be open")
	}
	save.Completed["INTRO-1"] = true
	if slices.Contains(unlockedStoryCities(save, cities), "TRYHARD") {
		t.Fatal("Chapter unlocked before final")
	}
	for _, c := range cities[0].Challenges {
		save.Completed[c.ID] = true
	}
	if !slices.Contains(unlockedStoryCities(save, cities), "TRYHARD") {
		t.Fatal("Next chapter should open")
	}
	if slices.Contains(unlockedStoryCities(save, cities), "SUMMIT") {
		t.Fatal("Epilogue unlocked early")
	}
	for _, city := range cities {
		for _, c := range city.Challenges {
			save.Completed[c.ID] = true
		}
	}
	if len(unlockedStoryCities(save, cities)) != 7 {
		t.Fatal("Season cannot be finished")
	}
}
func TestStoryObjectives(t *testing.T) {
	zero := 0
	cases := []struct {
		name                         string
		c                            StoryConditions
		b, o, received, demos, goals int
		want                         bool
	}{
		{name: "win", b: 2, o: 1, want: true}, {name: "draw", b: 2, o: 2}, {name: "loss", b: 1, o: 2},
		{name: "margin pass", c: StoryConditions{ScoreDifference: 3}, b: 4, o: 1, want: true},
		{name: "margin fail", c: StoryConditions{ScoreDifference: 3}, b: 3, o: 1},
		{name: "no demos pass", c: StoryConditions{SelfDemoCount: &zero}, b: 2, o: 0, want: true},
		{name: "no demos fail", c: StoryConditions{SelfDemoCount: &zero}, b: 2, o: 0, received: 1},
		{name: "demo target", c: StoryConditions{DemoAchievedCount: 2}, b: 3, o: 0, demos: 2, want: true},
		{name: "demo short", c: StoryConditions{DemoAchievedCount: 2}, b: 3, o: 0, demos: 1},
		{name: "personal goal missing", c: StoryConditions{GoalsScored: 1}, b: 3, o: 0},
		{name: "personal goal", c: StoryConditions{GoalsScored: 1}, b: 3, o: 0, goals: 1, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := storyCompleted(StoryChallenge{Conditions: tc.c}, tc.b, tc.o, tc.received, tc.demos, tc.goals); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
func TestStoryPersistenceAndPurchases(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "story", "career.json")
	save := newStorySave()
	save.Credits = 20
	if err := a.writeStory(save); err != nil {
		t.Fatal(err)
	}
	if err := a.PurchaseStory("boost-33", false); err == nil {
		t.Fatal("Retired purchases must be rejected")
	}
	if err := a.RenameStoryClub("Test club"); err != nil {
		t.Fatal(err)
	}
	read, err := a.loadStory()
	if err != nil {
		t.Fatal(err)
	}
	if read.Club != "Test club" || read.Credits != 20 {
		t.Fatal("Unexpected save mutation")
	}
	if err = a.StartStory("CHAMPIONSIAN-4", nil, nil, "steam", ""); err == nil {
		t.Fatal("Locked fixture launched")
	}
	if err = os.WriteFile(a.storyPath(), []byte("broken save"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = a.loadStory(); err == nil {
		t.Fatal("Corrupt save accepted")
	}
	if err = a.RenameStoryClub("oops"); err == nil {
		t.Fatal("Corrupt save overwritten")
	}
	data, _ := os.ReadFile(a.storyPath())
	if string(data) != "broken save" {
		t.Fatal("Corrupt save lost")
	}
	var _ = json.Valid
}
func TestStoryPlayers(t *testing.T) {
	for _, id := range []string{"human", "rookie", "pro", "allstar"} {
		p, err := storyPlayer(id, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !json.Valid(p.Player) {
			t.Fatal("Invalid player JSON")
		}
		if p.ToPlayer() == nil {
			t.Fatal("Missing player config")
		}
	}
	if _, err := storyPlayer("missing", nil); err == nil {
		t.Fatal("Must report missing bot, never substitute silently")
	}
}
