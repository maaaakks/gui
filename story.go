package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	rlbot "github.com/RLBot/go-interface"
	"github.com/RLBot/go-interface/flat"
)

// A new campaign inspired by RLBotGUI v4's Story Mode. See STORY_MODE.md
// for provenance, current behavior and prototype limitations.
//
//go:embed story-campaign.json
var storyCampaignJSON []byte

type StoryConditions struct {
	Win               *bool `json:"win,omitempty"`
	ScoreDifference   int   `json:"scoreDifference,omitempty"`
	SelfDemoCount     *int  `json:"selfDemoCount,omitempty"`
	DemoAchievedCount int   `json:"demoAchievedCount,omitempty"`
	GoalsScored       int   `json:"goalsScored,omitempty"`
}
type StoryChallenge struct {
	OpponentNames    []string        `json:"opponentNames"`
	BestOf           int             `json:"bestOf"`
	Brief            string          `json:"brief"`
	RankOffset       int             `json:"rankOffset"`
	EnemyRank        int             `json:"enemyRank"`
	OpponentBoostCap float32         `json:"opponentBoostCap"`
	ID               string          `json:"id"`
	HumanTeamSize    int             `json:"humanTeamSize"`
	OpponentBots     []string        `json:"opponentBots"`
	OriginalBots     []string        `json:"originalBots"`
	Map              string          `json:"map"`
	Display          string          `json:"display"`
	Optional         bool            `json:"optional"`
	DisabledBoost    bool            `json:"disabledBoost"`
	MaxScore         string          `json:"max_score"`
	Limitations      []string        `json:"limitations"`
	Conditions       StoryConditions `json:"completionConditions"`
	Adaptation       string          `json:"adaptation"`
}
type StoryCity struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Prereqs     []string         `json:"prereqs"`
	Challenges  []StoryChallenge `json:"challenges"`
}
type StoryAttempt struct {
	SeriesBestOf  int    `json:"seriesBestOf,omitempty"`
	SeriesWins    int    `json:"seriesWins,omitempty"`
	SeriesLosses  int    `json:"seriesLosses,omitempty"`
	KilobyteSkill *int   `json:"kilobyteSkill,omitempty"`
	ChallengeID   string `json:"challengeId"`
	Won           bool   `json:"won"`
	Completed     bool   `json:"completed"`
	Blue          int    `json:"blue"`
	Orange        int    `json:"orange"`
	At            string `json:"at"`
}
type StorySave struct {
	Vanilla        bool                   `json:"vanilla"`
	Lineup         []string               `json:"lineup"`
	Series         map[string]StorySeries `json:"series"`
	Modules        map[string]int         `json:"modules"`
	Equipped       []string               `json:"equipped"`
	StarterClaimed bool                   `json:"starterClaimed"`
	FlowMode       string                 `json:"flowMode"`
	FlowLevels     map[string]int         `json:"flowLevels"`
	Format         string                 `json:"format"`
	Ranks          map[string]int         `json:"ranks"`
	ProfileSet     bool                   `json:"profileSet"`
	ThreeGoalLead  *bool                  `json:"threeGoalLead,omitempty"`
	KilobyteSkill  *int                   `json:"kilobyteSkill,omitempty"`
	Version        int                    `json:"version"`
	Club           string                 `json:"club"`
	Credits        int                    `json:"credits"`
	Recruits       []string               `json:"recruits"`
	Upgrades       []string               `json:"upgrades"`
	Completed      map[string]bool        `json:"completed"`
	Attempts       []StoryAttempt         `json:"attempts"`
}
type StoryStatus struct {
	FlowCharge     float32 `json:"flowCharge"`
	FlowUses       int     `json:"flowUses"`
	ExtraBoost     float32 `json:"extraBoost"`
	EquipmentEvent string  `json:"equipmentEvent"`
	Active         bool    `json:"active"`
	ChallengeID    string  `json:"challengeId"`
	Message        string  `json:"message"`
	Blue           int     `json:"blue"`
	Orange         int     `json:"orange"`
}
type StoryView struct {
	Roster        []StoryRecruit `json:"roster"`
	Modules       []StoryModule  `json:"modules"`
	Slots         int            `json:"slots"`
	Wins          int            `json:"wins"`
	MaxTier       int            `json:"maxTier"`
	Cities        []StoryCity    `json:"cities"`
	Save          StorySave      `json:"save"`
	Unlocked      []string       `json:"unlocked"`
	Status        StoryStatus    `json:"status"`
	AvailableBots []string       `json:"availableBots"`
}

func storyCities() []StoryCity {
	var cities []StoryCity
	if err := json.Unmarshal(storyCampaignJSON, &cities); err != nil {
		panic(err)
	}
	return cities
}
func newStorySave() StorySave {
	save := StorySave{Version: 2, Club: "Your club", Recruits: []string{"rookie"}, Upgrades: []string{}, Completed: map[string]bool{}, Attempts: []StoryAttempt{}}
	normalizeStorySave(&save)
	return save
}
func (a *App) storyPath() string {
	if a.storySavePath != "" {
		return a.storySavePath
	}
	return filepath.Join(a.GetDefaultPath("story"), "career-v1.json")
}
func (a *App) loadStory() (StorySave, error) {
	data, err := os.ReadFile(a.storyPath())
	if errors.Is(err, os.ErrNotExist) {
		return newStorySave(), nil
	}
	if err != nil {
		return StorySave{}, err
	}
	var save StorySave
	if err = json.Unmarshal(data, &save); err != nil {
		return save, fmt.Errorf("Cannot read career save; preserved at %s: %w", a.storyPath(), err)
	}
	if (save.Version != 1 && save.Version != 2) || save.Completed == nil || save.Credits < 0 {
		return save, errors.New("Unsupported or invalid career save; file has been preserved")
	}
	normalizeStorySave(&save)
	return save, nil
}
func (a *App) writeStory(save StorySave) error {
	path := a.storyPath()
	if old, e := os.ReadFile(path); e == nil {
		var previous struct {
			Version int `json:"version"`
		}
		if json.Unmarshal(old, &previous) == nil && previous.Version == 1 && save.Version == 2 {
			backup := path + ".before-season-v2.json"
			if f, e := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); e == nil {
				_, _ = f.Write(old)
				_ = f.Close()
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(save, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "career-*.tmp")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func unlockedStoryCities(save StorySave, cities []StoryCity) []string {
	finished := map[string]bool{}
	for _, city := range cities {
		done := true
		for _, c := range city.Challenges {
			if !c.Optional && !save.Completed[c.ID] {
				done = false
			}
		}
		finished[city.ID] = done
	}
	unlocked := []string{}
	for _, city := range cities {
		open := true
		for _, p := range city.Prereqs {
			if !finished[p] {
				open = false
			}
		}
		if open {
			unlocked = append(unlocked, city.ID)
		}
	}
	return unlocked
}
func (a *App) GetStory() (StoryView, error) {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	save, err := a.loadStory()
	if err != nil {
		return StoryView{}, err
	}
	cities := storyCitiesForSave(save)
	return StoryView{Roster: storyRecruitmentRoster(), Modules: storyModules, Slots: storySlots(save), Wins: storyWins(save), MaxTier: storyMaxTier(save), Cities: cities, Save: save, Unlocked: unlockedStoryCities(save, cities), Status: a.storyStatus, AvailableBots: []string{}}, nil
}
func (a *App) RenameStoryClub(name string) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 1 || len([]rune(name)) > 40 {
		return errors.New("Club name must contain 1–40 characters")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	save.Club = name
	return a.writeStory(save)
}

var storyUpgradeCosts = map[string]int{"boost-33": 2, "boost-100": 4, "boost-regen": 6}

func (a *App) PurchaseStory(id string, recruit bool) error {
	if recruit {
		return a.RecruitStory(id, []string{a.GetDefaultPath("bots")})
	}
	return errors.New("The old boost upgrades have been retired. Use the workshop equipment instead.")
}

// Existing saves without this field start at Gold I. Zero remains Auto.
func storyThreeGoalLead(save StorySave) bool {
	return save.ThreeGoalLead == nil || *save.ThreeGoalLead
}
func (a *App) SetStoryThreeGoalLead(enabled bool) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish or stop your current challenge first")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	save.ThreeGoalLead = &enabled
	return a.writeStory(save)
}
func storyMatchMutators(c StoryChallenge, threeGoalLead bool) flat.MutatorSettingsT {
	mutators := flat.MutatorSettingsT{}
	// A first-to-three ceiling would incorrectly stop a 3–2 match.
	if !threeGoalLead && c.MaxScore == "3 Goals" {
		mutators.MaxScore = flat.MaxScoreMutatorThreeGoals
	}
	if c.DisabledBoost {
		mutators.BoostAmount = flat.BoostAmountMutatorNoBoost
	}
	return mutators
}
func storyEarlyFinish(threeGoalLead bool, blue, orange int) bool {
	return threeGoalLead && (blue-orange >= 3 || orange-blue >= 3)
}
func storySkill(save StorySave) int {
	if save.KilobyteSkill == nil {
		return 7
	}
	return *save.KilobyteSkill
}
func (a *App) SetStoryDifficulty(skill int) error {
	a.storyMu.Lock()
	defer a.storyMu.Unlock()
	if a.storyStatus.Active {
		return errors.New("Finish or stop your current challenge first")
	}
	if skill < 0 || skill > 22 {
		return errors.New("Difficulty must be Auto (0) or Bronze I through SSL (1–22)")
	}
	save, err := a.loadStory()
	if err != nil {
		return err
	}
	if storyHasOpenSeries(save) {
		return errors.New("Finish or restart your unfinished series before changing ranks")
	}
	save.KilobyteSkill = &skill
	return a.writeStory(save)
}
func storySkillLabel(skill int) string {
	if skill == 0 {
		return "Auto"
	}
	if skill == 22 {
		return "SSL"
	}
	if skill < 1 || skill > 22 {
		return "Unknown"
	}
	tiers := []string{"Bronze", "Silver", "Gold", "Platinum", "Diamond", "Champion", "Grand Champion"}
	return tiers[(skill-1)/3] + " " + []string{"I", "II", "III"}[(skill-1)%3]
}
func storyKilobytePlayer(bots []BotInfo, skill int) (PlayerJs, error) {
	if skill < 0 || skill > 22 {
		return PlayerJs{}, errors.New("Invalid Kilobyte difficulty")
	}
	for _, bot := range bots {
		if bot.Config.Settings.AgentId != "SubparNova/KilobyteReduced" {
			continue
		}
		settings := &bot.Config.Settings
		// Override only this launch's copied configuration, never the installed TOML.
		for _, command := range []string{settings.RunCommand, settings.RunCommandLinux} {
			if strings.Contains(command, "--skill") {
				return PlayerJs{}, errors.New("Kilobyte launch command already contains --skill; use the unmodified official botpack configuration")
			}
		}
		suffix := fmt.Sprintf(" --skill=%d --no-ui", skill)
		if settings.RunCommand != "" {
			settings.RunCommand += suffix
		}
		if settings.RunCommandLinux != "" {
			settings.RunCommandLinux += suffix
		}
		settings.Name = "Kilobyte[" + storySkillLabel(skill) + "]"
		bot.Icon = ""
		data, err := json.Marshal(bot)
		return PlayerJs{Sort: "rlbot", Player: data}, err
	}
	return PlayerJs{}, errors.New("KilobyteReduced is missing. Install the official v5 botpack on the Play page first")
}
func storyPlayer(id string, bots []BotInfo) (PlayerJs, error) {
	var value any
	sort := "psyonix"
	switch id {
	case "human":
		sort = "human"
		value = HumanInfo{}
	case "rookie":
		value = PsyonixBotInfo{Name: "Rookie", Skill: 1}
	case "pro":
		value = PsyonixBotInfo{Name: "Pro", Skill: 2}
	case "allstar":
		value = PsyonixBotInfo{Name: "All-Star", Skill: 3}
	default:
		for _, b := range bots {
			if b.Config.Settings.Name == id {
				b.Icon = ""
				value = b
				sort = "rlbot"
				break
			}
		}
		if value == nil {
			return PlayerJs{}, fmt.Errorf("Missing v5 bot: %s. Install/update the official RLBot botpack in Play first", id)
		}
	}
	data, err := json.Marshal(value)
	return PlayerJs{Sort: sort, Player: data}, err
}
func findStoryChallenge(id string) (StoryCity, StoryChallenge, error) {
	for _, city := range storyCities() {
		for _, c := range city.Challenges {
			if c.ID == id {
				return city, c, nil
			}
		}
	}
	return StoryCity{}, StoryChallenge{}, errors.New("Unknown challenge")
}

// The campaign owns match settings; freeplay settings and enabled training scripts
// must not leak into progression matches.
func (a *App) StartStory(id string, teammates []string, paths []string, launcher string, launcherArg string) (err error) {
	a.storyMu.Lock()
	if a.storyStatus.Active {
		a.storyMu.Unlock()
		return errors.New("A challenge is already running")
	}
	save, err := a.loadStory()
	if err != nil {
		a.storyMu.Unlock()
		return err
	}
	city, c, err := storyChapterMatch(save, id)
	if err != nil {
		a.storyMu.Unlock()
		return err
	}
	if !slices.Contains(unlockedStoryCities(save, storyCities()), city.ID) {
		a.storyMu.Unlock()
		return errors.New("District is locked")
	}
	if len(teammates) != c.HumanTeamSize-1 {
		a.storyMu.Unlock()
		return errors.New("Select the required number of teammates")
	}
	if e := storyValidateLineup(save, teammates); e != nil {
		a.storyMu.Unlock()
		return e
	}
	if !slices.Contains([]string{"steam", "epic", "legendary", "heroic", "custom", "nolaunch"}, launcher) {
		a.storyMu.Unlock()
		return errors.New("Select your Rocket League launcher on the Play page first")
	}
	a.storyStatus = StoryStatus{Active: true, ChallengeID: id, Message: "Preparing challenge…"}
	a.storyMu.Unlock()
	defer func() {
		if err != nil {
			a.storyMu.Lock()
			a.storyStatus.Active = false
			a.storyStatus.Message = err.Error()
			a.storyMu.Unlock()
		}
	}()
	bots := a.GetBots(paths)
	blue := []PlayerJs{}
	orange := []PlayerJs{}
	for _, name := range append([]string{"human"}, teammates...) {
		var p PlayerJs
		var e error
		if name == "human" {
			p, e = storyPlayer(name, bots)
		} else {
			p, e = storyRecruitPlayer(name, bots, storyReferenceRank(save, c.HumanTeamSize))
		}
		if e != nil {
			return e
		}
		blue = append(blue, p)
	}
	for i, name := range c.OpponentBots {
		var p PlayerJs
		var e error
		if slices.Contains(c.Limitations, "half-field") {
			p, e = storyPlayer(name, bots)
		} else {
			p, e = storyKilobytePlayer(bots, c.EnemyRank)
		}
		if e != nil {
			return e
		}
		if i < len(c.OpponentNames) {
			p, e = storyNamedPlayer(p, c.OpponentNames[i])
			if e != nil {
				return e
			}
		}
		orange = append(orange, p)
	}
	mode := "Soccar"
	if slices.Contains(c.Limitations, "half-field") {
		mode = "Heatseeker"
	}
	mutators := storyMatchMutators(c, storyThreeGoalLead(save))
	if save.Vanilla {
		mutators = flat.MutatorSettingsT{}
	}
	result := a.StartMatch(StartMatchOptions{Map: storyMapPackages[c.Map], GameMode: mode, BluePlayers: blue, OrangePlayers: orange, MutatorSettings: mutators, Launcher: launcher, LauncherArg: launcherArg, ExtraOptions: ExtraOptions{EnableRendering: flat.DebugRenderingOnByDefault, PerformanceMonitor: flat.PerformanceMonitorNeverShow, EnableStateSetting: !save.Vanilla, AutoStartAgents: true, WaitForAgents: true, ExistingMatchBehavior: byte(flat.ExistingMatchBehaviorRestart)}})
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
	go a.observeStory(conn, c, save, c.EnemyRank, storyThreeGoalLead(save))
	return nil
}
func storyScores(p *flat.GamePacketT) (blue, orange int) {
	for _, t := range p.Teams {
		if t.TeamIndex == 0 {
			blue = int(t.Score)
		} else if t.TeamIndex == 1 {
			orange = int(t.Score)
		}
	}
	return
}
func storyCompleted(c StoryChallenge, blue, orange, received, demos, goals int) bool {
	r := c.Conditions
	if (r.Win == nil || *r.Win) && blue <= orange {
		return false
	}
	if blue-orange < r.ScoreDifference {
		return false
	}
	if r.SelfDemoCount != nil && received > *r.SelfDemoCount {
		return false
	}
	return demos >= r.DemoAchievedCount && goals >= r.GoalsScored
}
func storyObjective(c StoryChallenge) string {
	text := "Win the match"
	if c.Conditions.ScoreDifference > 0 {
		text += fmt.Sprintf(" by %d+ goals", c.Conditions.ScoreDifference)
	}
	if c.Conditions.SelfDemoCount != nil {
		text += " | Avoid being demolished"
	}
	if c.Conditions.DemoAchievedCount > 0 {
		text += fmt.Sprintf(" | Opponent demos: %d+", c.Conditions.DemoAchievedCount)
	}
	if c.Conditions.GoalsScored > 0 {
		text += fmt.Sprintf(" | Score %d+ yourself", c.Conditions.GoalsScored)
	}
	return text
}

// v4 used Python map aliases; v5 expects Unreal package names.
var storyMapPackages = map[string]string{
	"Farmstead_Night": "Farm_Night_P", "BeckwithPark_Midnight": "Park_Night_P",
	"UrbanCentral_Night": "TrainStation_Night_P", "Mannfield": "EuroStadium_P",
	"DFHStadium": "Stadium_P", "DFHStadium_Day": "stadium_day_p",
	"BeckwithPark": "Park_P", "Farmstead": "farm_p", "UrbanCentral": "TrainStation_P", "UrbanCentral_Dawn": "TrainStation_Dawn_P",
	"UtopiaColiseum_Snowy": "UtopiaStadium_Snow_P", "Mannfield_Stormy": "EuroStadium_Rainy_P", "DFHStadium_Stormy": "Stadium_Foggy_P",
	"Wasteland": "wasteland_s_p", "Wasteland_Night": "Wasteland_Night_S_P", "AquaDome": "Underwater_P", "NeoTokyo": "NeoTokyo_Standard_P",
	"ChampionsField": "cs_p", "ChampionsField_Day": "cs_day_p", "Mannfield_Night": "EuroStadium_Night_P",
}

func (a *App) observeStory(conn rlbot.RLBotConnection, c StoryChallenge, career StorySave, skill int, threeGoalLead bool) {
	runtime := newStoryRuntime(career, c)
	lastHuman, count := -1, 0

	finished := false
	defer func() {
		runtime.restore(conn, lastHuman, count)
		_ = conn.SendPacket(&flat.DisconnectSignalT{})
		a.storyMu.Lock()
		defer a.storyMu.Unlock()
		a.storyStatus.Active = false
		if !finished {
			a.storyStatus.Message = "Challenge interrupted; no result or credits recorded."
		}
	}()
	received, demos, goals := 0, 0, 0
	wasDemo := map[int32]bool{}
	lastHUD := time.Time{}
	seenActive := false
	botsMuted := false
	previousFlowButton := true
	for {
		msg, err := conn.RecvPacket()
		if err != nil {
			return
		}
		if _, ok := msg.Value.(*flat.DisconnectSignalT); ok {
			return
		}
		if field, ok := msg.Value.(*flat.FieldInfoT); ok {
			runtime.field = field
		}
		p, ok := msg.Value.(*flat.GamePacketT)
		if !ok || p.MatchInfo == nil {
			continue
		}
		if len(p.Players) != c.HumanTeamSize+len(c.OpponentBots) {
			continue
		}
		human := -1
		for i, car := range p.Players {
			if !car.IsBot && car.Team == 0 {
				human = i
				break
			}
		}
		if human < 0 {
			continue
		}
		if !botsMuted {
			for i, car := range p.Players {
				if car.IsBot {
					_ = conn.SendPacket(&flat.RenderingStatusT{Index: uint32(i), IsBot: true, Status: false})
				}
			}
			botsMuted = true
		}
		lastHuman = human
		count = len(p.Players)
		phase := p.MatchInfo.MatchPhase
		if phase != flat.MatchPhaseEnded && phase != flat.MatchPhaseInactive {
			seenActive = true
		}
		for i, car := range p.Players {
			demo := car.DemolishedTimeout > 0
			if demo && !wasDemo[car.PlayerId] {
				if i == human {
					received++
				} else if car.Team == 1 {
					demos++
				}
			}
			wasDemo[car.PlayerId] = demo
		}
		// Match the v4 definition: opponent demolitions, whoever caused them.
		// Goal scoring uses the player's own scoreboard counter in v5.
		if p.Players[human].ScoreInfo != nil {
			goals = int(p.Players[human].ScoreInfo.Goals)
		}
		blue, orange := storyScores(p)
		earlyFinish := seenActive && storyEarlyFinish(threeGoalLead, blue, orange)
		if seenActive && (phase == flat.MatchPhaseEnded || earlyFinish) {
			a.storyMu.Lock()
			save, saveErr := a.loadStory()
			message := "Result could not be recorded."
			if saveErr == nil {
				message = storyRecordGame(&save, c, blue, orange, received, demos, goals, skill)
				saveErr = a.writeStory(save)
			}
			if saveErr != nil {
				message = "Result could not be saved: " + saveErr.Error()
			}
			a.storyStatus = StoryStatus{ChallengeID: c.ID, Blue: blue, Orange: orange, Message: message}
			a.storyMu.Unlock()
			if earlyFinish && phase != flat.MatchPhaseEnded {
				if stopErr := conn.SendPacket(&flat.StopCommandT{ShutdownServer: false}); stopErr != nil {
					a.storyMu.Lock()
					a.storyStatus.Message += " · Unable to stop the match: " + stopErr.Error()
					a.storyMu.Unlock()
				}
			}
			finished = true
			return
		}
		if phase == flat.MatchPhaseEnded || phase == flat.MatchPhaseInactive {
			continue
		}
		a.storyMu.Lock()
		trigger := a.storyFlowRequested
		a.storyFlowRequested = false
		a.storyMu.Unlock()
		button := storyFlowButtonDown()
		trigger = trigger || (button && !previousFlowButton)
		previousFlowButton = button
		if state := runtime.step(p, human, trigger); state != nil {
			_ = conn.SendPacket(state)
		}

		if time.Since(lastHUD) > 500*time.Millisecond {
			runtime.render(conn, p.Players[human].Boost)
			message := fmt.Sprintf("%d – %d", blue, orange)
			a.storyMu.Lock()
			a.storyStatus = StoryStatus{Active: true, ChallengeID: c.ID, Blue: blue, Orange: orange, Message: message, FlowCharge: runtime.Charge, FlowUses: runtime.Uses, ExtraBoost: runtime.Extra, EquipmentEvent: runtime.LastEvent}
			a.storyMu.Unlock()
			lastHUD = time.Now()
		}
	}
}
