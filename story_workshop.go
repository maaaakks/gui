package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type StoryModule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Values      []float32 `json:"values"`
	Unlock      int       `json:"unlock"`
	Available   bool      `json:"available"`
	Note        string    `json:"note"`
}

var storyModules = []StoryModule{
	{"reserve", "Auxiliary tank", "A full extra fuel reserve at match start. Large pads refill it; small pads store up to 12 overflow boost, even at a full main tank. Also stores equipment rewards. Used first to keep the main tank near 100.", []float32{25, 50, 75}, 2, true, "Total fuel capacity is 125 / 150 / 175. A reserve gauge appears above the native boost meter."},
	{"saves", "Defensive recovery", "Recover boost after a save.", []float32{15, 20, 25}, 2, true, "8-second cooldown. Action rewards share a 60-boost rolling-minute cap."},
	{"shots", "Shooting rhythm", "Recover boost after a shot on target.", []float32{8, 10, 12}, 2, true, "6-second cooldown. Action rewards share a 60-boost rolling-minute cap."},
	{"passing", "Link-up play", "Recover boost when a teammate receives your pass, before any goal.", []float32{15, 20, 25}, 5, true, "8-second cooldown. Short accidental contacts do not count. Shares the 60-boost rolling-minute action cap."},
	{"demos", "Contact", "Recover a large amount of boost after an opponent demolition credited to you.", []float32{50, 75, 100}, 8, true, "20-second cooldown. Teammate demolitions never count."},
	{"respawn", "Back on track", "Shorter personal respawn and extra boost when you return.", []float32{25, 50, 75}, 8, false, "Prototype pending: personal respawn timing is not exposed by the current interface."},
	{"reset", "Aerial connection", "Recover boost after a verified flip reset.", []float32{10, 12, 15}, 16, false, "Prototype pending: reliable flip-reset detection must be verified in game."},
}

func storyModule(id string) (StoryModule, bool) {
	for _, m := range storyModules {
		if m.ID == id {
			return m, true
		}
	}
	return StoryModule{}, false
}
func storyWins(save StorySave) int {
	n := 0
	for _, city := range storyCities() {
		for _, c := range city.Challenges {
			if save.Completed[c.ID] {
				n++
			}
		}
	}
	return n
}
func storySlots(save StorySave) int {
	n := storyWins(save)
	if n >= 5 {
		return 2
	}
	if n >= 2 {
		return 1
	}
	return 0
}
func storyMaxTier(save StorySave) int {
	n := storyWins(save)
	if n >= 16 {
		return 3
	}
	if n >= 8 {
		return 2
	}
	return 1
}
func normalizeStorySave(save *StorySave) {
	if save.Series == nil {
		save.Series = map[string]StorySeries{}
	}
	if save.Version == 1 {
		refund := 0
		for _, id := range save.Upgrades {
			refund += storyUpgradeCosts[id]
		}
		save.Credits = (save.Credits + refund) * 50
		save.Upgrades = []string{}
		save.Version = 2
	}
	if save.Modules == nil {
		save.Modules = map[string]int{}
	}
	if save.FlowLevels == nil {
		save.FlowLevels = map[string]int{}
	}
	if save.Ranks == nil {
		save.Ranks = map[string]int{"1v1": -1, "2v2": -1, "3v3": -1}
	}
	if save.Format == "" {
		save.Format = "ascension"
	}
	if save.Equipped == nil {
		save.Equipped = []string{}
	}
	if storyWins(*save) >= 5 {
		if save.Modules["passing"] == 0 {
			save.Modules["passing"] = 1
		}
		if save.FlowLevels["surge"] == 0 {
			save.FlowLevels["surge"] = 1
		}
		if save.FlowLevels["overdrive"] == 0 {
			save.FlowLevels["overdrive"] = 1
		}
	}
	if (save.Format == "trio") && !slices.Contains(save.Recruits, "pro") {
		save.Recruits = append(save.Recruits, "pro")
	}
	if save.Lineup == nil {
		save.Lineup = []string{}
	}
	clean := []string{}
	for _, id := range save.Lineup {
		if slices.Contains(save.Recruits, id) && !slices.Contains(clean, id) && len(clean) < 2 {
			clean = append(clean, id)
		}
	}
	save.Lineup = clean
	if save.FlowMode == "" {
		save.FlowMode = "surge"
	}
}
func storyCitiesForSave(save StorySave) []StoryCity {
	cities := storyCities()
	for i := range cities {
		for j := range cities[i].Challenges {
			c := &cities[i].Challenges[j]
			switch save.Format {
			case "solo":
				c.HumanTeamSize = 1
			case "duo":
				c.HumanTeamSize = 2
			case "trio":
				c.HumanTeamSize = 3
			}
			c.OpponentBots = make([]string, c.HumanTeamSize)
			for k := range c.OpponentBots {
				c.OpponentBots[k] = "Kilobyte"
			}
			c.OpponentNames = storyOpponentNames(*c)
			c.EnemyRank = storyReferenceRank(save, c.HumanTeamSize)
			if c.EnemyRank != 0 {
				c.EnemyRank = max(1, min(22, c.EnemyRank+c.RankOffset))
			}
		}
	}
	return cities
}
func storyReferenceRank(save StorySave, size int) int {
	if rank, ok := save.Ranks[fmt.Sprintf("%dv%d", size, size)]; ok && rank >= 1 && rank <= 22 {
		return rank
	}
	return storySkill(save)
}
func (a *App) SetStoryProfile(format string, ranks map[string]int) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish the current match first")
	}
	if !slices.Contains([]string{"ascension", "solo", "duo", "trio"}, format) {
		return errors.New("Unknown career format")
	}
	for _, key := range []string{"1v1", "2v2", "3v3"} {
		r, ok := ranks[key]
		if !ok || (r != -1 && (r < 1 || r > 22)) {
			return errors.New("Choose a rank or Unknown for each playlist")
		}
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	if storyHasOpenSeries(save) {
		return errors.New("Finish or restart your unfinished series before changing the career format or ranks")
	}
	save.Format = format
	save.Ranks = ranks
	save.ProfileSet = true
	return a.writeStory(save)
}
func (a *App) BuyStoryModule(id string) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish the current match first")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	m, ok := storyModule(id)
	if !ok || !m.Available {
		return errors.New("This module is not available in the playable build")
	}
	if storyWins(save) < m.Unlock {
		return fmt.Errorf("Available after %d season wins", m.Unlock)
	}
	tier := save.Modules[id]
	if tier >= 3 || tier >= storyMaxTier(save) {
		return errors.New("Reach the next chapter to upgrade this module")
	}
	cost := []int{120, 150, 250}[tier]
	if tier == 0 && !save.StarterClaimed && slices.Contains([]string{"reserve", "saves", "shots"}, id) {
		cost = 0
		save.StarterClaimed = true
	}
	if save.Credits < cost {
		return errors.New("Not enough credits")
	}
	save.Credits -= cost
	save.Modules[id] = tier + 1
	if len(save.Equipped) < storySlots(save) && !slices.Contains(save.Equipped, id) {
		save.Equipped = append(save.Equipped, id)
	}
	return a.writeStory(save)
}
func (a *App) EquipStoryModule(id string) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish the current match first")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	if slices.Contains(save.Equipped, id) {
		save.Equipped = slices.DeleteFunc(save.Equipped, func(v string) bool { return v == id })
		return a.writeStory(save)
	}
	m, ok := storyModule(id)
	if !ok || !m.Available || save.Modules[id] < 1 {
		return errors.New("Buy this module first")
	}
	if len(save.Equipped) >= storySlots(save) {
		return errors.New("Unequip a module to free a slot")
	}
	save.Equipped = append(save.Equipped, id)
	return a.writeStory(save)
}
func (a *App) SetStoryFlow(mode string, upgrade bool) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish the current match first")
	}
	if !slices.Contains([]string{"surge", "overdrive"}, mode) {
		return errors.New("Focus is still in technical testing")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	if storyWins(save) < 5 {
		return errors.New("Flow State unlocks after five season wins")
	}
	tier := save.FlowLevels[mode]
	if upgrade {
		if tier >= 3 || tier >= storyMaxTier(save) {
			return errors.New("Reach the next chapter to upgrade Flow State")
		}
		cost := 150
		if tier == 2 {
			cost = 250
		}
		if save.Credits < cost {
			return errors.New("Not enough credits")
		}
		save.Credits -= cost
		save.FlowLevels[mode] = tier + 1
	}
	save.FlowMode = mode
	return a.writeStory(save)
}
func (a *App) TriggerStoryFlow() error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.FreeMatch {
		return errors.New("Flow State is disabled in free matches")
	}
	if !a.storyStatus.Active {
		return errors.New("Start a career match first")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	if save.Vanilla {
		return errors.New("Flow State is disabled in Vanilla mode")
	}
	if a.storyStatus.FlowCharge < 100 || a.storyStatus.FlowUses >= 2 {
		return errors.New("Flow State is not ready")
	}
	a.storyFlowRequested = true
	return nil
}
func storyResultReward(save *StorySave, c StoryChallenge, completed bool) int {
	reward := 25
	if completed && !save.Completed[c.ID] {
		reward = 100
	}
	save.Credits += reward
	if completed {
		save.Completed[c.ID] = true
	}
	normalizeStorySave(save)
	return reward
}
func storyChapterMatch(save StorySave, id string) (StoryCity, StoryChallenge, error) {
	for _, city := range storyCitiesForSave(save) {
		for _, c := range city.Challenges {
			if c.ID == id {
				return city, c, nil
			}
		}
	}
	return StoryCity{}, StoryChallenge{}, errors.New("Unknown season fixture")
}
func storyModuleValue(save StorySave, id string) float32 {
	if save.Vanilla {
		return 0
	}
	if !slices.Contains(save.Equipped, id) {
		return 0
	}
	m, ok := storyModule(id)
	if !ok || !m.Available {
		return 0
	}
	level := save.Modules[id]
	if level < 1 || level > 3 {
		return 0
	}
	return m.Values[level-1]
}
func storyWorkshopSummary(save StorySave) string {
	if save.Vanilla {
		return "Vanilla Rocket League · equipment and Flow State disabled"
	}
	names := []string{}
	for _, id := range save.Equipped {
		if m, ok := storyModule(id); ok {
			names = append(names, m.Name)
		}
	}
	if len(names) == 0 {
		return "Standard car · 100 boost capacity"
	}
	return strings.Join(names, " / ")
}
