# Story Mode prototype

An experimental career mode for RLBot v5, inspired by v4's Story Mode. This is a place to try ideas and share reusable pieces, not a proposed final design for the official mode.

The regular RLBot navigation, Play page, application title and startup page are preserved. All new visual design lives inside Story Mode.

## Playable features

- Seven chapters, including an epilogue, with 24 playable fixtures.
- An ascending 1v1 → 2v2 → 3v3 campaign, or an entirely solo, doubles or standard campaign.
- A reference rank for each playlist, including an "I don't know" choice. Opponents start below the reference and progress above it.
- KilobyteReduced (`SubparNova/KilobyteReduced`) for opponents and, currently, every recruit. Teammates match the player's playlist reference; opponents have chapter-specific offsets and occasional boost caps.
- Arenas progress from Farmstead and local grounds to larger stadiums and Champions Field.
- BO3 chapter finals and a BO5 final, with series progress saved between games.
- An optional three-goal-lead finish, enabled by default. This applies to each game of a series, in either team's favour, and still checks any additional objectives.
- Named recruits, portraits, short biographies, recruitment costs and saved teammate selection for doubles and standard.
- A stable car body per recruit and rival, drawn from a curated Octane-hitbox catalogue. Recruitment cards show each driver's car; [STORY_CARS.md](STORY_CARS.md) lists the assignments and source.
- Club naming, a roster, match history, credits and saved campaign progression.
- Distinct completed, last-failed and unplayed fixture indicators.
- A workshop with unlocks, equipment slots and three upgrade tiers.
- A Vanilla switch that suspends equipment, Flow, the initial boost override and opponent boost caps while retaining career progress and purchases. Career rules, including the separately configured three-goal-lead option, still apply.

## Free match

Open the **Free match** tab to play a standard 1v1, 2v2 or 3v3 with a separate Bronze I–SSL selector for every bot, including teammates. Choose an arena, mix ranks freely, and launch using the Rocket League launcher selected on the regular Play page. The human keeps their own car and camera settings.

The last setup is saved locally in the GUI. Free matches do not require recruited characters, grant credits, change campaign/series progress, apply fuel handicaps, run equipment/Flow or use the career's three-goal-lead rule. They follow normal Soccar rules.

Each free-match slot has independent character and rank selectors. All nine club characters are available, including unrecruited ones; changing a character updates the portrait, name and assigned car while retaining the slot's rank. Characters cannot appear twice in one match. Format changes resolve conflicts with previously hidden slots, and the character choices are saved with the preset. These personalities are presentation, not aggression or tactical settings.

The launch button sits in a high-contrast action bar; global GUI blur styling does not override it. The README screenshot shows the real component with a sample setup and a mocked launch API, not a recorded live match.

Each Kilobyte driver gets a separate agent reservation and process, so one process cannot claim multiple differently ranked slots. This uses Byte's support for the server-assigned `RLBOT_AGENT_ID`; installed bot TOMLs are unchanged. The same reservation isolation also applies to named career drivers.

## Difficulty progression

Opponent tank limits increase across the campaign rather than restarting at 80 every chapter. These limit capacity, not boost thrust, speed or regeneration. Teammates stay at the player's playlist reference level with normal boost; the player's equipment is separate.

| Chapter | Rank offset from playlist reference | Opponent tank over the four fixtures |
| --- | --- | --- |
| The lot | −2 divisions | 70 → 75 → 80 → 80 |
| Find your duo | −1 division | 85 → 85 → 90 → 90 |
| Beyond the neighbourhood | −1 division, then reference level for the final | 95 → 95 → 100 → 100 |
| A full team | Reference level | 100 throughout |
| The bubble | +1 division | 100 throughout |
| The main stage | +2 divisions | 100 throughout |

Offsets are clamped to Bronze I–SSL and remain estimated bot difficulty labels, not measured MMR. The goal is a gentle early advantage followed by standard-fuel competition. These values are a first balancing pass, not validated win-rate targets. Vanilla disables the tank handicaps. Existing wins, credits and series scores are retained; revised settings apply when starting the next game.

Match details show the actual team names and only show a fuel handicap when it is active. Normal 100-capacity opponents are not presented as nerfed.

## Equipment and Flow: experimental

These systems are unfinished and need both gameplay testing and design discussion. Their presence is not a recommendation that the official v5 mode should use upgrades.

| Equipment | Effect, tiers I / II / III |
| --- | --- |
| Auxiliary tank | 25 / 50 / 75 reserve boost beyond the native 100; consumed first through native-tank top-ups. Large pads refill it; small pads store their unused portion (up to 12), including when the main tank is full. Also stores equipment reward overflow. |
| Defensive recovery | 15 / 20 / 25 boost after a save; 8-second cooldown |
| Shooting rhythm | 8 / 10 / 12 boost after a recorded shot; 6-second cooldown |
| Link-up play | 15 / 20 / 25 boost after a teammate receives a sufficiently separated pass; 8-second cooldown |
| Contact | 50 / 75 / 100 boost after a credited demolition; 20-second cooldown |

Save, shot and pass rewards share a rolling limit of 60 boost per minute. A purchase does not activate an item unless it is equipped. One slot opens after two fixture wins, the second after five. Higher tiers open after eight and sixteen wins.

Flow unlocks after five fixtures. It charges during active play, with some action bonuses, and allows two activations per game:

- **Surge:** injects 50 / 75 / 100 boost. Full tanks retain the charge.
- **Overdrive:** temporarily maintains native boost for 2 / 2.5 / 3 seconds, then restores the stored amount. Pauses preserve the remaining duration.
- **Focus, shorter respawns and flip-reset rewards:** visible prototypes, disabled because their effects have not been verified.

On Windows, the input fallback uses **R** or **L3**, only while Rocket League is in the foreground. These are fixed defaults, not a read-back of custom Rumble bindings. The Story page also offers an activation button when charged. Other platforms currently require that button.

## Known issues and limits

- Passive native-packet capture reproduced excessive reserve consumption: a requested one-point refill could deliver only about 0.28 points while boosting, but was charged as a full point. Transfers now debit observed delivery, accounting for held boost consumption. Whole-unit targets also prevent small reserve remainders from being truncated below the current tank and removing fuel. Regressions cover the captured 99-point plateau, reward overflow, integer targets and delayed application. **The corrected build still needs player validation in a real match.**
- Confusing or unreliable Flow activation remains under investigation. Full-tank Surge retains its charge, initial held inputs do not activate it, and charging versus activation is shown separately; these protections have synthetic regression coverage.
- The native fuel meter remains capped at 100. Extra fuel is tracked separately in a compact reserve card above the native boost gauge, at the lower right. White text sits on a tinted background with a fill bar: green above 50%, orange from 25% to 50%, red below 25%. The total is not displayed. Flow remains at the lower left. Keeping the native meter near 100 uses state-setting packets and is subject to packet timing.
- A reserve is finite, refills on large pads, accepts small-pad overflow and is emptied by demolition. This is not permanent infinite boost.
- Reserve top-ups now run during the playable kickoff as well as active play, with their own retry clock; this does not charge Flow before active play. At match start, tier I means 100 native boost plus 25 reserve. The native Rocket League gauge shows 100 and the reserve card shows 25 / 25.
- To collect a pad at 100 native boost with room in the reserve, Story briefly opens one point of native capacity while the car is at an available pad. Reserve top-ups wait for that pickup. Fuel is awarded only after the native pad switches to unavailable, preserving the game's shared pad cooldown. Large pads fill the reserve; small pads bank up to 12, subtracting fuel already received by the main tank and accounting for the artificial opening and reported held-boost consumption. Unavailable, distant and aerially overflown pads do not trigger this opening. This workaround has synthetic regression coverage and still needs in-game validation.
- Reserve top-ups begin at 99.5 native boost. Acknowledgements account for reported sustained boost use; ignored transfers time out after 120 ms and retry from 200 ms instead of waiting up to a second. These are responsiveness changes, not larger fuel rewards; native packet timing still needs in-game validation.
- Rank labels are estimates from the bot package, not measured human MMR. Unknown ranks use a provisional fallback rather than a placement assessment.
- All recruits currently have the same adjustable underlying bot at the same reference level. Names and portraits are presentation, not distinct learned personalities. Recruitment prices and unlocks are inherited placeholders and do not yet reflect meaningful differences in skill or value.
- Pass detection is a heuristic based on touch order, time and distance. It cannot reliably identify every intentional pass.
- Costs, rewards, module balance and difficulty progression need a complete balancing pass.
- There are no salaries, player training, prestige, spectator management career or completed Renown/difficulty-medal system.
- Fake quick-chat overlays have been removed. A match-manager client cannot send native quick chat on behalf of bots it does not control. No portraits replace Rocket League's native bot avatar.
- Story artwork is temporary AI-generated concept art and remains fairly large. It can be replaced independently of the gameplay code.
- Windows is the exercised platform. Linux gameplay and packaging have not been validated for this prototype.

## Try the branch

1. Install RLBot v5 and its official botpack. Confirm **KilobyteReduced** is present; arbitrary other Byte builds may not accept the same `--skill` and `--no-ui` arguments.
2. Follow the upstream [build prerequisites](README.md#installing-build-dependencies). Use the Wails CLI version matching `go.mod` (`v3.0.0-alpha.40` for this snapshot), not an unpinned newer alpha.
3. Build this branch using the upstream workflow, or the explicit Windows steps below.
4. Start the installed `RLBotServer.exe`, then the GUI built from this branch. Open **Story Mode** from the existing navigation.
5. Set reference ranks in **Club**, select a fixture and press **Play match**. Finish or stop the match before changing profile or equipment settings.

An explicit PowerShell build, with Go, a compatible C compiler, Node, pnpm and `wails3` already on PATH:

```powershell
# Bootstrap the embedded asset path for binding generation on a fresh clone.
New-Item -ItemType Directory -Force frontend/dist | Out-Null
if (-not (Test-Path frontend/dist/index.html)) {
    Set-Content frontend/dist/index.html '<!doctype html><title>Build in progress</title>'
}
wails3 generate bindings -clean=true
Push-Location frontend
pnpm install --frozen-lockfile
pnpm run build
Pop-Location
go test ./...
go build -tags production -buildvcs=false -ldflags='-H windowsgui' -o bin/rlbotgui.exe .
```

Generated bindings and frontend bundles stay ignored. No trained bot weights, user saves, credentials, local toolchains or installed executables are included.

Career saves use RLBot's local data directory (`%LOCALAPPDATA%\RLBot5\story\career-v1.json` on Windows). This branch shares that location with other builds. Back up an existing career before experimenting. The JSON save schema migrates this prototype's older version-one data; this is **not** an importer for arbitrary original v4 save files.

## Verification

The normal `go test ./...` suite uses temporary save paths and synthetic packets. It covers campaign progression, difficulty configuration, teammate names and levels, purchase validation, line-ups, series results, Vanilla mode, equipment and fuel transfer cases. It does not establish that every mechanic feels right or works reliably under real-game latency.

The two `storyprobe` test files are excluded from normal runs. They launch or restart Rocket League matches. Run them only deliberately, with an idle offline game and a configured local server:

```powershell
go test -tags storyprobe -run '^TestLiveWorkshopRoundtrip$' -v .
```

## Discussion points

- Start from standard Rocket League, using difficulty modifiers and Renown to unlock areas, then treat equipment as optional experiments?
- Keep recruitment mostly cosmetic initially, or let a named player progress through different underlying bots later?
- How should bot changes affect identity and playstyle, given that most bots expose neither difficulty nor aggression tuning?
- What would make credits and recruitment decisions meaningful before adding salaries and a larger budget simulation?
- Which equipment effects are enjoyable enough to keep, and how can their benefit remain legible without overwhelming normal Rocket League?
- What is the smallest complete campaign worth maintaining in the main project?

## Provenance

Built on the MIT-licensed [RLBot/gui](https://github.com/RLBot/gui) v5 codebase, starting from `faa9404d0463d7bcc9144a5008dff45d6f435b67`. The existing license and credits are retained.

The campaign loop and initial exploration were inspired by [RLBotGUI v4's Story Mode](https://github.com/RLBot/RLBotGUI). This snapshot has new chapter text, fixtures and artwork; it is not a faithful port of v4's specialist opponents, upgrades or save format.

Difficulty comes from SubparNova's [ByteBotpack](https://github.com/SubparN0va/ByteBotpack). The bot remains a separately installed dependency with its own author, models and licensing. This branch does not retrain it.

Concept artwork under `frontend/src/assets/story/` was generated during development with OpenAI's image-generation tool. It is supplied as replaceable prototype artwork, with no claim that it depicts real people or is official Rocket League art.
