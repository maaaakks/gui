package main

import (
	"fmt"
	"github.com/RLBot/go-interface/flat"
	"math"
	"testing"
)

func TestReserveIntegerTargetsPreserveFuelBudget(t *testing.T) {
	for _, tier := range []int{1, 2, 3} {
		for _, delay := range []int{1, 2, 3} {
			t.Run(fmt.Sprintf("tier%d/delay%d", tier, delay), func(t *testing.T) {
				s := newStorySave()
				s.Modules["reserve"] = tier
				s.Equipped = []string{"reserve"}
				r := newStoryRuntime(s, StoryChallenge{})
				boost := float32(100)
				queue := map[int]float32{}
				for i := 0; i <= 240; i++ {
					if value, ok := queue[i]; ok {
						boost = value
						delete(queue, i)
					}
					if i > 0 {
						boost = max(float32(0), boost-100.0/3.0/120)
					}
					p := runtimePacket(float32(i)/120, boost)
					p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
					state := r.step(p, 0, false)
					if state != nil && state.CarStates[0].BoostAmount != nil {
						// RLBotCore casts desired boost to an integer.
						queue[i+delay] = float32(math.Floor(float64(state.CarStates[0].BoostAmount.Val)))
					}
				}
				want := r.capacity() - 2*100.0/3.0
				if math.Abs(float64(boost+r.Extra-want)) > 1.5 {
					t.Fatalf("Fuel lost or created: native %.3f reserve %.3f want total %.3f", boost, r.Extra, want)
				}
			})
		}
	}
}

func TestIgnoredTransferWhileBoostingDoesNotSpendReserve(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.initialFuelDone = true
	for i := 0; i < 60; i++ {
		p := runtimePacket(float32(i)/120, 99-float32(i)*100.0/3.0/120)
		p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
		r.step(p, 0, false) // Deliberately ignore every returned state.
	}
	if math.Abs(float64(r.Extra-25)) > .001 {
		t.Fatal("Unapplied transfers consumed fuel", r.Extra)
	}
}

func TestFractionalReserveNeverLowersNativeFuel(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.initialFuelDone = true
	r.Extra = .1
	state := r.step(runtimePacket(.01, 98.8), 0, false)
	if state != nil || r.pendingAmount != 0 || r.Extra != .1 {
		t.Fatal("A fractional target would be truncated below the native tank")
	}
	r.Extra = .4
	state = r.step(runtimePacket(.02, 98.7), 0, false)
	if state == nil || state.CarStates[0].BoostAmount.Val != 99 {
		t.Fatal("Small remainder must still transfer once a whole-unit target is reachable")
	}
}

// Minimal sequence derived from the live capture: the native meter stays at
// exactly 99 while boost is held and reserve top-ups arrive. Charging one
// requested unit per packet empties 25 reserve in ~0.2 s rather than ~0.75 s.
func TestReserveAccountsForObservedQuantizedDelivery(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.initialFuelDone = true
	p := runtimePacket(.01, 99)
	p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
	r.step(p, 0, false)
	for i := 1; i <= 20; i++ {
		p = runtimePacket(.01+float32(i)/120, 99)
		p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
		r.step(p, 0, false)
	}
	want := float32(25) - 20*(100.0/3.0/120)
	if math.Abs(float64(r.Extra-want)) > .1 {
		t.Fatalf("Reserve burned faster than actual native delivery: got %.3f want %.3f", r.Extra, want)
	}
}

func TestRewardOverflowCanBeSpentWithoutWholeUnitLoss(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.initialFuelDone = true
	r.Extra = 0
	main := float32(100)
	r.gain("surge", 15, 0, &main, false)
	if r.Extra != 15 || main != 100 {
		t.Fatal("Reward at full native tank was not banked", r.Extra, main)
	}
	p := runtimePacket(.01, 99)
	p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
	r.step(p, 0, false)
	for i := 1; i <= 20; i++ {
		p = runtimePacket(.01+float32(i)/120, 99)
		p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
		r.step(p, 0, false)
	}
	want := float32(15) - 20*(100.0/3.0/120)
	if math.Abs(float64(r.Extra-want)) > .1 {
		t.Fatal("Overflow was charged for fuel not delivered", r.Extra, want)
	}
}
