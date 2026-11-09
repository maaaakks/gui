package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

type StoryRecruit struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Portrait string `json:"portrait"`
	Bio      string `json:"bio"`
	BotName  string `json:"botName"`
	Adaptive bool   `json:"adaptive"`
	Cost     int    `json:"cost"`
	Unlock   int    `json:"unlock"`
}

var storyRoster = []StoryRecruit{
	{ID: "rookie", Name: "Morgan Vale", Portrait: "morgan", Bio: "Usually laughing, occasionally covered in oil. Will celebrate your scrappiest goal like a tournament winner.", BotName: "KilobyteReduced", Adaptive: true},
	{ID: "pro", Name: "Casey Reed", Portrait: "casey", Bio: "Competitive, impatient, and already asking for a rematch. A raised eyebrow says more than a speech.", BotName: "KilobyteReduced", Adaptive: true, Cost: 150, Unlock: 4},
	{ID: "allstar", Name: "Jamie Cole", Portrait: "jamie", Bio: "The loudest laugh at the club. A bad bounce is usually the start of another joke.", BotName: "KilobyteReduced", Adaptive: true, Cost: 75, Unlock: 4},
	{ID: "Beast from the East", Name: "Rowan Hayes", Portrait: "rowan", Bio: "A crooked grin and one comment too many. Loves a close game almost as much as the last word.", BotName: "KilobyteReduced", Adaptive: true, Cost: 125, Unlock: 4},
	{ID: "Phoenix CS", Name: "Lena Cross", Portrait: "lena", Bio: "Quiet on the drive over. Says little during a match, but knows when the team needs a steady word.", BotName: "KilobyteReduced", Adaptive: true, Cost: 225, Unlock: 8},
	{ID: "Necto", Name: "Avery Stone", Portrait: "avery", Bio: "Lives for the big occasion. The first fist in the air and the last person ready to leave.", BotName: "KilobyteReduced", Adaptive: true, Cost: 350, Unlock: 12},
	{ID: "Willo 24k", Name: "Silas Ward", Portrait: "silas", Bio: "Doesn't celebrate until the scoreboard stops. You get a nod if the job was done properly.", BotName: "KilobyteReduced", Adaptive: true, Cost: 300, Unlock: 12},
	{ID: "Kira", Name: "Mara Voss", Portrait: "mara", Bio: "No small talk before kickoff. A long look is usually the whole conversation.", BotName: "KilobyteReduced", Adaptive: true, Cost: 400, Unlock: 16},
	{ID: "Fury", Name: "Viktor Rook", Portrait: "viktor", Bio: "Takes the match personally. The rematch request arrives before the replay ends.", BotName: "KilobyteReduced", Adaptive: true, Cost: 500, Unlock: 20},
}

func storyRecruit(id string) (StoryRecruit, bool) {
	for _, r := range storyRoster {
		if r.ID == id {
			return r, true
		}
	}
	return StoryRecruit{}, false
}

func storyNamedPlayer(p PlayerJs, name string) (PlayerJs, error) {
	if p.Sort == "rlbot" {
		var bot BotInfo
		if err := json.Unmarshal(p.Player, &bot); err != nil {
			return PlayerJs{}, err
		}
		bot.Config.Settings.Name = name
		data, err := json.Marshal(bot)
		p.Player = data
		return p, err
	}
	if p.Sort == "psyonix" {
		var bot PsyonixBotInfo
		if err := json.Unmarshal(p.Player, &bot); err != nil {
			return PlayerJs{}, err
		}
		bot.Name = name
		data, err := json.Marshal(bot)
		p.Player = data
		return p, err
	}
	return p, nil
}

func storyRecruitPlayer(id string, bots []BotInfo, rank int) (PlayerJs, error) {
	recruit, known := storyRecruit(id)
	if !known {
		return storyPlayer(id, bots)
	} // Preserve legacy recruits.
	p, err := storyKilobytePlayer(bots, rank)
	if err != nil {
		return PlayerJs{}, err
	}
	return storyNamedPlayer(p, recruit.Name)
}

func (a *App) RecruitStory(id string, paths []string) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish or stop the current match first")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	recruit, ok := storyRecruit(id)
	if !ok {
		return errors.New("Unknown recruit")
	}
	if slices.Contains(save.Recruits, id) {
		return errors.New("Already in your roster")
	}
	if storyWins(save) < recruit.Unlock {
		return fmt.Errorf("Available after %d fixtures won", recruit.Unlock)
	}
	if save.Credits < recruit.Cost {
		return errors.New("Not enough credits")
	}
	if _, err = storyRecruitPlayer(id, a.GetBots(paths), storyReferenceRank(save, 2)); err != nil {
		return err
	}
	save.Credits -= recruit.Cost
	save.Recruits = append(save.Recruits, id)
	return a.writeStory(save)
}

func storyValidateLineup(save StorySave, ids []string) error {
	if len(ids) > 2 {
		return errors.New("Choose at most two teammates")
	}
	used := map[string]bool{}
	for _, id := range ids {
		if !slices.Contains(save.Recruits, id) {
			return errors.New("Teammate has not been recruited")
		}
		if used[id] {
			return errors.New("Choose a different driver for each teammate slot")
		}
		used[id] = true
	}
	return nil
}
func (a *App) SetStoryLineup(ids []string) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish or stop the current match first")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	if err = storyValidateLineup(save, ids); err != nil {
		return err
	}
	save.Lineup = ids
	return a.writeStory(save)
}

var storyRivals = []string{"Logan Price", "Jules Mercer", "Finn Archer", "Skye Bennett", "Blake Porter", "Maya Torres", "Oscar Lane", "Zoe Walker", "Noah Carter", "Tess Moreno", "Luca Hayes", "Aria Knox"}

func storyOpponentNames(c StoryChallenge) []string {
	seed := 0
	for _, b := range []byte(c.ID) {
		seed = seed*31 + int(b)
		seed %= len(storyRivals)
	}
	names := make([]string, len(c.OpponentBots))
	for i := range names {
		names[i] = storyRivals[(seed+i)%len(storyRivals)]
	}
	return names
}
