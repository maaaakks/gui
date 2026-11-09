package main

import (
	"errors"
	"fmt"
	"slices"

	rlbot "github.com/RLBot/go-interface"
	"github.com/RLBot/go-interface/flat"
)

// Ranks describe each slot, independently; the human is always blue slot zero.
type StoryFreeMatchOptions struct {
	Size        int    `json:"size"`
	Arena       string `json:"arena"`
	BlueRanks   []int  `json:"blueRanks"`
	OrangeRanks []int  `json:"orangeRanks"`
}

var storyFreeMatchArenas = []string{"BeckwithPark", "Farmstead", "UrbanCentral", "Mannfield", "DFHStadium", "ChampionsField"}

func storyFreeMatchConfig(o StoryFreeMatchOptions, bots []BotInfo, launcher, launcherArg string) (StartMatchOptions, error) {
	if o.Size < 1 || o.Size > 3 || len(o.BlueRanks) != o.Size-1 || len(o.OrangeRanks) != o.Size {
		return StartMatchOptions{}, errors.New("Choose 1v1, 2v2 or 3v3 and a rank for every bot")
	}
	if !slices.Contains(storyFreeMatchArenas, o.Arena) {
		return StartMatchOptions{}, errors.New("Choose an available free-match arena")
	}
	if !slices.Contains([]string{"steam", "epic", "legendary", "heroic", "custom", "nolaunch"}, launcher) {
		return StartMatchOptions{}, errors.New("Select your Rocket League launcher on the Play page first")
	}
	for _, ranks := range [][]int{o.BlueRanks, o.OrangeRanks} {
		for _, rank := range ranks {
			if rank < 1 || rank > 22 {
				return StartMatchOptions{}, errors.New("Choose a bot rank between Bronze I and SSL")
			}
		}
	}
	human, err := storyPlayer("human", nil)
	if err != nil {
		return StartMatchOptions{}, err
	}
	config := StartMatchOptions{
		Map: storyMapPackages[o.Arena], GameMode: "Soccar", Launcher: launcher, LauncherArg: launcherArg,
		BluePlayers: []PlayerJs{human}, OrangePlayers: []PlayerJs{},
		ExtraOptions: ExtraOptions{
			EnableRendering:    flat.DebugRenderingAlwaysOff,
			PerformanceMonitor: flat.PerformanceMonitorNeverShow,
			AutoStartAgents:    true, WaitForAgents: true,
			ExistingMatchBehavior: byte(flat.ExistingMatchBehaviorRestart),
		},
	}
	for team, ranks := range [][]int{o.BlueRanks, o.OrangeRanks} {
		for i, rank := range ranks {
			player, err := storyKilobytePlayer(bots, rank)
			if err != nil {
				return StartMatchOptions{}, err
			}
			name := []string{"Morgan Vale", "Casey Reed"}[min(i, 1)]
			if team == 1 {
				name = []string{"Rowan Hayes", "Lena Cross", "Viktor Rook"}[i]
			}
			player, err = storyNamedPlayer(player, name)
			if err != nil {
				return StartMatchOptions{}, err
			}
			if team == 0 {
				config.BluePlayers = append(config.BluePlayers, player)
			} else {
				config.OrangePlayers = append(config.OrangePlayers, player)
			}
		}
	}
	return config, nil
}

func (a *App) StartStoryFreeMatch(o StoryFreeMatchOptions, paths []string, launcher, launcherArg string) (err error) {
	a.storyMu.Lock()
	if a.storyStatus.Active {
		a.storyMu.Unlock()
		return errors.New("Finish or stop the current match first")
	}
	a.storyStatus = StoryStatus{Active: true, FreeMatch: true, Message: "Preparing free match…"}
	a.storyFlowRequested = false
	a.storyMu.Unlock()
	defer func() {
		if err != nil {
			a.storyMu.Lock()
			a.storyStatus = StoryStatus{FreeMatch: true, Message: err.Error()}
			a.storyMu.Unlock()
		}
	}()
	config, err := storyFreeMatchConfig(o, a.GetBots(paths), launcher, launcherArg)
	if err != nil {
		return err
	}
	result := a.StartMatch(config)
	if !result.Success {
		return errors.New(result.Message)
	}
	conn, err := rlbot.Connect(a.rlbotAddress)
	if err != nil {
		return err
	}
	if err = conn.SendPacket(&flat.ConnectionSettingsT{CloseBetweenMatches: true}); err != nil {
		return err
	}
	if err = conn.SendPacket(&flat.InitCompleteT{}); err != nil {
		return err
	}
	go a.observeStoryFreeMatch(conn, o.Size*2)
	return nil
}

// Observe only lifecycle and score. There is intentionally no StoryRuntime,
// equipment state setting, career load/write, reward or campaign result here.
func (a *App) observeStoryFreeMatch(conn rlbot.RLBotConnection, players int) {
	finished := false
	defer conn.SendPacket(&flat.DisconnectSignalT{})
	defer func() {
		a.storyMu.Lock()
		defer a.storyMu.Unlock()
		if a.storyStatus.FreeMatch {
			a.storyStatus.Active = false
			if !finished {
				a.storyStatus.Message = "Free match stopped."
			}
		}
	}()
	for {
		msg, err := conn.RecvPacket()
		if err != nil {
			return
		}
		if _, ok := msg.Value.(*flat.DisconnectSignalT); ok {
			return
		}
		p, ok := msg.Value.(*flat.GamePacketT)
		if !ok || p.MatchInfo == nil || len(p.Players) != players {
			continue
		}
		blue, orange := storyScores(p)
		ended := p.MatchInfo.MatchPhase == flat.MatchPhaseEnded
		message := fmt.Sprintf("Free match · %d – %d", blue, orange)
		if ended {
			message = fmt.Sprintf("Free match finished · %d – %d", blue, orange)
		}
		a.storyMu.Lock()
		a.storyStatus = StoryStatus{Active: true, FreeMatch: true, Blue: blue, Orange: orange, Message: message}
		a.storyMu.Unlock()
		if ended {
			finished = true
			return
		}
	}
}
