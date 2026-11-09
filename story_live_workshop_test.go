//go:build storyprobe

package main

import (
	rlbot "github.com/RLBot/go-interface"
	"github.com/RLBot/go-interface/flat"
	"testing"
	"time"
)

func TestLiveWorkshopRoundtrip(t *testing.T) {
	a := NewApp()
	h, _ := storyPlayer("human", nil)
	b, _ := storyPlayer("rookie", nil)
	result := a.StartMatch(StartMatchOptions{Map: "Park_P", GameMode: "Soccar", BluePlayers: []PlayerJs{h}, OrangePlayers: []PlayerJs{b}, Launcher: "steam", ExtraOptions: ExtraOptions{EnableRendering: flat.DebugRenderingOnByDefault, PerformanceMonitor: flat.PerformanceMonitorNeverShow, EnableStateSetting: true, AutoStartAgents: true, WaitForAgents: true, ExistingMatchBehavior: byte(flat.ExistingMatchBehaviorRestart)}})
	if !result.Success {
		t.Fatal(result.Message)
	}
	defer a.StopMatch(false)
	conn, err := rlbot.Connect(a.rlbotAddress)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.SendPacket(&flat.DisconnectSignalT{})
	conn.SendPacket(&flat.ConnectionSettingsT{CloseBetweenMatches: true})
	conn.SendPacket(&flat.InitCompleteT{})
	packets := make(chan *flat.GamePacketT, 32)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			m, e := conn.RecvPacket()
			if e != nil {
				return
			}
			if p, ok := m.Value.(*flat.GamePacketT); ok {
				select {
				case packets <- p:
				case <-done:
					return
				}
			}
		}
	}()
	next := func() *flat.GamePacketT {
		select {
		case p := <-packets:
			return p
		case <-time.After(10 * time.Second):
			t.Fatal("No game packets")
			return nil
		}
	}
	p := next()
	for len(p.Players) != 2 || p.MatchInfo.MatchPhase != flat.MatchPhaseActive {
		p = next()
	}
	human := 0
	for i, c := range p.Players {
		if c.Team == 0 {
			human = i
		}
	}
	awaitBoost := func(want float32) *flat.GamePacketT {
		for i := 0; i < 240; i++ {
			p = next()
			if len(p.Players) > human && p.MatchInfo.MatchPhase == flat.MatchPhaseActive && p.Players[human].DemolishedTimeout <= 0 && p.Players[human].Boost >= want-1 && p.Players[human].Boost <= want+1 {
				return p
			}
		}
		t.Fatalf("Boost %.0f not observed; last %.1f", want, p.Players[human].Boost)
		return nil
	}
	setBoost := func(value float32) *flat.GamePacketT {
		for p.MatchInfo.MatchPhase != flat.MatchPhaseActive {
			p = next()
		}
		cars := []*flat.DesiredCarStateT{{}, {}}
		cars[human].BoostAmount = &flat.FloatT{Val: value}
		conn.SendPacket(&flat.DesiredGameStateT{CarStates: cars})
		return awaitBoost(value)
	}
	save := newStorySave()
	storyTestWins(&save, 5)
	save.Modules["reserve"] = 2
	save.Equipped = []string{"reserve"}
	runtime := newStoryRuntime(save, StoryChallenge{})
	p = setBoost(20)
	runtime.started = true
	runtime.initialFuelDone = true
	runtime.activeTime = 26
	runtime.lastSeconds = p.MatchInfo.SecondsElapsed
	runtime.Extra = 50
	state := runtime.step(p, human, false)
	if state == nil {
		t.Fatal("No auxiliary transfer")
	}
	conn.SendPacket(state)
	p = awaitBoost(70)
	runtime.step(p, human, false)
	if runtime.Extra != 0 {
		t.Fatal("Transfer not acknowledged")
	}
	t.Log("LIVE: auxiliary reserve transferred 50 into the native tank (20 -> 70)")
	p = setBoost(20)
	runtime.save.FlowMode = "surge"
	runtime.Charge = 100
	runtime.activeTime = 30
	runtime.lastSeconds = p.MatchInfo.SecondsElapsed
	state = runtime.step(p, human, true)
	if state == nil {
		t.Fatalf("No flow state generated in phase %v, uses=%d, charge=%v", p.MatchInfo.MatchPhase, runtime.Uses, runtime.Charge)
	}
	conn.SendPacket(state)
	awaitBoost(70)
	t.Log("LIVE: Surge I injected 50 boost (20 -> 70)")
	p = setBoost(20)
	runtime.save.FlowMode = "overdrive"
	runtime.Charge = 100
	runtime.activeTime = 35
	runtime.lastSeconds = p.MatchInfo.SecondsElapsed
	state = runtime.step(p, human, true)
	if state == nil {
		t.Fatalf("No flow state generated in phase %v, uses=%d, charge=%v", p.MatchInfo.MatchPhase, runtime.Uses, runtime.Charge)
	}
	conn.SendPacket(state)
	p = awaitBoost(100)
	runtime.activeTime = 40
	runtime.lastSeconds = p.MatchInfo.SecondsElapsed
	state = runtime.step(p, human, false)
	conn.SendPacket(state)
	awaitBoost(20)
	t.Log("LIVE: Overdrive fuel restored correctly (20 -> 100 temporary -> 20)")
	runtime.Charge = 100
	runtime.Uses = 0
	runtime.render(conn, 20)
	runtime.restore(conn, human, 2)
}
