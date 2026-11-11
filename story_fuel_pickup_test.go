package main

import (
	"math"
	"testing"

	"github.com/RLBot/go-interface/flat"
)

func reservePickupRuntime() *StoryRuntime {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.initialFuelDone = true
	r.field = &flat.FieldInfoT{BoostPads: []*flat.BoostPadT{{Location: &flat.Vector3T{}, IsFullBoost: true}}}
	return r
}

func pickupPacket(seconds, boost float32, available bool) *flat.GamePacketT {
	p := runtimePacket(seconds, boost)
	p.BoostPads = []*flat.BoostPadStateT{{IsActive: available}}
	return p
}

func TestLargePadCanRefillReserveAtFullNativeTank(t *testing.T) {
	r := reservePickupRuntime()
	r.Extra = 10
	state := r.step(pickupPacket(.01, 100, true), 0, false)
	if state == nil || state.CarStates[0].BoostAmount.Val != 99 || r.Extra != 10 {
		t.Fatal("Must make room for a real pickup without inventing reserve fuel")
	}
	// The lowering packet arrived, but the native pickup is not reported yet.
	// Reserve top-ups must not cancel the opening by immediately restoring 100.
	state = r.step(pickupPacket(.02, 99, true), 0, false)
	if state != nil || r.Extra != 10 {
		t.Fatal("Reserve competed with the native pad pickup")
	}
	// Rocket League collected it: native fuel is 100 and the pad is cooling down.
	r.step(pickupPacket(.03, 100, false), 0, false)
	if r.Extra != 25 || r.pendingAmount != 0 {
		t.Fatal("Confirmed large-pad pickup failed to fill the reserve", r.Extra)
	}
	r.Extra = 5
	r.step(pickupPacket(.04, 100, false), 0, false)
	if r.Extra != 5 {
		t.Fatal("Unavailable pad refilled the reserve again")
	}
}

func TestPickupOpeningOnlyAtEligiblePad(t *testing.T) {
	for _, name := range []string{"far", "airborne", "unavailable", "full reserve", "vanilla", "paused", "countdown", "demolished", "contested", "overdrive"} {
		t.Run(name, func(t *testing.T) {
			r := reservePickupRuntime()
			r.Extra = 10
			p := pickupPacket(.01, 100, true)
			switch name {
			case "far":
				p.Players[0].Physics.Location.X = 500
			case "airborne":
				p.Players[0].Physics.Location.Z = 500
			case "unavailable":
				p.BoostPads[0].IsActive = false
			case "full reserve":
				r.Extra = 25
			case "vanilla":
				r.save.Vanilla = true
			case "paused":
				p.MatchInfo.MatchPhase = flat.MatchPhasePaused
			case "countdown":
				p.MatchInfo.MatchPhase = flat.MatchPhaseCountdown
			case "demolished":
				p.Players[0].DemolishedTimeout = 2
			case "contested":
				p.Players[1].Physics.Location.X = 0
			case "overdrive":
				r.OverdriveUntil = 10
			}
			if state := r.step(p, 0, false); state != nil && state.CarStates[0].BoostAmount != nil {
				t.Fatal("Ineligible pad interaction altered native fuel")
			}
		})
	}
}

func TestSmallPadsRefillReserveAt100(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		r := reservePickupRuntime()
		r.field.BoostPads[0].IsFullBoost = false
		r.Extra = 0
		state := r.step(pickupPacket(.01, 100, true), 0, false)
		if state == nil || state.CarStates[0].BoostAmount.Val != 99 || r.Extra != 0 {
			t.Fatal("Full native tank must allow a small-pad pickup, without early rewards")
		}
		if delayed {
			if state = r.step(pickupPacket(.02, 99, true), 0, false); state != nil {
				t.Fatal("Top-up prevented small-pad pickup")
			}
		}
		r.step(pickupPacket(.03, 100, false), 0, false)
		if r.Extra != 12 {
			t.Fatal("Small pad at 100 must store 12 including the artificial opening", delayed, r.Extra)
		}
		r.step(pickupPacket(.04, 100, false), 0, false)
		if r.Extra != 12 {
			t.Fatal("Cooling pad awarded fuel repeatedly")
		}
	}
}

func TestSmallPadStoresOnlyNativeOverflow(t *testing.T) {
	for _, sample := range []struct{ before, reserve, want float32 }{{88, 0, 0}, {93, 0, 5}, {98, 10, 20}, {100, 24, 25}, {100, 25, 25}} {
		r := reservePickupRuntime()
		r.field.BoostPads[0].IsFullBoost = false
		r.Extra = sample.reserve
		r.step(pickupPacket(.01, sample.before, true), 0, false)
		r.step(pickupPacket(.02, 100, false), 0, false)
		if r.Extra != sample.want {
			t.Fatal("Small pad duplicated native fuel or exceeded reserve capacity", sample, r.Extra)
		}
	}
}

func TestSmallPadOverflowAccountsForBoostHeldDuringOpening(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		r := reservePickupRuntime()
		r.field.BoostPads[0].IsFullBoost = false
		r.Extra = 0
		p := pickupPacket(.01, 100, true)
		p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
		r.step(p, 0, false)
		if delayed {
			p = pickupPacket(.02, 99, true)
			p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
			r.step(p, 0, false)
		}
		p = pickupPacket(.03, 100, false)
		p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
		r.step(p, 0, false)
		want := float32(12 - 2.0/3.0)
		if math.Abs(float64(r.Extra-want)) > .001 {
			t.Fatal("Opening or held boost caused a fuel accounting error", delayed, r.Extra, want)
		}
	}
}

func TestReserveWorksFromFirstKickoff(t *testing.T) {
	r := reservePickupRuntime()
	r.initialFuelDone = false
	p := runtimePacket(0, 100)
	p.MatchInfo.MatchPhase = flat.MatchPhaseCountdown
	r.step(p, 0, false)
	if r.Extra != 25 || p.Players[0].Boost+r.Extra != 125 {
		t.Fatal("First countdown must start with 100 native plus 25 reserve")
	}
	p = runtimePacket(.01, 99)
	p.MatchInfo.MatchPhase = flat.MatchPhaseKickoff
	p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
	state := r.step(p, 0, false)
	if state == nil || state.CarStates[0].BoostAmount.Val != 100 {
		t.Fatal("Reserve must take over before the kickoff ball touch")
	}
	p = runtimePacket(.04, 100)
	p.MatchInfo.MatchPhase = flat.MatchPhaseKickoff
	p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
	r.step(p, 0, false)
	if r.Extra < 22.99 || r.Extra > 23.01 || r.activeTime != 0 || r.Charge != 0 {
		t.Fatal("Kickoff fuel accounting changed Flow or failed to account for delivery", r.Extra)
	}
}

func TestIgnoredKickoffTransferRetriesBeforeBallTouch(t *testing.T) {
	r := reservePickupRuntime()
	for _, seconds := range []float32{0, .15} {
		p := runtimePacket(seconds, 98)
		p.MatchInfo.MatchPhase = flat.MatchPhaseKickoff
		r.step(p, 0, false)
	}
	p := runtimePacket(.22, 98)
	p.MatchInfo.MatchPhase = flat.MatchPhaseKickoff
	state := r.step(p, 0, false)
	if state == nil || state.CarStates[0].BoostAmount.Val != 100 || r.Extra != 25 {
		t.Fatal("Kickoff transfer retry waited for the Active phase")
	}
}
