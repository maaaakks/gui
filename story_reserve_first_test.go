package main

import "testing"

func TestReserveConsumedBeforeMainTank(t *testing.T) {
	s := newStorySave()
	s.Modules = map[string]int{"reserve": 1}
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.Extra = 25
	state := r.step(runtimePacket(.1, 99), 0, false)
	if state == nil || state.CarStates[0].BoostAmount == nil || state.CarStates[0].BoostAmount.Val != 100 {
		t.Fatal("Reserve must top up at 99, not wait until 25")
	}
	if r.Extra != 25 {
		t.Fatal("Keep reserve until transfer is acknowledged")
	}
	// An unchanged packet must not acknowledge a small refill.
	r.step(runtimePacket(.11, 99), 0, false)
	if r.Extra != 25 {
		t.Fatal("Unapplied top-up consumed reserve")
	}
	// The next packet includes consumption since the refill was applied.
	r.step(runtimePacket(.12, 99.7), 0, false)
	if r.Extra != 24 {
		t.Fatal("Applied top-up must consume exactly one reserve boost", r.Extra)
	}
	// Exhausting the reserve must not generate any further fuel.
	r.Extra = 0
	state = r.step(runtimePacket(.2, 60), 0, false)
	if state != nil && state.CarStates[0].BoostAmount != nil {
		t.Fatal("Empty reserve refilled main tank")
	}
}

func TestReserveTransferDoesNotCreateFuelWhenIgnored(t *testing.T) {
	s := newStorySave()
	s.Modules = map[string]int{"reserve": 1}
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.Extra = 25
	r.step(runtimePacket(.1, 98), 0, false)
	retries := 0
	for _, sec := range []float32{.2, .3, .4, .5, .6, .7} {
		state := r.step(runtimePacket(sec, 98), 0, false)
		if state != nil && state.CarStates[0].BoostAmount != nil {
			retries++
		}
	}
	if r.Extra != 25 || retries < 1 || retries > 3 {
		t.Fatal("Ignored transfer should retry at a bounded rate without consuming reserve", r.Extra, retries)
	}
}
