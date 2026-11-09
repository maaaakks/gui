package main

import (
	"github.com/RLBot/go-interface/flat"
	"os"
	"path/filepath"
	"testing"
)

func storyTestWins(save *StorySave, n int) {
	i := 0
	for _, city := range storyCities() {
		for _, c := range city.Challenges {
			if i < n {
				save.Completed[c.ID] = true
			}
			i++
		}
	}
	normalizeStorySave(save)
}
func TestSeasonMigration(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	legacy := `{"version":1,"club":"Boost FC","credits":3,"recruits":["rookie"],"upgrades":["boost-33","boost-100"],"completed":{"INTRO-1":true},"attempts":[]}`
	if err := os.WriteFile(a.storyPath(), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := a.loadStory()
	if err != nil {
		t.Fatal(err)
	}
	if s.Version != 2 || s.Credits != 450 || !s.Completed["INTRO-1"] || s.Club != "Boost FC" || len(s.Upgrades) != 0 {
		t.Fatal("Migration lost data", s)
	}
	if err = a.writeStory(s); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(a.storyPath() + ".before-season-v2.json")
	if err != nil || string(backup) != legacy {
		t.Fatal("Original save not preserved")
	}
	s, err = a.loadStory()
	if err != nil || s.Credits != 450 {
		t.Fatal("Migration applied twice")
	}
}
func TestSeasonFormatsAndRanks(t *testing.T) {
	s := newStorySave()
	s.Ranks = map[string]int{"1v1": 16, "2v2": 10, "3v3": 7}
	for _, format := range []string{"ascension", "solo", "duo", "trio"} {
		s.Format = format
		cities := storyCitiesForSave(s)
		for _, city := range cities {
			for _, c := range city.Challenges {
				expected := map[string]int{"solo": 1, "duo": 2, "trio": 3}[format]
				if expected > 0 && c.HumanTeamSize != expected {
					t.Fatal("Wrong format")
				}
				if len(c.OpponentBots) != c.HumanTeamSize {
					t.Fatal("Unbalanced default roster")
				}
				if c.EnemyRank < 1 || c.EnemyRank > 22 {
					t.Fatal("Invalid rank")
				}
			}
		}
	}
	s.Format = "ascension"
	cities := storyCitiesForSave(s)
	if cities[0].Challenges[0].EnemyRank != 14 || cities[1].Challenges[0].EnemyRank != 9 || cities[3].Challenges[0].EnemyRank != 7 {
		t.Fatal("Playlist rank not used")
	}
}
func TestSeasonWorkshop(t *testing.T) {
	a := NewApp()
	a.storySavePath = filepath.Join(t.TempDir(), "career.json")
	s := newStorySave()
	s.Credits = 1000
	a.writeStory(s)
	if err := a.BuyStoryModule("saves"); err == nil {
		t.Fatal("Early unlock")
	}
	storyTestWins(&s, 2)
	a.writeStory(s)
	if err := a.BuyStoryModule("saves"); err != nil {
		t.Fatal(err)
	}
	s, _ = a.loadStory()
	if s.Credits != 1000 || s.Modules["saves"] != 1 || !s.StarterClaimed || len(s.Equipped) != 1 {
		t.Fatal("Free module not granted")
	}
	if err := a.BuyStoryModule("shots"); err != nil {
		t.Fatal(err)
	}
	if err := a.EquipStoryModule("shots"); err == nil {
		t.Fatal("Slot limit bypassed")
	}
	if err := a.BuyStoryModule("saves"); err == nil {
		t.Fatal("Tier lock bypassed")
	}
	if err := a.BuyStoryModule("respawn"); err == nil {
		t.Fatal("Unverified effect sold")
	}
	s, _ = a.loadStory()
	storyTestWins(&s, 8)
	a.writeStory(s)
	if err := a.BuyStoryModule("saves"); err != nil {
		t.Fatal(err)
	}
	if err := a.EquipStoryModule("shots"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetStoryFlow("overdrive", true); err != nil {
		t.Fatal(err)
	}
	if err := a.SetStoryFlow("focus", false); err == nil {
		t.Fatal("Unverified flow mode enabled")
	}
	s, _ = a.loadStory()
	before := s.Credits
	c := storyCities()[2].Challenges[0]
	if reward := storyResultReward(&s, c, true); reward != 100 || s.Credits != before+100 {
		t.Fatal("Wrong first-win reward")
	}
	if reward := storyResultReward(&s, c, true); reward != 25 {
		t.Fatal("Repeat farming reward")
	}
}
func runtimePacket(seconds float32, boost float32) *flat.GamePacketT {
	return &flat.GamePacketT{MatchInfo: &flat.MatchInfoT{SecondsElapsed: seconds, MatchPhase: flat.MatchPhaseActive}, Players: []*flat.PlayerInfoT{{Team: 0, Boost: boost, ScoreInfo: &flat.ScoreInfoT{}, Physics: &flat.PhysicsT{Location: &flat.Vector3T{}}, LatestTouch: &flat.TouchT{}}, {Team: 0, IsBot: true, Boost: 100, Physics: &flat.PhysicsT{Location: &flat.Vector3T{X: 1500}}, LatestTouch: &flat.TouchT{}}, {Team: 1, IsBot: true, Boost: 100, Physics: &flat.PhysicsT{Location: &flat.Vector3T{X: 3000}}, LatestTouch: &flat.TouchT{}}}}
}
func TestEquipmentRuntime(t *testing.T) {
	s := newStorySave()
	s.Modules = map[string]int{"saves": 3, "passing": 3, "demos": 3}
	s.Equipped = []string{"saves", "passing"}
	r := newStoryRuntime(s, StoryChallenge{OpponentBoostCap: 80})
	p := runtimePacket(1, 33)
	state := r.step(p, 0, false)
	if state.CarStates[0].BoostAmount.Val != 100 || state.CarStates[2].BoostAmount.Val != 80 {
		t.Fatal("Initial boost or opponent handicap missing")
	}
	p = runtimePacket(1.1, 30)
	p.Players[0].ScoreInfo.Saves = 1
	state = r.step(p, 0, false)
	if state.CarStates[0].BoostAmount.Val != 55 {
		t.Fatal("Save reward missing")
	}
	p.MatchInfo.SecondsElapsed = 1.2
	if state = r.step(p, 0, false); state.CarStates[0].BoostAmount != nil {
		t.Fatal("Duplicate score reward")
	}
	p.Players[0].LatestTouch.GameSeconds = 2
	p.MatchInfo.SecondsElapsed = 2
	r.step(p, 0, false)
	for _, sec := range []float32{2.1, 2.2, 2.3, 2.4} {
		p.MatchInfo.SecondsElapsed = sec
		r.step(p, 0, false)
	}
	p.Players[1].LatestTouch.GameSeconds = 2.5
	p.MatchInfo.SecondsElapsed = 2.5
	state = r.step(p, 0, false)
	if state.CarStates[0].BoostAmount == nil || state.CarStates[0].BoostAmount.Val != 55 {
		t.Fatal("Pass not rewarded on reception")
	}
	s.Equipped = []string{"demos"}
	r = newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 20), 0, false)
	p = runtimePacket(.1, 0)
	p.Players[0].ScoreInfo.Demolitions = 1
	state = r.step(p, 0, false)
	if state.CarStates[0].BoostAmount.Val != 100 {
		t.Fatal("Demo reward missing")
	}
}
func TestAuxiliaryTransferAcknowledgement(t *testing.T) {
	s := newStorySave()
	s.Modules = map[string]int{"reserve": 2}
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.Extra = 50
	p := runtimePacket(.1, 20)
	state := r.step(p, 0, false)
	if state.CarStates[0].BoostAmount.Val != 70 || r.Extra != 50 {
		t.Fatal("Transfer must await acknowledgement")
	}
	r.step(runtimePacket(.2, 70), 0, false)
	if r.Extra != 0 {
		t.Fatal("Reserve not debited on acknowledgement")
	}
}
func TestFlowActivationAndRestore(t *testing.T) {
	s := newStorySave()
	storyTestWins(&s, 5)
	s.FlowMode = "overdrive"
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.activeTime = 26
	r.Charge = 100
	p := runtimePacket(.1, 20)
	p.Players[0].LastInput = &flat.ControllerStateT{UseItem: true}
	state := r.step(p, 0, false)
	if r.Uses != 1 || r.Charge != 0 || state.CarStates[0].BoostAmount.Val != 100 {
		t.Fatal("Flow not activated")
	}
	r.activeTime = 30
	p = runtimePacket(.2, 100)
	state = r.step(p, 0, false)
	if state.CarStates[0].BoostAmount.Val != 20 || r.OverdriveUntil != 0 {
		t.Fatal("Stored fuel not restored")
	}
}
