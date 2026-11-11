package main

import (
	"github.com/RLBot/go-interface/flat"
	"math"
	"testing"
)

func TestReserveAcknowledgesWhileGaugeKeepsFalling(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	p := runtimePacket(0, 100)
	p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
	r.step(p, 0, false)
	p = runtimePacket(.06, 98)
	p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
	r.step(p, 0, false)
	// The transfer arrived, then another 3 boost was spent before the packet.
	// 97 is below both 98 and 100; the older rise-only check waits for timeout.
	p = runtimePacket(.15, 97)
	p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
	r.step(p, 0, false)
	if r.Extra != 23 || r.reserveFault {
		t.Fatal("Held boost hid a successful transfer", r.Extra, r.reserveFault)
	}
}

func TestReserveRetriesBeforeQuarterSecond(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.step(runtimePacket(.01, 98), 0, false)
	r.step(runtimePacket(.15, 98), 0, false)
	state := r.step(runtimePacket(.22, 98), 0, false)
	if state == nil || state.CarStates[0].BoostAmount == nil || r.Extra != 25 {
		t.Fatal("Reserve recovery still stalls after an ignored packet")
	}
}

func TestReserveFuelBudgetWithDelayedApplication(t *testing.T) {
	for _, delay := range []int{1, 2, 3} {
		s := newStorySave()
		s.Modules["reserve"] = 1
		s.Equipped = []string{"reserve"}
		r := newStoryRuntime(s, StoryChallenge{})
		r.step(runtimePacket(0, 100), 0, false)
		boost := float32(100)
		queue := map[int]float32{}
		for i := 1; i <= 240; i++ {
			if value, ok := queue[i]; ok {
				boost = value
				delete(queue, i)
			}
			boost = max(float32(0), boost-100.0/3.0/120)
			p := runtimePacket(float32(i)/120, boost)
			p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
			state := r.step(p, 0, false)
			if state != nil && state.CarStates[0].BoostAmount != nil {
				queue[i+delay] = state.CarStates[0].BoostAmount.Val
			}
		}
		if r.Extra > 0.1 || boost < 57 || boost > 60 {
			t.Fatal("Latency created fuel or stranded reserve", delay, boost, r.Extra)
		}
	}
}

func TestFullReserveAtMatchStart(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	if r.Extra != 25 || r.capacity() != 125 {
		t.Fatal("Equipped capacity must start usable, not empty")
	}
}
func TestReserveAcknowledgesDelayedTransferWhileBoosting(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	for _, sample := range []struct{ sec, boost float32 }{{.1, 98}, {.12, 98 - 2.0/3.0}, {.14, 100 - 4.0/3.0}} {
		p := runtimePacket(sample.sec, sample.boost)
		p.Players[0].LastInput = &flat.ControllerStateT{Boost: true}
		r.step(p, 0, false)
	}
	// Transfer was applied, but continuous boosting means no sample near 100.
	if math.Abs(float64(r.Extra-23)) > .001 || r.reserveFault {
		t.Fatal("Visible rise must acknowledge transfer during sustained boost", r.Extra)
	}
}
func TestReserveCanRecoverWithoutRestartingMatch(t *testing.T) {
	s := newStorySave()
	s.Modules["reserve"] = 1
	s.Equipped = []string{"reserve"}
	r := newStoryRuntime(s, StoryChallenge{})
	r.step(runtimePacket(0, 100), 0, false)
	r.reserveFault = true
	r.fuelTime = 2
	r.pendingSince = 0
	state := r.step(runtimePacket(.1, 85), 0, false)
	if state == nil || state.CarStates[0].BoostAmount == nil || state.CarStates[0].BoostAmount.Val != 100 || r.reserveFault {
		t.Fatal("Missed tiny transfer permanently stranded reserve")
	}
	r.step(runtimePacket(.2, 99.5), 0, false)
	if r.Extra != 10.5 {
		t.Fatal("Recovery transfer not charged once", r.Extra)
	}
}
