//go:build storyprobe

package main

import (
	"encoding/json"
	rlbot "github.com/RLBot/go-interface"
	"github.com/RLBot/go-interface/flat"
	"os"
	"testing"
	"time"
)

// Explicit opt-in probe. Never runs in the normal test suite.
func TestLiveStoryCapabilities(t *testing.T) {
	a := NewApp()
	h, _ := storyPlayer("human", nil)
	b, _ := storyPlayer("rookie", nil)
	r := a.StartMatch(StartMatchOptions{Map: "Park_P", GameMode: "Soccar", BluePlayers: []PlayerJs{h}, OrangePlayers: []PlayerJs{b}, Launcher: "steam", ExtraOptions: ExtraOptions{EnableRendering: flat.DebugRenderingAlwaysOff, PerformanceMonitor: flat.PerformanceMonitorNeverShow, EnableStateSetting: true, AutoStartAgents: true, WaitForAgents: true, ExistingMatchBehavior: byte(flat.ExistingMatchBehaviorRestart)}})
	if !r.Success {
		t.Fatal(r.Message)
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
		case <-time.After(15 * time.Second):
			t.Fatal("No game packets")
			return nil
		}
	}
	p := next()
	for len(p.Players) != 2 || p.MatchInfo == nil || p.MatchInfo.MatchPhase != flat.MatchPhaseActive {
		p = next()
	}
	result := map[string]any{}
	for _, target := range []float32{100, 125, 150, 175} {
		for len(packets) > 0 {
			<-packets
		}
		cars := []*flat.DesiredCarStateT{{BoostAmount: &flat.FloatT{Val: target}}, {}}
		if err = conn.SendPacket(&flat.DesiredGameStateT{CarStates: cars}); err != nil {
			t.Fatal(err)
		}
		maxSeen := float32(0)
		for i := 0; i < 90; i++ {
			p = next()
			if len(p.Players) > 0 && p.Players[0].Boost > maxSeen {
				maxSeen = p.Players[0].Boost
			}
		}
		result[storyNumber(target)] = maxSeen
		result["phase"] = p.MatchInfo.MatchPhase.String()
		result["seconds"] = p.MatchInfo.SecondsElapsed
	}
	conn.SendPacket(&flat.DesiredGameStateT{CarStates: []*flat.DesiredCarStateT{{BoostAmount: &flat.FloatT{Val: 100}}, {}}, MatchInfo: &flat.DesiredMatchInfoT{GameSpeed: &flat.FloatT{Val: 0.85}}})
	speed := float32(1)
	for i := 0; i < 90; i++ {
		p = next()
		if p.MatchInfo != nil {
			speed = p.MatchInfo.GameSpeed
		}
	}
	result["focus_game_speed"] = speed
	conn.SendPacket(&flat.DesiredGameStateT{MatchInfo: &flat.DesiredMatchInfoT{GameSpeed: &flat.FloatT{Val: 1}}})
	bytes, _ := json.MarshalIndent(result, "", "  ")
	path := os.Getenv("RLSTORY_PROBE_REPORT")
	if path != "" {
		if err = os.WriteFile(path, bytes, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Log(string(bytes))
}
func storyNumber(n float32) string { b, _ := json.Marshal(n); return string(b) }
