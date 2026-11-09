package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/RLBot/go-interface/flat"
)

func TestStoryCarCatalogueUsesRealBodyProducts(t *testing.T) {
	f, err := os.Open("frontend/src/assets/items.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true // The game's item dump includes literal quotes in names.
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	bodies := map[uint32]string{}
	for _, row := range rows {
		if len(row) >= 4 && row[1] == "Body" {
			id, err := strconv.ParseUint(row[0], 10, 32)
			if err != nil {
				t.Fatal(err)
			}
			bodies[uint32(id)] = row[3]
		}
	}
	for _, car := range storyOctaneCars {
		if bodies[car.ID] != car.Name {
			t.Fatalf("Body mismatch for %s (%d): %s", car.Name, car.ID, bodies[car.ID])
		}
	}
	for _, recruit := range storyRecruitmentRoster() {
		car, ok := storyCarForDriver(recruit.Name)
		if !ok || recruit.CarName != car.Name {
			t.Fatal("Recruit is missing car identity", recruit.Name)
		}
	}
	for _, name := range storyRivals {
		if _, ok := storyCarForDriver(name); !ok {
			t.Fatal("Rival is missing car identity", name)
		}
	}
}

func TestStoryCarsReachBothTeamsWithoutMutatingInstalledLoadout(t *testing.T) {
	original := &LoadoutConfig{Blue: TeamLoadoutConfig{CarId: 23, DecalId: 11377, WheelsId: 4296, TeamColorId: 54, Paint: TeamPaintConfig{CarPaintId: 3}}, Orange: TeamLoadoutConfig{CarId: 23, DecalId: 11377, TeamColorId: 34}}
	bytes, _ := json.Marshal(BotInfo{Config: BotConfig{Settings: BotSettings{Name: "Kilobyte"}}, Loadout: original})
	player := PlayerJs{Sort: "rlbot", Player: bytes}
	for _, name := range []string{"Morgan Vale", "Casey Reed", "Viktor Rook", "Aria Knox"} {
		p, err := storyNamedPlayer(player, name)
		if err != nil {
			t.Fatal(err)
		}
		var bot BotInfo
		if err := json.Unmarshal(p.Player, &bot); err != nil {
			t.Fatal(err)
		}
		car, _ := storyCarForDriver(name)
		for _, team := range []uint32{0, 1} {
			config := bot.ToPlayerConfig(team)
			custom := config.Variety.Value.(*flat.CustomBotT)
			if custom.Loadout == nil || custom.Loadout.CarId != car.ID || custom.Loadout.DecalId != 0 {
				t.Fatal("Assigned body missing from launch config", name, team)
			}
		}
		if bot.Loadout.Blue.WheelsId != 4296 || bot.Loadout.Blue.TeamColorId != 54 || bot.Loadout.Orange.TeamColorId != 34 {
			t.Fatal("Unrelated cosmetics changed")
		}
	}
	if original.Blue.CarId != 23 || original.Blue.DecalId != 11377 || original.Blue.Paint.CarPaintId != 3 {
		t.Fatal("Installed bot loadout mutated")
	}
	copy := storyDriverLoadout(original, "Casey Reed")
	if copy == original || original.Blue.CarId != 23 {
		t.Fatal("Shared loadout mutated")
	}
	if storyDriverLoadout(original, "Unrelated bot") != original {
		t.Fatal("Non-story loadout changed")
	}
	human := PlayerJs{Sort: "human", Player: json.RawMessage(`{}`)}
	got, err := storyNamedPlayer(human, "Morgan Vale")
	if err != nil || string(got.Player) != string(human.Player) || got.Sort != "human" {
		t.Fatal("Human loadout changed")
	}
}
