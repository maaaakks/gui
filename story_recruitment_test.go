package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecruitsMatchPlayerReference(t *testing.T) {
	bots := []BotInfo{{Config: BotConfig{Settings: BotSettings{Name: "Kilobyte", AgentId: "SubparNova/KilobyteReduced", RunCommand: "kilobyte.exe"}}}}
	for _, recruit := range storyRoster {
		for _, rank := range []int{1, 10, 18, 22} {
			p, err := storyRecruitPlayer(recruit.ID, bots, rank)
			if err != nil {
				t.Fatal(err)
			}
			var bot BotInfo
			if err = json.Unmarshal(p.Player, &bot); err != nil {
				t.Fatal(err)
			}
			if p.Sort != "rlbot" || bot.Config.Settings.Name != recruit.Name || !strings.Contains(bot.Config.Settings.RunCommand, fmt.Sprintf("--skill=%d ", rank)) {
				t.Fatal("Teammate does not match playlist reference", recruit.ID, rank, bot.Config.Settings)
			}
		}
	}
	if bots[0].Config.Settings.Name != "Kilobyte" {
		t.Fatal("Installed metadata mutated")
	}
}

func TestRecruitPurchaseGatingAndSavedLineup(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	s := newStorySave()
	s.Credits = 300
	a.writeStory(s)
	if err := a.RecruitStory("allstar", nil); err == nil {
		t.Fatal("Locked recruit was available")
	}
	storyTestWins(&s, 4)
	a.writeStory(s)
	if err := a.RecruitStory("allstar", nil); err == nil {
		t.Fatal("Missing adjustable model accepted")
	}
	unchanged, _ := a.loadStory()
	if unchanged.Credits != 300 || len(unchanged.Recruits) != len(s.Recruits) {
		t.Fatal("Failed recruitment changed save")
	}
	botDir := t.TempDir()
	botConfig := "[settings]\nname = \"Kilobyte\"\nagent_id = \"SubparNova/KilobyteReduced\"\nrun_command = \"kilobyte.exe\"\n"
	if err := os.WriteFile(filepath.Join(botDir, "kilobyte.bot.toml"), []byte(botConfig), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.RecruitStory("allstar", []string{botDir}); err != nil {
		t.Fatal(err)
	}
	loaded, _ := a.loadStory()
	if loaded.Credits != 225 {
		t.Fatal("Wrong recruit cost", loaded.Credits)
	}
	if err := a.RecruitStory("allstar", nil); err == nil {
		t.Fatal("Duplicate recruit charged again")
	}
	if err := a.SetStoryLineup([]string{"rookie", "allstar"}); err != nil {
		t.Fatal(err)
	}
	loaded, _ = a.loadStory()
	if len(loaded.Lineup) != 2 || loaded.Lineup[1] != "allstar" {
		t.Fatal("Lineup did not persist")
	}
	if err := a.SetStoryLineup([]string{"rookie", "rookie"}); err == nil {
		t.Fatal("Duplicate driver in lineup")
	}
	if err := a.SetStoryLineup([]string{"Necto"}); err == nil {
		t.Fatal("Unrecruited driver selectable")
	}
	a.storyStatus.Active = true
	if err := a.SetStoryLineup([]string{"rookie"}); err == nil {
		t.Fatal("Lineup changed during game")
	}
	if err := a.RecruitStory("pro", nil); err == nil {
		t.Fatal("Recruiting during game allowed")
	}
}

func TestRivalsAndTrioRosterCompatibility(t *testing.T) {
	s := newStorySave()
	s.Format = "trio"
	normalizeStorySave(&s)
	if len(s.Recruits) < 2 {
		t.Fatal("Trio format needs a starter pair")
	}
	for _, city := range storyCitiesForSave(s) {
		for _, c := range city.Challenges {
			if len(c.OpponentNames) != len(c.OpponentBots) {
				t.Fatal("Missing opponent identity")
			}
			seen := map[string]bool{}
			for _, name := range c.OpponentNames {
				if name == "" || seen[name] || strings.Contains(name, "Kilobyte") {
					t.Fatal("Invalid rival identity")
				}
				seen[name] = true
			}
		}
	}
	s.Lineup = []string{"rookie", "rookie", "missing"}
	normalizeStorySave(&s)
	if len(s.Lineup) != 1 || s.Lineup[0] != "rookie" {
		t.Fatal("Invalid old lineup not normalized")
	}
}
