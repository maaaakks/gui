package main

import (
	"github.com/RLBot/go-interface/flat"
	"testing"
)

func TestFlowDoesNotConsumeChargeWhileDemolishedOrPaused(t *testing.T) {
	s := newStorySave()
	storyTestWins(&s, 5)
	for _, phase := range []flat.MatchPhase{flat.MatchPhaseActive, flat.MatchPhasePaused, flat.MatchPhaseReplay} {
		r := newStoryRuntime(s, StoryChallenge{})
		r.step(runtimePacket(0, 100), 0, false)
		r.Charge = 100
		r.activeTime = 30
		p := runtimePacket(.1, 20)
		p.MatchInfo.MatchPhase = phase
		if phase == flat.MatchPhaseActive {
			p.Players[0].DemolishedTimeout = 1.5
		}
		r.step(p, 0, true)
		if r.Uses != 0 || r.Charge != 100 {
			t.Fatal("An unavailable car consumed a Flow charge", phase)
		}
	}
}
