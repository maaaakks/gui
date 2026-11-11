package main

import (
	"fmt"
	rlbot "github.com/RLBot/go-interface"
	"github.com/RLBot/go-interface/flat"
	"math"
	"sort"
)

type storyFuelGain struct{ at, amount float32 }

const (
	storyReserveAckTimeout float32 = 0.12
	storyReserveRetryDelay float32 = 0.20
)

type storyTouchEvent struct {
	index int
	at    float32
	ball  uint32
}
type StoryRuntime struct {
	initialFuelDone                            bool
	save                                       StorySave
	challenge                                  StoryChallenge
	field                                      *flat.FieldInfoT
	activeTime, lastSeconds                    float32
	wasActive, started                         bool
	fuelTime                                   float32
	wasFuelActive                              bool
	previousStats                              flat.ScoreInfoT
	previousTouches                            map[int]float32
	cooldown                                   map[string]float32
	gains                                      []storyFuelGain
	lastPads                                   []bool
	previousFuel                               float32
	padOpening                                 int
	padOpeningFuel, padOpeningSpent            float32
	passAt                                     float32
	passBall                                   uint32
	passPosition                               *flat.Vector3T
	passPending                                bool
	Extra                                      float32
	pendingAmount, pendingTarget, pendingSince float32
	pendingLowest                              float32
	pendingSpent                               float32
	wasBoosting                                bool
	reserveFault                               bool
	Charge                                     float32
	Uses                                       int
	OverdriveUntil, flowStored                 float32
	itemDown                                   bool
	TotalAdded                                 float32
	LastEvent                                  string
}

func newStoryRuntime(save StorySave, c StoryChallenge) *StoryRuntime {
	return &StoryRuntime{itemDown: true, padOpening: -1, Extra: storyModuleValue(save, "reserve"), save: save, challenge: c, previousTouches: map[int]float32{}, cooldown: map[string]float32{}}
}
func (r *StoryRuntime) capacity() float32 { return 100 + storyModuleValue(r.save, "reserve") }
func (r *StoryRuntime) gain(id string, amount, cooldown float32, target *float32, frequent bool) {
	if last, ok := r.cooldown[id]; ok && r.activeTime-last < cooldown {
		return
	}
	if frequent {
		used := float32(0)
		fresh := r.gains[:0]
		for _, g := range r.gains {
			if r.activeTime-g.at < 60 {
				fresh = append(fresh, g)
				used += g.amount
			}
		}
		r.gains = fresh
		amount = min(amount, max(float32(0), 60-used))
	}
	amount = min(amount, max(float32(0), r.capacity()-*target-r.Extra))
	if amount <= 0 {
		return
	}
	main := min(amount, 100-*target)
	*target += main
	r.Extra += amount - main
	r.cooldown[id] = r.activeTime
	r.TotalAdded += amount
	if frequent {
		r.gains = append(r.gains, storyFuelGain{r.activeTime, amount})
	}
	r.LastEvent = fmt.Sprintf("%s +%.0f boost", id, amount)
}
func storyDistance2(a, b *flat.Vector3T) float32 {
	if a == nil || b == nil {
		return float32(math.Inf(1))
	}
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return dx*dx + dy*dy + dz*dz
}
func (r *StoryRuntime) passes(p *flat.GamePacketT, human int, target *float32) {
	events := []storyTouchEvent{}
	for i, car := range p.Players {
		touch := car.LatestTouch
		if touch == nil {
			continue
		}
		last, ok := r.previousTouches[i]
		r.previousTouches[i] = touch.GameSeconds
		if ok && touch.GameSeconds > last {
			events = append(events, storyTouchEvent{i, touch.GameSeconds, touch.BallIndex})
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].at < events[j].at })
	for j, e := range events {
		if (j > 0 && events[j-1].at == e.at) || (j+1 < len(events) && events[j+1].at == e.at) {
			r.passPending = false
			continue
		}
		car := p.Players[e.index]
		if e.index == human {
			r.passPending = true
			r.passAt = e.at
			r.passBall = e.ball
			if car.Physics != nil && car.Physics.Location != nil {
				pos := *car.Physics.Location
				r.passPosition = &pos
			}
			continue
		}
		if car.Team != p.Players[human].Team {
			r.passPending = false
			continue
		}
		if r.passPending && e.ball == r.passBall && e.at-r.passAt >= 0.25 && e.at-r.passAt <= 4 && car.Physics != nil && storyDistance2(r.passPosition, car.Physics.Location) >= 800*800 {
			r.gain("pass", storyModuleValue(r.save, "passing"), 8, target, true)
			r.chargeAction()
			r.passPending = false
		}
	}
}
func (r *StoryRuntime) chargeAction() {
	if r.OverdriveUntil > r.activeTime {
		return
	}
	if last, ok := r.cooldown["flow-action"]; !ok || r.activeTime-last >= 8 {
		r.Charge = min(float32(100), r.Charge+6)
		r.cooldown["flow-action"] = r.activeTime
	}
}
func (r *StoryRuntime) pads(p *flat.GamePacketT, human int, spent float32) {
	if len(r.lastPads) != len(p.BoostPads) {
		r.lastPads = make([]bool, len(p.BoostPads))
		for i, pad := range p.BoostPads {
			r.lastPads[i] = pad.IsActive
		}
		return
	}
	if r.field == nil || storyModuleValue(r.save, "reserve") == 0 {
		return
	}
	for i, pad := range p.BoostPads {
		collected := r.lastPads[i] && !pad.IsActive
		r.lastPads[i] = pad.IsActive
		if !collected || i >= len(r.field.BoostPads) {
			continue
		}
		h := p.Players[human]
		if h.Physics == nil {
			continue
		}
		loc := r.field.BoostPads[i].Location
		distance := storyDistance2(h.Physics.Location, loc)
		if distance > 400*400 {
			continue
		}
		unambiguous := true
		for j, other := range p.Players {
			if j != human && other.Physics != nil && storyDistance2(other.Physics.Location, loc) <= distance+100*100 {
				unambiguous = false
			}
		}
		if unambiguous {
			r.pendingAmount = 0
			capacity := storyModuleValue(r.save, "reserve")
			if r.field.BoostPads[i].IsFullBoost {
				r.Extra = capacity
				r.LastEvent = "Auxiliary tank refilled"
			} else {
				// Native boost already includes the portion received by the
				// main tank. Store only the unused part of this twelve-unit pad.
				nativeGain := max(float32(0), h.Boost-r.previousFuel+spent)
				if r.padOpening == i {
					// Measure from before the artificial opening, returning its
					// one point without counting native pad fuel twice.
					nativeGain = max(float32(0), h.Boost-r.padOpeningFuel+r.padOpeningSpent)
				}
				overflow := 12 - nativeGain
				added := min(max(float32(0), min(float32(12), overflow)), max(float32(0), capacity-r.Extra))
				r.Extra += added
				if added > 0 {
					r.LastEvent = fmt.Sprintf("Auxiliary tank +%.0f boost", added)
				}
			}
			if r.padOpening == i {
				r.padOpening = -1
				r.padOpeningSpent = 0
			}
			r.reserveFault = false
		}
	}
}

// A native pad pickup can raise boost independently of a pending transfer.
// Do not charge that native pickup as extra reserve delivery.
func (r *StoryRuntime) humanPickedPad(p *flat.GamePacketT, human int) bool {
	if r.field == nil || p.Players[human].Physics == nil {
		return false
	}
	for i, pad := range p.BoostPads {
		if i >= len(r.lastPads) || i >= len(r.field.BoostPads) || !r.lastPads[i] || pad.IsActive {
			continue
		}
		loc := r.field.BoostPads[i].Location
		distance := storyDistance2(p.Players[human].Physics.Location, loc)
		if distance > 400*400 {
			continue
		}
		nearest := true
		for j, other := range p.Players {
			if j != human && other.Physics != nil && storyDistance2(other.Physics.Location, loc) <= distance+100*100 {
				nearest = false
			}
		}
		if nearest {
			return true
		}
	}
	return false
}

// At 100 native boost Rocket League leaves the pickup active. Briefly make
// room only while touching an available pad, so the game itself collects
// it and owns the normal respawn timer for every player.
func (r *StoryRuntime) touchingPad(p *flat.GamePacketT, human int) int {
	car := p.Players[human]
	if r.field == nil || car.Physics == nil || car.Physics.Location == nil {
		return -1
	}
	for i, pad := range p.BoostPads {
		if pad == nil || !pad.IsActive || i >= len(r.field.BoostPads) {
			continue
		}
		info := r.field.BoostPads[i]
		if info == nil || info.Location == nil {
			continue
		}
		pos, loc := car.Physics.Location, info.Location
		dx, dy := pos.X-loc.X, pos.Y-loc.Y
		radius, height := float32(120), float64(100)
		if info.IsFullBoost {
			radius, height = 200, 150
		}
		if dx*dx+dy*dy > radius*radius || math.Abs(float64(pos.Z-loc.Z)) > height {
			continue
		}
		nearest := true
		distance := storyDistance2(pos, loc)
		for j, other := range p.Players {
			if j != human && other.Physics != nil && storyDistance2(other.Physics.Location, loc) <= distance+100*100 {
				nearest = false
			}
		}
		if nearest {
			return i
		}
	}
	return -1
}
func (r *StoryRuntime) step(p *flat.GamePacketT, human int, trigger bool) *flat.DesiredGameStateT {
	if p.MatchInfo == nil || human < 0 || human >= len(p.Players) {
		return nil
	}
	car := p.Players[human]
	boosting := car.LastInput != nil && car.LastInput.Boost
	wasBoosting := r.wasBoosting
	r.wasBoosting = boosting
	phase := p.MatchInfo.MatchPhase
	active := phase == flat.MatchPhaseActive
	fuelActive := active || phase == flat.MatchPhaseKickoff
	dt := p.MatchInfo.SecondsElapsed - r.lastSeconds
	r.lastSeconds = p.MatchInfo.SecondsElapsed
	if dt < 0 || dt > 0.25 {
		r.passPending = false
		dt = 0
	}
	if active && r.wasActive {
		r.activeTime += dt
	}
	r.wasActive = active
	if fuelActive && r.wasFuelActive {
		r.fuelTime += dt
	}
	r.wasFuelActive = fuelActive
	if r.save.Vanilla {
		return nil
	}
	defer func() { r.previousFuel = car.Boost }()
	target := car.Boost
	item := car.LastInput != nil && car.LastInput.UseItem
	pressed := item && !r.itemDown
	r.itemDown = item
	if !r.started {
		r.started = true
		if car.ScoreInfo != nil {
			r.previousStats = *car.ScoreInfo
		}
		for i, c := range p.Players {
			if c.LatestTouch != nil {
				r.previousTouches[i] = c.LatestTouch.GameSeconds
			}
		}
	}
	if !r.initialFuelDone && phase != flat.MatchPhaseEnded && phase != flat.MatchPhaseInactive {
		target = 100
		r.initialFuelDone = active || car.Boost >= 99
	}
	if r.pendingAmount > 0 {
		// The refill can have arrived even if the native gauge never rises:
		// holding normal boost consumes 100/3 units per active second.
		// Account only for intervals bracketed by reported held inputs.
		if fuelActive && wasBoosting && boosting {
			r.pendingSpent += (100.0 / 3.0) * dt
		}
		// A fixed ten-point tolerance acknowledges tiny top-ups before they
		// are applied. Scale it to the amount actually transferred instead.
		tolerance := min(float32(0.75), r.pendingAmount*0.75)
		// Under sustained boost the packet can already be below the requested
		// target. A rise from the lowest observed native level also proves
		// that the pending top-up arrived; do not require a near-100 sample.
		rose := car.Boost > r.pendingLowest+min(float32(0.1), r.pendingAmount/4)
		// The core applies integer boost targets asynchronously. At 120 Hz a
		// request for one unit can deliver only a fraction before the next
		// packet. Debit observed delivery, not a minimum of the request size.
		debit := max(float32(0), car.Boost+r.pendingSpent-(r.pendingTarget-r.pendingAmount))
		pickedPad := r.humanPickedPad(p, human)
		if pickedPad || (debit > 0.05 && (car.Boost+r.pendingSpent >= r.pendingTarget-tolerance || rose)) {
			// Native pad fuel cannot be distinguished from a simultaneous
			// transfer. Leave it free rather than charging it to the reserve.
			if pickedPad {
				debit = 0
			}
			if debit > r.Extra {
				target = max(float32(0), target-(debit-r.Extra))
			}
			r.Extra = max(float32(0), r.Extra-debit)
			r.pendingAmount = 0
		} else if r.fuelTime-r.pendingSince > storyReserveAckTimeout {
			r.pendingAmount = 0
			r.reserveFault = true
			r.LastEvent = "Auxiliary transfer delayed; retrying"
		}
		r.pendingLowest = min(r.pendingLowest, car.Boost)
	}
	if car.DemolishedTimeout > 0 {
		r.Extra = 0
		r.pendingAmount = 0
		r.padOpening = -1
		r.padOpeningSpent = 0
		r.passPending = false
	}
	spent := float32(0)
	if fuelActive && wasBoosting && boosting {
		spent = (100.0 / 3.0) * dt
	}
	if r.padOpening >= 0 {
		r.padOpeningSpent += spent
	}
	r.pads(p, human, spent)
	flowReady := !r.save.Vanilla && storyWins(r.save) >= 5
	if active && flowReady && r.OverdriveUntil <= r.activeTime {
		r.Charge = min(float32(100), r.Charge+dt)
	}
	if active && car.ScoreInfo != nil {
		stats := car.ScoreInfo
		if stats.Saves > r.previousStats.Saves {
			r.gain("save", storyModuleValue(r.save, "saves"), 8, &target, true)
			r.chargeAction()
		}
		if stats.Shots > r.previousStats.Shots {
			r.gain("shot", storyModuleValue(r.save, "shots"), 6, &target, true)
			r.chargeAction()
		}
		if stats.Demolitions > r.previousStats.Demolitions {
			r.gain("demo", storyModuleValue(r.save, "demos"), 20, &target, false)
		}
		r.passes(p, human, &target)
	}
	if car.ScoreInfo != nil {
		r.previousStats = *car.ScoreInfo
	}
	if !active {
		r.passPending = false
		for i, c := range p.Players {
			if c.LatestTouch != nil {
				r.previousTouches[i] = c.LatestTouch.GameSeconds
			}
		}
	}
	if active && car.DemolishedTimeout <= 0 && flowReady && (pressed || trigger) && r.Charge >= 100 && r.Uses < 2 && r.activeTime >= 25 {
		canActivate := r.save.FlowMode == "overdrive" || target+r.Extra < r.capacity()-0.05
		if !canActivate {
			r.LastEvent = "Fuel full — Flow charge kept"
		}
		if canActivate {
			r.Charge = 0
			r.Uses++
			tier := max(1, min(3, r.save.FlowLevels[r.save.FlowMode]))
			if r.save.FlowMode == "overdrive" {
				r.flowStored = target
				r.OverdriveUntil = r.activeTime + []float32{2, 2.5, 3}[tier-1]
				r.LastEvent = "Overdrive active"
			} else {
				r.gain("surge", []float32{50, 75, 100}[tier-1], 0, &target, false)
			}
		}
	}
	if r.OverdriveUntil > 0 {
		if active && r.activeTime < r.OverdriveUntil && car.DemolishedTimeout == 0 {
			target = 100
		} else if car.DemolishedTimeout > 0 {
			r.OverdriveUntil = 0
			r.LastEvent = "Overdrive interrupted"
		} else if active && r.activeTime >= r.OverdriveUntil {
			target = r.flowStored
			r.OverdriveUntil = 0
			r.LastEvent = "Overdrive complete"
		}
	}
	// Use auxiliary fuel first. Batch small top-ups so the native gauge stays
	// around 99–100 rather than waiting for the main tank to reach 25.
	// Never refill during a pause, demolition or an Overdrive activation.
	// Retry a missed transfer after a bounded delay.
	// Keep this bounded by the native deficit and the remaining reserve.
	contactPad := -1
	if fuelActive && car.DemolishedTimeout <= 0 && r.OverdriveUntil == 0 && r.Extra < storyModuleValue(r.save, "reserve") {
		contactPad = r.touchingPad(p, human)
	}
	padContact := contactPad >= 0
	if r.padOpening != contactPad {
		r.padOpening = -1
		r.padOpeningSpent = 0
	}
	if padContact && target >= 99.5 {
		if r.padOpening != contactPad {
			r.padOpeningFuel = target
			r.padOpeningSpent = 0
		}
		r.padOpening = contactPad
		target = 99
	}
	if fuelActive && r.reserveFault && target <= 99.5 && r.fuelTime-r.pendingSince >= storyReserveRetryDelay {
		r.reserveFault = false
	}
	if fuelActive && !padContact && car.DemolishedTimeout <= 0 && target <= 99.5 && r.Extra > 0 && r.pendingAmount == 0 && !r.reserveFault && r.OverdriveUntil == 0 {
		// The core truncates boost commands to whole units. Never send a
		// fractional refill which would round below the current native tank
		// and actually remove fuel. A sub-unit remainder waits until useful.
		nativeTarget := float32(math.Floor(float64(min(float32(100), target+r.Extra))))
		amount := nativeTarget - target
		if amount > 0.05 {
			r.pendingAmount = amount
			r.pendingTarget = nativeTarget
			r.pendingLowest = car.Boost
			r.pendingSpent = 0
			r.pendingSince = r.fuelTime
			target = nativeTarget
		}
	}
	cars := make([]*flat.DesiredCarStateT, len(p.Players))
	changed := false
	for i, c := range p.Players {
		cars[i] = &flat.DesiredCarStateT{}
		if c.Team == 1 && c.Boost > r.challenge.OpponentBoostCap && r.challenge.OpponentBoostCap > 0 {
			cars[i].BoostAmount = &flat.FloatT{Val: r.challenge.OpponentBoostCap}
			changed = true
		}
	}
	if math.Abs(float64(target-car.Boost)) > 0.05 {
		cars[human].BoostAmount = &flat.FloatT{Val: max(float32(0), min(float32(100), target))}
		changed = true
	}
	if changed {
		return &flat.DesiredGameStateT{CarStates: cars}
	}
	return nil
}
func (r *StoryRuntime) restore(conn rlbot.RLBotConnection, human, count int) {
	if r.OverdriveUntil > 0 && human >= 0 && human < count {
		cars := make([]*flat.DesiredCarStateT, count)
		for i := range cars {
			cars[i] = &flat.DesiredCarStateT{}
		}
		cars[human].BoostAmount = &flat.FloatT{Val: r.flowStored}
		_ = conn.SendPacket(&flat.DesiredGameStateT{CarStates: cars})
	}
	_ = conn.SendPacket(&flat.RemoveRenderGroupT{Id: 7420})
}
func (r *StoryRuntime) render(conn rlbot.RLBotConnection, boost float32) {
	text := ""
	if !r.save.Vanilla && storyWins(r.save) >= 5 {
		text = fmt.Sprintf("FLOW %.0f%% · %s · %d uses left", r.Charge, r.save.FlowMode, 2-r.Uses)
		if r.Charge >= 100 && r.Uses < 2 {
			text = fmt.Sprintf("FLOW READY · R / L3 · %s · %d uses left", r.save.FlowMode, 2-r.Uses)
		}
		if r.Uses >= 2 {
			text = "FLOW · no uses left this game"
		}
		if r.OverdriveUntil > r.activeTime {
			text = "OVERDRIVE"
		}
	}
	messages := []*flat.RenderMessageT{}
	if text != "" {
		messages = append(messages, &flat.RenderMessageT{Variety: &flat.RenderTypeT{Type: flat.RenderTypeString2D, Value: &flat.String2DT{Text: text, X: .025, Y: .91, Scale: .8, Foreground: &flat.ColorT{R: 235, G: 232, B: 220, A: 240}, Background: &flat.ColorT{R: 12, G: 14, B: 16, A: 150}}}})
	}
	if !r.save.Vanilla && r.capacity() > 100 {
		reserve := fmt.Sprintf("RESERVE %.0f / %.0f", r.Extra, r.capacity()-100)
		total := fmt.Sprintf("TOTAL %.0f / %.0f", min(r.capacity(), boost+r.Extra), r.capacity())
		if r.reserveFault {
			total = "SYNCING"
		}
		for i, label := range []string{reserve, total} {
			messages = append(messages, &flat.RenderMessageT{Variety: &flat.RenderTypeT{Type: flat.RenderTypeString2D, Value: &flat.String2DT{Text: label, X: .84, Y: .74 + float32(i)*.028, Scale: .75, Foreground: &flat.ColorT{R: 255, G: 206, B: 115, A: 245}, Background: &flat.ColorT{R: 12, G: 14, B: 16, A: 150}}}})
		}
	}
	if len(messages) == 0 {
		_ = conn.SendPacket(&flat.RemoveRenderGroupT{Id: 7420})
		return
	}
	_ = conn.SendPacket(&flat.RenderGroupT{Id: 7420, RenderMessages: messages})
}
