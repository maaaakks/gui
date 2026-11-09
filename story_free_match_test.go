package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RLBot/go-interface/flat"
)

func freeMatchTestBots() []BotInfo {
	return []BotInfo{{Config: BotConfig{Settings: BotSettings{Name: "Kilobyte", AgentId: "SubparNova/KilobyteReduced", RunCommand: "kilobyte.exe", Hivemind: true}}}}
}

func TestFreeMatchUsesEachSlotRankWithStandardRules(t *testing.T) {
	for size := 1; size <= 3; size++ {
		o := StoryFreeMatchOptions{Size: size, Arena: "BeckwithPark", BlueRanks: []int{1, 22}[:size-1], OrangeRanks: []int{16, 10, 7}[:size]}
		bots := freeMatchTestBots()
		c, err := storyFreeMatchConfig(o, bots, "steam", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(c.BluePlayers) != size || len(c.OrangePlayers) != size || c.BluePlayers[0].Sort != "human" {
			t.Fatal("Incorrect teams", size, c)
		}
		seenAgents := map[string]bool{}
		for side, players := range [][]PlayerJs{c.BluePlayers[1:], c.OrangePlayers} {
			for i, p := range players {
				var bot BotInfo
				if err = json.Unmarshal(p.Player, &bot); err != nil {
					t.Fatal(err)
				}
				ranks := [][]int{o.BlueRanks, o.OrangeRanks}[side]
				if !strings.Contains(bot.Config.Settings.RunCommand, fmt.Sprintf(" --skill=%d ", ranks[i])) || bot.Loadout == nil {
					t.Fatal("Slot lost its rank or car", side, i, bot)
				}
				if bot.Config.Settings.Hivemind || seenAgents[bot.Config.Settings.AgentId] || bot.Config.Settings.AgentId == "SubparNova/KilobyteReduced" {
					t.Fatal("Bots share a rank reservation", side, i, bot.Config.Settings)
				}
				seenAgents[bot.Config.Settings.AgentId] = true
			}
		}
		if c.ExtraOptions.EnableStateSetting || c.ExtraOptions.Freeplay || len(c.Scripts) != 0 || c.MutatorSettings != (flat.MutatorSettingsT{}) {
			t.Fatal("Career modifiers leaked into free match")
		}
		if bots[0].Config.Settings.RunCommand != "kilobyte.exe" {
			t.Fatal("Installed bot changed")
		}
	}
}

func TestFreeMatchRejectsInvalidSetup(t *testing.T) {
	valid := StoryFreeMatchOptions{Size: 2, Arena: "Mannfield", BlueRanks: []int{10}, OrangeRanks: []int{16, 22}}
	for _, change := range []func(*StoryFreeMatchOptions){
		func(o *StoryFreeMatchOptions) { o.BlueDrivers = []string{"missing"} },
		func(o *StoryFreeMatchOptions) { o.OrangeDrivers = []string{"rookie", "rookie"} },
		func(o *StoryFreeMatchOptions) {
			o.BlueDrivers = []string{"pro"}
			o.OrangeDrivers = []string{"pro", "Fury"}
		},
		func(o *StoryFreeMatchOptions) { o.BlueDrivers = []string{"rookie", "pro"} },
		func(o *StoryFreeMatchOptions) { o.Size = 4 }, func(o *StoryFreeMatchOptions) { o.BlueRanks = nil },
		func(o *StoryFreeMatchOptions) { o.OrangeRanks = []int{10} }, func(o *StoryFreeMatchOptions) { o.OrangeRanks = []int{0, 10} },
		func(o *StoryFreeMatchOptions) { o.BlueRanks = []int{23} }, func(o *StoryFreeMatchOptions) { o.Arena = "untrusted-map" },
	} {
		o := valid
		change(&o)
		if _, err := storyFreeMatchConfig(o, freeMatchTestBots(), "steam", ""); err == nil {
			t.Fatal("Invalid setup accepted", o)
		}
	}
	if _, err := storyFreeMatchConfig(valid, freeMatchTestBots(), "", ""); err == nil {
		t.Fatal("Missing launcher accepted")
	}
	if _, err := storyFreeMatchConfig(valid, nil, "steam", ""); err == nil {
		t.Fatal("Missing adjustable bot accepted")
	}
}

func TestFreeMatchCharacterIsIndependentFromRankAndRecruitment(t *testing.T) {
	o := StoryFreeMatchOptions{Size: 2, Arena: "Mannfield", BlueRanks: []int{10}, OrangeRanks: []int{16, 22}, BlueDrivers: []string{"Kira"}, OrangeDrivers: []string{"allstar", "Necto"}}
	c, err := storyFreeMatchConfig(o, freeMatchTestBots(), "steam", "")
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range []PlayerJs{c.BluePlayers[1], c.OrangePlayers[0], c.OrangePlayers[1]} {
		var bot BotInfo
		if err = json.Unmarshal(p.Player, &bot); err != nil {
			t.Fatal(err)
		}
		name := []string{"Mara Voss", "Jamie Cole", "Avery Stone"}[i]
		rank := []int{10, 16, 22}[i]
		car, _ := storyCarForDriver(name)
		if bot.Config.Settings.Name != name || bot.Loadout.Blue.CarId != car.ID || bot.Loadout.Orange.CarId != car.ID || !strings.Contains(bot.Config.Settings.RunCommand, fmt.Sprintf(" --skill=%d ", rank)) {
			t.Fatal("Character, car or rank selection lost", bot)
		}
	}
}

func TestFreeMatchFailureDoesNotCreateCareerOrKeepLock(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	o := StoryFreeMatchOptions{Size: 1, Arena: "Mannfield", OrangeRanks: []int{16}}
	if err := a.StartStoryFreeMatch(o, nil, "steam", ""); err == nil {
		t.Fatal("Missing bot accepted")
	}
	if a.storyStatus.Active {
		t.Fatal("Failed setup left match locked")
	}
	if _, err := os.Stat(a.storySavePath); !os.IsNotExist(err) {
		t.Fatal("Free match touched career save", err)
	}
	a.storyStatus = StoryStatus{Active: true, ChallengeID: "INTRO-1"}
	if err := a.StartStoryFreeMatch(o, nil, "steam", ""); err == nil || a.storyStatus.ChallengeID != "INTRO-1" {
		t.Fatal("Free match interrupted campaign")
	}
	a.storyStatus = StoryStatus{Active: true, FreeMatch: true, FlowCharge: 100}
	if err := a.TriggerStoryFlow(); err == nil {
		t.Fatal("Flow enabled in free match")
	}
}
