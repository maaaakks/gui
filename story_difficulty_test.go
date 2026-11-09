package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoryDifficultyMigrationAndSaving(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	old := newStorySave()
	if storySkill(old) != 7 {
		t.Fatal("Existing saves should default to Gold I")
	}
	if err := a.writeStory(old); err != nil {
		t.Fatal(err)
	}
	for _, skill := range []int{0, 1, 7, 16, 22} {
		if err := a.SetStoryDifficulty(skill); err != nil {
			t.Fatal(err)
		}
		save, err := a.loadStory()
		if err != nil {
			t.Fatal(err)
		}
		if storySkill(save) != skill {
			t.Fatalf("Difficulty not saved: %d", skill)
		}
	}
	for _, skill := range []int{-1, 23} {
		if err := a.SetStoryDifficulty(skill); err == nil {
			t.Fatal("Invalid difficulty accepted")
		}
	}
	a.storyStatus.Active = true
	if err := a.SetStoryDifficulty(3); err == nil {
		t.Fatal("Difficulty must be fixed during a match")
	}
	save, _ := a.loadStory()
	if storySkill(save) != 22 {
		t.Fatal("Rejected change altered the save")
	}
}

func TestStoryKilobyteLaunchConfiguration(t *testing.T) {
	source := BotInfo{Config: BotConfig{Settings: BotSettings{
		Name: "Kilobyte[Champ I]", AgentId: "SubparNova/KilobyteReduced", Hivemind: true,
		RunCommand: `x86_64-pc-windows-msvc\GGLBot.exe`, RunCommandLinux: "x86_64-unknown-linux-gnu/GGLBot",
	}}}
	for _, skill := range []int{0, 1, 7, 22} {
		player, err := storyKilobytePlayer([]BotInfo{source}, skill)
		if err != nil {
			t.Fatal(err)
		}
		var bot BotInfo
		if err = json.Unmarshal(player.Player, &bot); err != nil {
			t.Fatal(err)
		}
		if player.Sort != "rlbot" || !strings.Contains(bot.Config.Settings.RunCommand, " --no-ui") || !strings.Contains(bot.Config.Settings.RunCommandLinux, " --skill=") {
			t.Fatal(bot.Config.Settings)
		}
		if bot.Config.Settings.Name != "Kilobyte["+storySkillLabel(skill)+"]" {
			t.Fatal("Wrong displayed rank")
		}
		config := player.ToPlayer().ToPlayerConfig(1)
		if config.Team != 1 {
			t.Fatal("Wrong team")
		}
	}
	if strings.Contains(source.Config.Settings.RunCommand, "--skill") {
		t.Fatal("Installed bot configuration was mutated")
	}
	if _, err := storyKilobytePlayer(nil, 7); err == nil {
		t.Fatal("Missing bot should not be substituted silently")
	}
	if _, err := storyKilobytePlayer([]BotInfo{source}, 23); err == nil {
		t.Fatal("Invalid skill")
	}
	source.Config.Settings.RunCommand += " --skill=3"
	if _, err := storyKilobytePlayer([]BotInfo{source}, 7); err == nil {
		t.Fatal("Duplicate skill options")
	}
}
