package main

import "testing"

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
	r.step(runtimePacket(.1, 98), 0, false)
	r.step(runtimePacket(.12, 97), 0, false)
	// Transfer was applied, but continuous boosting means no sample near 100.
	r.step(runtimePacket(.14, 98.5), 0, false)
	if r.Extra != 23 || r.reserveFault {
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
	r.activeTime = 2
	r.pendingSince = 0
	state := r.step(runtimePacket(.1, 85), 0, false)
	if state == nil || state.CarStates[0].BoostAmount == nil || state.CarStates[0].BoostAmount.Val != 100 || r.reserveFault {
		t.Fatal("Missed tiny transfer permanently stranded reserve")
	}
	r.step(runtimePacket(.2, 99.5), 0, false)
	if r.Extra != 10 {
		t.Fatal("Recovery transfer not charged once", r.Extra)
	}
}
