package main

import (
	"github.com/RLBot/go-interface/flat"
	"testing"
)

// Simulate native fuel consumption between receipt of a transfer and the next
// packet. At lower packet rates the gauge never visibly rises to 100.
func TestReserveContinuousBoostRoundtrip(t *testing.T) {
	for _, fps := range []float32{15, 60, 120} {
		s := newStorySave()
		s.Modules["reserve"] = 1
		s.Equipped = []string{"reserve"}
		r := newStoryRuntime(s, StoryChallenge{})
		r.step(runtimePacket(0, 100), 0, false)
		boost := float32(100)
		dt := 1 / fps
		for i := 1; i <= int(fps*2); i++ {
			boost = max(float32(0), boost-33.33333*dt)
			p := runtimePacket(float32(i)*dt, boost)
			p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
			state := r.step(p, 0, false)
			if state != nil && state.CarStates[0].BoostAmount != nil {
				boost = state.CarStates[0].BoostAmount.Val
			}
		}
		// 100 + 25 - 2 seconds * 33.33 = 58.33. Small state-setting latency
		// may leave a sub-point residue, but must not strand the whole reserve.
		if r.Extra > 1 || boost < 56 || boost > 61 {
			t.Fatalf("fps=%v: reserve=%v boost=%v", fps, r.Extra, boost)
		}
	}
}

func TestReserveRefillsAndCanBeUsedAgain(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.field = &flat.FieldInfoT{BoostPads: []*flat.BoostPadT{{Location: &flat.Vector3T{}, IsFullBoost: true}}}
	p := runtimePacket(0, 100)
	p.BoostPads = []*flat.BoostPadStateT{{IsActive: true}}
	r.step(p, 0, false)
	r.Extra = 0
	p = runtimePacket(.1, 100)
	p.BoostPads = []*flat.BoostPadStateT{{IsActive: false}}
	r.step(p, 0, false)
	if r.Extra != 25 {
		t.Fatal("Large pad did not replenish reserve", r.Extra)
	}
	p = runtimePacket(.2, 90)
	p.BoostPads = []*flat.BoostPadStateT{{IsActive: false}}
	state := r.step(p, 0, false)
	if state == nil || state.CarStates[0].BoostAmount == nil || state.CarStates[0].BoostAmount.Val != 100 {
		t.Fatal("Refilled reserve not usable")
	}
	r.step(runtimePacket(.3, 100), 0, false)
	if r.Extra != 15 {
		t.Fatal("Second transfer not charged", r.Extra)
	}
}

func TestFlowKeepsChargeAtFullFuelAndRequiresFreshPress(t *testing.T) {
	s := newStorySave()
	storyTestWins(&s, 5)
	r := newStoryRuntime(s, StoryChallenge{})
	p := runtimePacket(0, 20)
	p.Players[0].LastInput = &flat.ControllerStateT{UseItem: true}
	r.initialFuelDone = true
	r.Charge = 100
	r.activeTime = 30
	r.step(p, 0, false)
	if r.Uses != 0 {
		t.Fatal("Held button activated Flow on initialization")
	}
	r.step(runtimePacket(.1, 100), 0, false)
	r.step(runtimePacket(.2, 100), 0, true)
	if r.Charge != 100 || r.Uses != 0 {
		t.Fatal("Surge wasted charge at full fuel")
	}
	state := r.step(runtimePacket(.3, 20), 0, true)
	if r.Uses != 1 || state == nil || state.CarStates[0].BoostAmount.Val != 70 {
		t.Fatal("Surge did not activate when useful")
	}
}

func TestOverdriveSurvivesPause(t *testing.T) {
	s := newStorySave()
	storyTestWins(&s, 5)
	s.FlowMode = "overdrive"
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.Charge = 100
	r.activeTime = 30
	r.step(runtimePacket(.1, 20), 0, true)
	until := r.OverdriveUntil
	p := runtimePacket(.2, 100)
	p.MatchInfo.MatchPhase = flat.MatchPhasePaused
	r.step(p, 0, false)
	if r.OverdriveUntil != until {
		t.Fatal("Pause cancelled Overdrive")
	}
}

func TestReserveRetriesWithoutWaitingForLowTank(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.reserveFault = true
	r.pendingSince = 0
	r.fuelTime = 2
	state := r.step(runtimePacket(.1, 98), 0, false)
	if state == nil || state.CarStates[0].BoostAmount == nil || state.CarStates[0].BoostAmount.Val != 100 {
		t.Fatal("Reserve still stranded at high boost")
	}
}
