package main

// Product IDs come from frontend/src/assets/items.csv. All bodies in this
// catalogue are listed under Octane by Epic's Rocket League Car Hitboxes page:
// https://www.epicgames.com/help/c-37599050/c-32343914/a20257614?lang=en-US
// Console-exclusive bodies and licensed real-world cars are not assigned.
type storyCar struct {
	Name string
	ID   uint32
}

var storyOctaneCars = []storyCar{
	{"Octane", 23}, {"Backfire", 21}, {"Road Hog", 25}, {"Gizmo", 26},
	{"Takumi", 402}, {"Scarab", 404}, {"Zippy", 523}, {"Grog", 607},
	{"Marauder", 1172}, {"Takumi RX-T", 1295}, {"Road Hog XL", 1300},
	{"Proteus", 1475}, {"Triton", 1478}, {"Vulcan", 1533}, {"Octane ZSR", 1568},
	{"Twinzer", 2853}, {"Fennec", 4284}, {"Mudcat", 4318}, {"Mudcat G1", 4319},
	{"Mudcat GXT", 4320}, {"Harbinger", 4906}, {"Harbinger GXT", 5039},
	{"Outlaw", 5020}, {"Outlaw GXT", 5837}, {"Dingo", 5361}, {"Jackal", 5951},
	{"Xentari", 8383}, {"Shokunin", 10900}, {"Shokunin GXT", 10901},
	{"Fennec ZR-F", 11603}, {"Corlay", 12104},
}

// Identity is attached to the fictional driver, independently from rank,
// chapter, team colour or underlying bot. These choices do not tune behavior.
var storyDriverCars = map[string]string{
	"Morgan Vale":  "Backfire",
	"Casey Reed":   "Fennec",
	"Jamie Cole":   "Scarab",
	"Rowan Hayes":  "Twinzer",
	"Lena Cross":   "Takumi",
	"Avery Stone":  "Dingo",
	"Silas Ward":   "Octane ZSR",
	"Mara Voss":    "Jackal",
	"Viktor Rook":  "Road Hog XL",
	"Logan Price":  "Octane",
	"Jules Mercer": "Takumi RX-T",
	"Finn Archer":  "Zippy",
	"Skye Bennett": "Vulcan",
	"Blake Porter": "Road Hog",
	"Maya Torres":  "Mudcat GXT",
	"Oscar Lane":   "Grog",
	"Zoe Walker":   "Fennec ZR-F",
	"Noah Carter":  "Gizmo",
	"Tess Moreno":  "Harbinger GXT",
	"Luca Hayes":   "Outlaw GXT",
	"Aria Knox":    "Proteus",
}

func storyCarForDriver(name string) (storyCar, bool) {
	model, ok := storyDriverCars[name]
	if !ok {
		return storyCar{}, false
	}
	for _, car := range storyOctaneCars {
		if car.Name == model {
			return car, true
		}
	}
	return storyCar{}, false
}

func storyDriverLoadout(original *LoadoutConfig, name string) *LoadoutConfig {
	car, ok := storyCarForDriver(name)
	if !ok {
		return original
	}
	// Copy before editing: installed bot metadata can be shared by other players.
	loadout := LoadoutConfig{
		Blue:   TeamLoadoutConfig{TeamColorId: 54, CustomColorId: 11, WheelsId: 376, BoostId: 63},
		Orange: TeamLoadoutConfig{TeamColorId: 34, CustomColorId: 31, WheelsId: 376, BoostId: 63},
	}
	if original != nil {
		loadout = *original
	}
	for _, team := range []*TeamLoadoutConfig{&loadout.Blue, &loadout.Orange} {
		team.CarId = car.ID
		// An Octane-specific decal/paint or engine must not leak onto another body.
		team.DecalId = 0
		team.Paint.CarPaintId = 0
		team.Paint.DecalPaintId = 0
		team.EngineAudioId = 0
	}
	return &loadout
}

func storyRecruitmentRoster() []StoryRecruit {
	roster := append([]StoryRecruit(nil), storyRoster...)
	for i := range roster {
		if car, ok := storyCarForDriver(roster[i].Name); ok {
			roster[i].CarName = car.Name
		}
	}
	return roster
}
