<script lang="ts">
  import { onMount } from "svelte";
  import StoryFreeMatch from "./StoryFreeMatch.svelte";
  import {
    App,
    type StoryView,
    type StoryChallenge,
    type StoryRecruit,
  } from "../../bindings/gui";
  import lot from "../assets/story/the-lot.png";
  import workshop from "../assets/story/workshop.png";
  import summit from "../assets/story/at-the-top.png";
  import cityArena from "../assets/story/beyond.png";
  import teamArena from "../assets/story/a-full-team.png";
  import bubbleArena from "../assets/story/the-bubble.png";
  import finalArena from "../assets/story/the-main-stage.png";

  import reserveArt from "../assets/story/equipment/reserve.png";
  import savesArt from "../assets/story/equipment/saves.png";
  import shotsArt from "../assets/story/equipment/shots.png";
  import passingArt from "../assets/story/equipment/passing.png";
  import demosArt from "../assets/story/equipment/demos.png";
  import respawnArt from "../assets/story/equipment/respawn.png";
  import resetArt from "../assets/story/equipment/reset.png";
  import surgeArt from "../assets/story/equipment/surge.png";
  import overdriveArt from "../assets/story/equipment/overdrive.png";
  import focusArt from "../assets/story/equipment/focus.png";

  import morganPortrait from "../assets/story/recruits/morgan-v2.png";
  import caseyPortrait from "../assets/story/recruits/casey-v2.png";
  import jamiePortrait from "../assets/story/recruits/jamie-v2.png";
  import rowanPortrait from "../assets/story/recruits/rowan-v2.png";
  import lenaPortrait from "../assets/story/recruits/lena-v2.png";
  import averyPortrait from "../assets/story/recruits/avery-v2.png";

  import silasPortrait from "../assets/story/recruits/silas.png";
  import maraPortrait from "../assets/story/recruits/mara.png";
  import viktorPortrait from "../assets/story/recruits/viktor.png";

  let {
    paths = [],
    visible = false,
  }: {
    paths?: { installPath: string; visible: boolean; isDependency: boolean }[];
    visible?: boolean;
  } = $props();
  let view = $state<StoryView | null>(null);
  let selectedCity = $state("INTRO"),
    selectedId = $state("INTRO-1"),
    tab = $state("season");
  let busy = $state(false),
    error = $state(""),
    club = $state(""),
    profileLoaded = $state(false);
  let format = $state("ascension");
  let ranks = $state<Record<string, number>>({
    "1v1": -1,
    "2v2": -1,
    "3v3": -1,
  });
  let mates = $state<string[]>([]);
  const rankNames = [
    "Auto",
    ...[
      "Bronze",
      "Silver",
      "Gold",
      "Platinum",
      "Diamond",
      "Champion",
      "Grand Champion",
    ].flatMap((t) => ["I", "II", "III"].map((d) => `${t} ${d}`)),
    "SSL",
  ];
  const city = $derived(view?.cities.find((c) => c.id === selectedCity));
  const challenge = $derived(city?.challenges.find((c) => c.id === selectedId));
  const open = $derived(view?.unlocked.includes(selectedCity) ?? false);
  const total = $derived(
    view?.cities.reduce((n, c) => n + c.challenges.length, 0) ?? 24,
  );
  const completed = $derived(view?.wins ?? 0);
  const garageBackground = $derived(
    completed >= 24
      ? summit
      : completed >= 20
        ? finalArena
        : completed >= 12
          ? teamArena
          : workshop,
  );
  const chapterIndex = $derived(
    view?.cities.findIndex((c) => c.id === selectedCity) ?? 0,
  );
  const covers: Record<string, string> = {
    INTRO: lot,
    TRYHARD: workshop,
    PBOOST: cityArena,
    WASTELAND: teamArena,
    CAMPANDSNIPE: bubbleArena,
    CHAMPIONSIAN: finalArena,
    SUMMIT: summit,
  };
  const cover = $derived(covers[selectedCity] ?? lot);
  const arenaNames: Record<string, string> = {
    Farmstead: "Farmstead",
    Farmstead_Night: "Farmstead (Night)",
    BeckwithPark: "Beckwith Park",
    BeckwithPark_Midnight: "Beckwith Park (Midnight)",
    UrbanCentral: "Urban Central",
    UrbanCentral_Dawn: "Urban Central (Dawn)",
    UrbanCentral_Night: "Urban Central (Night)",
    Mannfield: "Mannfield",
    Mannfield_Night: "Mannfield (Night)",
    DFHStadium: "DFH Stadium",
    DFHStadium_Day: "DFH Stadium (Day)",
    ChampionsField: "Champions Field",
    ChampionsField_Day: "Champions Field (Day)",
  };
  const portraits: Record<string, string> = {
    morgan: morganPortrait,
    casey: caseyPortrait,
    jamie: jamiePortrait,
    rowan: rowanPortrait,
    lena: lenaPortrait,
    avery: averyPortrait,
    silas: silasPortrait,
    mara: maraPortrait,
    viktor: viktorPortrait,
  };
  const label = (id: string) =>
    view?.roster.find((r) => r.id === id)?.name ?? id;
  const portrait = (id: string) =>
    portraits[view?.roster.find((r) => r.id === id)?.portrait ?? ""];
  function botPaths() {
    return paths
      .filter((p) => p.visible && !p.isDependency)
      .map((p) => p.installPath);
  }
  function selectedLineup() {
    const ids: string[] = [];
    for (let i = 0; i < (challenge?.humanTeamSize ?? 1) - 1; i++) {
      const preferred = mates[i] ?? view?.save.lineup?.[i];
      ids.push(
        preferred &&
          view?.save.recruits.includes(preferred) &&
          !ids.includes(preferred)
          ? preferred
          : (view?.save.recruits.find((id) => !ids.includes(id)) ?? ""),
      );
    }
    return ids;
  }
  function chooseMate(index: number, id: string) {
    const ids = selectedLineup();
    ids[index] = id;
    mates = ids;
    act(() => App.SetStoryLineup(ids.filter(Boolean)));
  }
  function recruitLevel(recruit: StoryRecruit) {
    if (!recruit.adaptive) return "Fixed difficulty · not rank calibrated";
    const size =
      challenge?.humanTeamSize && challenge.humanTeamSize > 1
        ? challenge.humanTeamSize
        : 2;
    let rank = view?.save.ranks[`${size}v${size}`] ?? -1;
    if (rank < 1) rank = view?.save.kilobyteSkill ?? 7;
    return rank === 0
      ? "Adaptive · automatic level"
      : `Estimated ${rankNames[Math.max(1, Math.min(22, rank))]} · adaptive`;
  }

  function bestOf(c: StoryChallenge) {
    return c.bestOf === 3 || c.bestOf === 5 ? c.bestOf : 1;
  }
  function seriesScore(c: StoryChallenge) {
    const score = view?.save.series?.[c.id];
    const target = Math.floor(bestOf(c) / 2) + 1;
    return score && score.wins < target && score.losses < target
      ? { wins: score.wins, losses: score.losses }
      : { wins: 0, losses: 0 };
  }
  function seriesOpen(c: StoryChallenge) {
    const score = seriesScore(c);
    return bestOf(c) > 1 && score.wins + score.losses > 0;
  }
  function fixtureState(c: StoryChallenge) {
    if (view?.save.completed[c.id]) return "completed";
    if (view?.status.active && view.status.challengeId === c.id)
      return "playing";
    if (bestOf(c) > 1) {
      const score = view?.save.series?.[c.id];
      if (score && score.losses >= Math.floor(bestOf(c) / 2) + 1)
        return "failed";
      return seriesOpen(c) ? "in-progress" : "unplayed";
    }
    const latest = [...(view?.save.attempts ?? [])]
      .reverse()
      .find((attempt) => attempt.challengeId === c.id);
    return latest && !latest.completed ? "failed" : "unplayed";
  }
  const fixtureStatusLabels: Record<string, string> = {
    completed: "Completed",
    failed: "Previous attempt failed",
    playing: "Match in progress",
    "in-progress": "Series in progress",
    unplayed: "Not completed yet",
  };
  function seriesLabel(c: StoryChallenge) {
    const score = seriesScore(c);
    return bestOf(c) > 1
      ? `BO${bestOf(c)}${seriesOpen(c) ? ` · ${score.wins}–${score.losses}` : ""}`
      : "Single game";
  }
  function playLabel(c: StoryChallenge) {
    const score = seriesScore(c);
    return bestOf(c) === 1
      ? "Play match"
      : seriesOpen(c)
        ? `Play game ${score.wins + score.losses + 1}`
        : `Start BO${bestOf(c)}`;
  }
  function objectives(c: StoryChallenge) {
    const r = c.completionConditions;
    return [
      bestOf(c) > 1
        ? "Win " + (Math.floor(bestOf(c) / 2) + 1) + " games"
        : "Win the match",
      r.scoreDifference ? `Lead by ${r.scoreDifference}+ goals` : "",
      r.selfDemoCount != null ? "Avoid being demolished" : "",
      r.demoAchievedCount ? `${r.demoAchievedCount}+ opponent demolitions` : "",
      r.goalsScored ? `Score ${r.goalsScored}+ yourself` : "",
    ].filter(Boolean);
  }
  async function refresh() {
    try {
      view = await App.GetStory();
      if (!club) club = view.save.club;
      if (!profileLoaded) {
        format = view.save.format;
        ranks = { ...view.save.ranks };
        const pending = view.cities
          .flatMap((area) => area.challenges.map((f) => ({ area, fixture: f })))
          .find(({ fixture }) => seriesOpen(fixture));
        const next = view.cities.find(
          (c) =>
            view?.unlocked.includes(c.id) &&
            c.challenges.some((f) => !view?.save.completed[f.id]),
        );
        if (pending) {
          selectedCity = pending.area.id;
          selectedId = pending.fixture.id;
        } else if (next) {
          selectedCity = next.id;
          selectedId =
            next.challenges.find((f) => !view?.save.completed[f.id])?.id ??
            next.challenges[0].id;
        } else if (view.unlocked.includes("SUMMIT")) {
          selectedCity = "SUMMIT";
          selectedId = "";
        }
        profileLoaded = true;
      }
    } catch (e) {
      error = String(e);
    }
  }
  onMount(() => {
    refresh();
    const t = setInterval(() => {
      if (visible) refresh();
    }, 1000);
    return () => clearInterval(t);
  });
  async function act(fn: () => Promise<unknown>) {
    busy = true;
    error = "";
    try {
      await fn();
      await refresh();
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }
  function selectChapter(id: string) {
    selectedCity = id;
    const rows = view?.cities.find((c) => c.id === id)?.challenges;
    selectedId =
      rows?.find((c) => !view?.save.completed[c.id])?.id ?? rows?.[0]?.id ?? "";
    mates = [];
  }
  function start() {
    if (!challenge || !view) return;
    const selected = selectedLineup();
    if (selected.some((id) => !id)) {
      tab = "recruitment";
      error = "Recruit another driver to complete your line-up.";
      return;
    }
    act(() =>
      App.StartStory(
        challenge!.id,
        selected,
        botPaths(),
        localStorage.getItem("MS_LAUNCHER") ?? "",
        localStorage.getItem("MS_LAUNCHER_ARG") ?? "",
      ),
    );
  }

  function modulePrice(id: string, level: number) {
    if (
      level === 0 &&
      !view?.save.starterClaimed &&
      ["reserve", "saves", "shots"].includes(id)
    )
      return 0;
    return [120, 150, 250][level] ?? 0;
  }

  const equipmentArt: Record<string, string> = {
    reserve: reserveArt,
    saves: savesArt,
    shots: shotsArt,
    passing: passingArt,
    demos: demosArt,
    respawn: respawnArt,
    reset: resetArt,
    surge: surgeArt,
    overdrive: overdriveArt,
    focus: focusArt,
  };
  const clubNames = [
    "Backstreet United",
    "Late Kickoff",
    "Second Touch FC",
    "Corner Club",
    "Crossbar Social",
    "Sunday Drivers",
    "Loose Ball Athletic",
    "Last Minute FC",
    "Overtime Union",
    "The Rusty Bumpers",
  ];
  function suggestClub() {
    const choices = clubNames.filter((name) => name !== club);
    club = choices[Math.floor(Math.random() * choices.length)];
  }
  function moduleEffect(id: string, values: number[]) {
    const level = Math.max(1, view?.save.modules[id] ?? 0);
    return values[level - 1] ?? values[0];
  }
  function flowEffect(id: string) {
    const level = Math.max(1, view?.save.flowLevels[id] ?? 1);
    return id === "surge"
      ? `+${[50, 75, 100][level - 1]} boost`
      : id === "overdrive"
        ? `${[2, 2.5, 3][level - 1]} seconds`
        : "Not available";
  }
  const locked = $derived(busy || !!view?.status.active);
</script>

<section class="career" aria-label="Career">
  <header class="masthead">
    <h1 class="wordmark">RLStory</h1>
    <nav aria-label="Career sections">
      {#each [["season", "Career"], ["free", "Free match"], ["workshop", "Workshop"], ["recruitment", "Recruitment"], ["club", "Club"]] as [id, title]}<button
          class:active={tab === id}
          aria-current={tab === id ? "page" : undefined}
          onclick={() => (tab = id)}>{title}</button
        >{/each}
    </nav>
    <div class="totals">
      <span class="club-title">{view?.save.club ?? "Your club"}</span><span
        >{view?.save.credits ?? 0} <small>Credits</small></span
      >
    </div>
  </header>
  {#if error}<div class="notice error" role="alert">{error}</div>{/if}
  {#if view?.status.message}<div class="notice" role="status">
      <span>{view.status.message}</span
      >{#if view.status.active}{#if !view.save.vanilla && !view.status.freeMatch}<div
            class="live-flow"
          >
            <progress max="100" value={view.status.flowCharge}></progress><span
              >Flow {Math.round(view.status.flowCharge)}% · {2 -
                view.status.flowUses} uses left · press R / L3 when ready</span
            ><button
              disabled={busy ||
                view.status.flowCharge < 100 ||
                view.status.flowUses >= 2}
              onclick={() => act(() => App.TriggerStoryFlow())}
              >Activate Flow</button
            >
          </div>{/if}<button
          disabled={busy}
          onclick={() =>
            act(async () => {
              const r = await App.StopMatch(false);
              if (!r.success) throw new Error(r.message);
            })}>Stop match</button
        >{/if}
    </div>{/if}
  {#if !view}<p class="loading">Loading career…</p>
  {:else if tab === "free"}
    <StoryFreeMatch
      paths={botPaths()}
      ranks={view.save.ranks}
      {rankNames}
      {locked}
      onlaunch={refresh}
      roster={view.roster}
      {portraits}
    />
  {:else if tab === "season"}
    {#if !view.save.profileSet}<div class="setup-hint">
        <span>Choose your playlist ranks to set the starting difficulty.</span
        ><button onclick={() => (tab = "club")}>Set your level →</button>
      </div>{/if}
    <div class="season">
      <aside aria-label="Chapters">
        <h2 class="season-title">Season One</h2>
        {#each view.cities as area, i}<button
            aria-current={area.id === selectedCity ? "step" : undefined}
            class:selected={area.id === selectedCity}
            onclick={() => selectChapter(area.id)}
            ><span class="chapter-number">{String(i + 1).padStart(2, "0")}</span
            ><span
              ><strong>{area.name}</strong><small
                >{area.challenges.length
                  ? `${area.challenges.filter((c) => view?.save.completed[c.id]).length} / ${area.challenges.length}`
                  : "Epilogue"}</small
              ></span
            >{#if !view.unlocked.includes(area.id)}<span
                class="access"
                aria-label="Locked"
                ><svg
                  width="13"
                  height="15"
                  viewBox="0 0 16 20"
                  aria-hidden="true"
                  ><path
                    d="M4 8V5a4 4 0 0 1 8 0v3M3 8h10a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1Z"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                  /></svg
                ></span
              >{/if}</button
          >{/each}
      </aside>
      <div class="chapter">
        <div
          class="chapter-cover"
          style={`background-image:linear-gradient(0deg,#0b1d36dc,transparent 90%),url(${cover})`}
        >
          <p class="eyebrow">
            {selectedCity === "SUMMIT"
              ? "EPILOGUE"
              : `CHAPTER ${String(chapterIndex + 1).padStart(2, "0")}`}
          </p>
          <h2>{city?.name}</h2>
        </div>
        <p class="synopsis">{city?.description}</p>
        {#if !open}<p class="lock">
            Complete {city?.prereqs
              .map((id) => view?.cities.find((c) => c.id === id)?.name)
              .join(" and ")} to continue.
          </p>{/if}
        {#if selectedCity === "SUMMIT"}<div class="epilogue">
            <h3>
              {open
                ? "You brought it home."
                : "The last page is still unwritten."}
            </h3>
            <p>
              {open
                ? "The fixtures remain open for return matches. Change your equipment, try another format, or revisit an old rival."
                : "Win the main-stage final to reach the epilogue."}
            </p>
          </div>{/if}
        {#each city?.challenges ?? [] as c, i}{@const state =
            fixtureState(c)}<button
            class="fixture"
            aria-current={selectedId === c.id ? "true" : undefined}
            class:selected={selectedId === c.id}
            onclick={() => {
              selectedId = c.id;
              mates = [];
            }}
            ><span class="marker"
              ><span
                class="fixture-status"
                class:done={state === "completed"}
                class:failed={state === "failed"}
                role="img"
                aria-label={fixtureStatusLabels[state]}
                >{state === "completed"
                  ? "✓"
                  : state === "failed"
                    ? "✕"
                    : String(i + 1).padStart(2, "0")}</span
              ></span
            ><span
              ><strong>{c.display}</strong><small
                >{c.humanTeamSize}v{c.opponentBots.length} · {c.opponentNames.join(
                  " / ",
                )} · {seriesLabel(c)}</small
              ></span
            ></button
          >{/each}
      </div>
      {#if challenge}
        <article class="brief">
          <p class="brief-label">
            {view.save.completed[challenge.id] ? "Return match" : "Next match"}
          </p>
          <h2>{challenge.display}</h2>
          <ul class="objectives">
            {#each objectives(challenge) as objective}<li>
                {objective}
              </li>{/each}
          </ul>
          {#if bestOf(challenge) > 1}<div class="series-panel">
              <span
                >Best of {bestOf(challenge)} · First to {Math.floor(
                  bestOf(challenge) / 2,
                ) + 1} wins</span
              ><strong
                >You {seriesScore(challenge).wins} – {seriesScore(challenge)
                  .losses} Opponent</strong
              ><small
                >Progress saves after each game. Rewards are awarded when the
                series ends.</small
              >{#if seriesOpen(challenge)}<button
                  class="series-reset"
                  disabled={locked}
                  onclick={() => act(() => App.ResetStorySeries(challenge!.id))}
                  >Restart series at 0–0</button
                >{/if}
            </div>{/if}
          <div class="match-facts">
            <span
              >{challenge.humanTeamSize}v{challenge.opponentBots.length}</span
            ><span>{rankNames[challenge.enemyRank]}</span><span
              >{view.save.completed[challenge.id] ? 25 : 100} credits</span
            >
          </div>
          {#each selectedLineup() as mate, i}<div class="lineup-driver">
              {#if mate && portrait(mate)}<img
                  src={portrait(mate)}
                  alt=""
                />{/if}<label class="teammate"
                >Teammate {i + 1}<select
                  value={mate}
                  disabled={locked}
                  onchange={(e) => chooseMate(i, e.currentTarget.value)}
                  >{#if !mate}<option value="">Recruit a driver first</option
                    >{/if}{#each view.save.recruits as id}<option
                      value={id}
                      disabled={selectedLineup().some(
                        (other, index) => index !== i && other === id,
                      )}>{label(id)}</option
                    >{/each}</select
                ></label
              >
            </div>{/each}
          {#if challenge.humanTeamSize > 1}<button
              class="recruit-link"
              onclick={() => (tab = "recruitment")}>Manage recruitment →</button
            >{/if}
          <label class="check"
            ><input
              type="checkbox"
              checked={view.save.threeGoalLead ?? true}
              disabled={locked}
              onchange={(e) =>
                act(() => App.SetStoryThreeGoalLead(e.currentTarget.checked))}
            />Stop each game at a three-goal lead</label
          >
          <button
            class="primary play"
            disabled={!open || locked}
            onclick={start}
            >{busy
              ? "Preparing…"
              : view.status.active
                ? "Match in progress"
                : playLabel(challenge)}</button
          >
          <details class="match-details">
            <summary>Match details</summary>
            <p>{challenge.brief}</p>
            <dl class="fixture-details">
              <dt>Arena</dt>
              <dd>{arenaNames[challenge.map] ?? challenge.map}</dd>
              <dt>Format</dt>
              <dd>
                {challenge.humanTeamSize}v{challenge.opponentBots.length} · {bestOf(
                  challenge,
                ) > 1
                  ? `BO${bestOf(challenge)}`
                  : "Single game"}
              </dd>
              <dt>Opponents</dt>
              <dd>{challenge.opponentNames.join(", ")}</dd>
              <dt>Your team</dt>
              <dd>
                {["You", ...selectedLineup().filter(Boolean).map(label)].join(
                  ", ",
                )}
              </dd>
              {#if view.save.vanilla}
                <dt>Rules</dt>
                <dd>Vanilla · equipment off</dd>
              {:else if challenge.opponentBoostCap > 0 && challenge.opponentBoostCap < 100}
                <dt>Handicap</dt>
                <dd>
                  Opponent boost capacity −{100 - challenge.opponentBoostCap}% ({challenge.opponentBoostCap}
                  max)
                </dd>
              {/if}
            </dl>
          </details>
        </article>
      {/if}
    </div>
  {:else if tab === "recruitment"}
    <section class="recruitment" aria-label="Recruitment">
      <div class="recruitment-heading">
        <div>
          <p class="eyebrow">THE CLUB</p>
          <h2>Find your next teammate.</h2>
          <p>
            Choose your line-up before a 2v2 or 3v3 fixture. All teammates match
            your playlist reference level for now.
          </p>
        </div>
        <span>{view.save.recruits.length} in your roster</span>
      </div>
      <div class="recruitment-grid">
        {#each view.roster as recruit}
          <article
            class="recruit-card"
            class:recruited={view.save.recruits.includes(recruit.id)}
          >
            <img
              class="driver-portrait"
              src={portraits[recruit.portrait]}
              alt={`Portrait of ${recruit.name}`}
              loading="lazy"
            />
            <div class="recruit-content">
              <p class="eyebrow">
                {view.save.recruits.includes(recruit.id)
                  ? "YOUR ROSTER"
                  : view.wins < recruit.unlock
                    ? "SCOUTING"
                    : "AVAILABLE"}
              </p>
              <h3>{recruit.name}</h3>
              <p>{recruit.bio}</p>
              <p class="recruit-level">{recruitLevel(recruit)}</p>
              <p class="recruit-level">{recruit.carName}</p>
              <details>
                <summary>Bot details</summary>
                <p>
                  {recruit.botName}{recruit.adaptive
                    ? " · matches your playlist reference level"
                    : " · original fixed difficulty"}.
                </p>
              </details>
              <div class="recruit-actions">
                {#if view.save.recruits.includes(recruit.id)}<span
                    >In your roster ✓</span
                  >{:else}<button
                    disabled={locked ||
                      view.wins < recruit.unlock ||
                      view.save.credits < recruit.cost}
                    onclick={() =>
                      act(() => App.RecruitStory(recruit.id, botPaths()))}
                    >Recruit · {recruit.cost} credits</button
                  >{#if view.wins < recruit.unlock}<small
                      >Unlocks after {recruit.unlock} fixtures won</small
                    >{/if}{/if}
              </div>
            </div>
          </article>
        {/each}
      </div>
    </section>
  {:else if tab === "workshop"}
    <div
      class="workshop-banner"
      style={`background-image:linear-gradient(90deg,#0b1d36e8,#0b1d3620),url(${garageBackground})`}
    >
      <p class="eyebrow">THE WORKSHOP</p>
      <h2>Build for the way you play.</h2>
      <p>
        Two passive modules. One active ability.<br />Normal boost is available
        from the first fixture.
      </p>
    </div>
    <div class="vanilla-controls">
      <label class="vanilla-switch"
        ><span
          ><strong>Vanilla Rocket League</strong><small
            >Standard boost rules. No equipment effects, Flow State or opponent
            boost handicaps.</small
          ></span
        ><input
          type="checkbox"
          role="switch"
          checked={view.save.vanilla}
          disabled={locked}
          onchange={(e) =>
            act(() => App.SetStoryVanilla(e.currentTarget.checked))}
        /></label
      >{#if view.save.vanilla}<p class="vanilla-note" role="status">
          Vanilla mode is on. Your purchases and equipped loadout are saved for
          when you switch back. Career progression and the three-goal-lead
          option remain active.
        </p>{/if}
    </div>
    <div class="workshop-body">
      <section class="flow" aria-label="Flow State">
        <div>
          <p class="eyebrow">FLOW STATE</p>
          <h2>One charge. Your call.</h2>
          <p>
            Unlocks after five wins. Press R on keyboard or the left stick on a
            controller when the gauge is full (default Rumble buttons). A manual
            activation button is also available during a match.
          </p>
        </div>
        <div class="flow-modes">
          {#each [{ id: "surge", name: "Surge", description: "An immediate boost injection." }, { id: "overdrive", name: "Overdrive", description: "Experimental: normal boost propulsion without consuming stored fuel." }, { id: "focus", name: "Focus", description: "A short global slowdown. Still in technical testing." }] as mode}
            <article
              class="illustrated-card"
              class:equipped={view.save.flowMode === mode.id}
            >
              <img
                class="card-art"
                src={equipmentArt[mode.id]}
                alt=""
                loading="lazy"
              />
              <h3>{mode.name}</h3>
              <p>{mode.description}</p>
              <strong class="effect">{flowEffect(mode.id)}</strong>
              <div class="module-actions">
                {#if mode.id !== "focus"}
                  <button
                    disabled={locked ||
                      view.wins < 5 ||
                      view.save.flowMode === mode.id}
                    onclick={() => act(() => App.SetStoryFlow(mode.id, false))}
                    >{view.save.flowMode === mode.id
                      ? "Selected"
                      : "Select"}</button
                  >
                  {#if (view.save.flowLevels[mode.id] ?? 1) < 3}<button
                      disabled={locked ||
                        view.wins < 5 ||
                        (view.save.flowLevels[mode.id] ?? 1) >= view.maxTier ||
                        view.save.credits <
                          ((view.save.flowLevels[mode.id] ?? 1) === 2
                            ? 250
                            : 150)}
                      onclick={() => act(() => App.SetStoryFlow(mode.id, true))}
                      >Upgrade · {(view.save.flowLevels[mode.id] ?? 1) === 2
                        ? 250
                        : 150} credits</button
                    >{:else}<span>Fully tuned</span>{/if}
                {/if}
              </div>
            </article>
          {/each}
        </div>
      </section>
      <div class="equipment-summary">
        <div>
          <p class="eyebrow">EQUIPMENT</p>
          <strong
            >{view.slots} equipment slot{view.slots === 1 ? "" : "s"} available</strong
          >
        </div>
        <span
          >{view.save.equipped.length}/{view.slots} equipped · {view.wins < 2
            ? "First module unlocks after two wins."
            : view.wins < 5
              ? "Second slot unlocks after five wins."
              : "Change equipment freely between matches."}</span
        >
      </div>
      <div class="modules">
        {#each view.modules as module}
          <article
            class="module illustrated-card"
            class:unavailable={!module.available}
          >
            <img
              class="card-art"
              src={equipmentArt[module.id]}
              alt=""
              loading="lazy"
            />
            <div class="module-top">
              <p class="eyebrow">
                {view.save.equipped.includes(module.id)
                  ? "EQUIPPED"
                  : !module.available
                    ? "IN DEVELOPMENT"
                    : view.save.modules[module.id]
                      ? "OWNED"
                      : "AVAILABLE TO UNLOCK"}
              </p>
              <span
                >{module.id === "reserve"
                  ? "FUEL"
                  : module.id === "demos"
                    ? "CONTACT"
                    : "RECOVERY"}</span
              >
            </div>
            <h3>{module.name}</h3>
            <p>{module.description}</p>
            <strong class="effect"
              >+{moduleEffect(module.id, module.values)}
              {module.id === "reserve" ? "reserve boost" : "boost"}</strong
            >
            <p class="small">
              {module.id === "reserve"
                ? `Main tank: 100. Total fuel capacity: ${100 + moduleEffect(module.id, module.values)}.`
                : module.note}
            </p>
            <div class="module-actions">
              {#if module.available}
                {#if (view.save.modules[module.id] ?? 0) < 3}<button
                    disabled={locked ||
                      view.wins < module.unlock ||
                      (view.save.modules[module.id] ?? 0) >= view.maxTier ||
                      view.save.credits <
                        modulePrice(
                          module.id,
                          view.save.modules[module.id] ?? 0,
                        )}
                    onclick={() => act(() => App.BuyStoryModule(module.id))}
                    >{modulePrice(
                      module.id,
                      view.save.modules[module.id] ?? 0,
                    ) === 0
                      ? "Choose free module"
                      : `${view.save.modules[module.id] ? "Upgrade" : "Buy"} · ${modulePrice(module.id, view.save.modules[module.id] ?? 0)} credits`}</button
                  >{:else}<span>Fully tuned</span>{/if}
                {#if view.save.modules[module.id]}<button
                    disabled={locked ||
                      (!view.save.equipped.includes(module.id) &&
                        view.save.equipped.length >= view.slots)}
                    onclick={() => act(() => App.EquipStoryModule(module.id))}
                    >{view.save.equipped.includes(module.id)
                      ? "Unequip"
                      : "Equip"}</button
                  >{/if}
                {#if view.wins < module.unlock}<small
                    >Available after {module.unlock} wins</small
                  >{/if}
              {:else}<span>Not available in this build</span>{/if}
            </div>
          </article>
        {/each}
      </div>
    </div>
  {:else}
    <div class="settings">
      <article>
        <p class="eyebrow">CLUB</p>
        <h2>Your name on the sheet.</h2>
        <label>Club name<input maxlength="40" bind:value={club} /></label>
        <div class="club-name-actions">
          <button disabled={locked} onclick={suggestClub}
            >Suggest a name ↻</button
          ><button
            class="primary"
            disabled={locked || !club.trim()}
            onclick={() => act(() => App.RenameStoryClub(club))}
            >Save name</button
          >
        </div>
        <h3>Roster</h3>
        <div class="club-roster">
          {#each view.save.recruits as id}<div>
              {#if portrait(id)}<img
                  src={portrait(id)}
                  alt=""
                  loading="lazy"
                />{/if}<span>{label(id)}</span>
            </div>{/each}
        </div>
        <button class="recruit-link" onclick={() => (tab = "recruitment")}
          >Open recruitment →</button
        >
      </article>
      <article>
        <p class="eyebrow">CAREER FORMAT & LEVEL</p>
        <h2>Start where you belong.</h2>
        <label
          >Format<select bind:value={format}
            ><option value="ascension">Ascension · 1v1 → 2v2 → 3v3</option
            ><option value="solo">Solo · 1v1 throughout</option><option
              value="duo">Duo · 2v2 throughout</option
            ><option value="trio">Trio · 3v3 throughout</option></select
          ></label
        >{#each ["1v1", "2v2", "3v3"] as playlist}<label
            >{playlist} reference rank<select bind:value={ranks[playlist]}
              ><option value={-1}>I don't know</option
              >{#each rankNames.slice(1) as name, i}<option value={i + 1}
                  >{name}</option
                >{/each}</select
            ></label
          >{/each}
        <p class="small">
          Early opponents start below your reference level; later chapters move
          above it. Kilobyte ranks are estimates. Unknown currently uses {rankNames[
            view.save.kilobyteSkill ?? 7
          ]} as a provisional reference, not a measured placement.
        </p>
        <button
          class="primary"
          disabled={locked}
          onclick={() => act(() => App.SetStoryProfile(format, ranks))}
          >Save career settings</button
        >
      </article>
      <article class="history">
        <p class="eyebrow">RECENT FIXTURES</p>
        {#each [...view.save.attempts].reverse().slice(0, 8) as attempt}<p>
            <strong>{attempt.challengeId}</strong> · {attempt.blue}–{attempt.orange}
            · {attempt.seriesBestOf
              ? `BO${attempt.seriesBestOf} ${attempt.seriesWins}–${attempt.seriesLosses} · ${attempt.won ? "Game won" : "Game lost"}`
              : attempt.completed
                ? "Win"
                : "Attempt finished"}
          </p>{/each}
      </article>
    </div>
  {/if}
  <footer>
    <span>Offline career · Progress saved automatically</span><span
      >{completed} / {total} fixtures won</span
    >
  </footer>
</section>

<style>
  .career {
    --navy: #0b1c33;
    --surface: #112944;
    --line: #ffffff20;
    --text: #f2f6ff;
    --muted: #aebed3;
    --blue: #0879df;
    --orange: #f58a13;
    display: flex;
    flex-direction: column;
    min-height: 100%;
    width: 100%;
    background: var(--navy);
    color: var(--text);
    font-family: "Bahnschrift", "Segoe UI", sans-serif;
    font-size: 14px;
  }
  .career * {
    box-sizing: border-box;
  }
  h1,
  h2,
  h3,
  p {
    margin: 0;
  }
  h2 {
    font-size: 28px;
    font-weight: 500;
    line-height: 1.2;
  }
  h3 {
    font-size: 20px;
    font-weight: 500;
  }
  p {
    font-size: 14px;
    line-height: 1.6;
    color: var(--muted);
  }
  small,
  .small {
    font-size: 12px;
    line-height: 1.6;
    color: var(--muted);
  }
  .eyebrow {
    font-size: 11px;
    letter-spacing: 0.1em;
    color: var(--muted);
    margin-bottom: 10px;
    font-weight: 500;
  }
  .masthead {
    display: flex;
    align-items: center;
    gap: 42px;
    padding: 24px 32px;
    flex-wrap: wrap;
  }
  .wordmark {
    font-size: 27px;
    font-style: italic;
    font-weight: 600;
    letter-spacing: 0.01em;
    padding-right: 30px;
    border-right: 1px solid var(--line);
  }
  .masthead nav {
    display: flex;
    gap: 12px;
  }
  .masthead nav button {
    border: 0;
    background: transparent;
    padding: 9px 19px;
    font-size: 18px;
    color: var(--text);
  }
  .masthead nav button.active {
    background: var(--blue);
  }
  .totals {
    margin-left: auto;
    display: flex;
    gap: 25px;
    align-items: center;
    font-size: 16px;
  }
  .totals > span + span {
    padding-left: 25px;
    border-left: 1px solid var(--line);
  }
  .totals small {
    font-size: 15px;
    color: var(--text);
  }
  .club-title {
    max-width: 230px;
    overflow-wrap: anywhere;
  }
  button {
    cursor: pointer;
    box-shadow: none;
    backdrop-filter: none !important;
    -webkit-backdrop-filter: none !important;
    isolation: auto;
    border: 1px solid #ffffff30;
    background: var(--surface);
    color: var(--text);
    border-radius: 5px;
    font-family: inherit;
    font-size: 13px;
    font-weight: 400;
    padding: 10px 14px;
    line-height: 1.25;
    transition: background-color 0.12s ease;
  }
  button:not(:disabled):hover {
    background: #1b3b5e;
  }
  button:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
  button:focus-visible,
  input:focus-visible,
  select:focus-visible,
  summary:focus-visible {
    outline: 2px solid #70bdff;
    outline-offset: 3px;
  }
  .career button.primary {
    background: var(--orange);
    color: #111d2e;
    border: 0;
    font-weight: 550;
  }
  .career button.primary:not(:disabled):hover {
    background: #ffa02e;
  }
  label {
    display: block;
    font-size: 13px;
    color: var(--text);
  }
  input,
  select {
    font: inherit;
    color: var(--text);
    background: #08182c;
    border: 1px solid #ffffff30;
    border-radius: 4px;
    padding: 11px;
    width: 100%;
    display: block;
    margin: 8px 0 18px;
    color-scheme: dark;
  }
  input:disabled,
  select:disabled {
    opacity: 0.5;
  }
  .notice {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 14px;
    margin: 10px 28px;
    padding: 14px 18px;
    background: #153456;
    font-size: 13px;
  }
  .notice > button:last-child {
    margin-left: auto;
  }
  .error {
    background: #532c37;
    overflow-wrap: anywhere;
  }
  .live-flow {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
  }
  .live-flow progress {
    width: 100px;
    height: 8px;
    appearance: none;
    border: 0;
    background: #07182b;
    border-radius: 4px;
    overflow: hidden;
  }
  .live-flow progress::-webkit-progress-bar {
    background: #07182b;
  }
  .live-flow progress::-webkit-progress-value {
    background: var(--blue);
  }
  .setup-hint {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 15px;
    margin: 0 28px 6px;
    padding: 10px 0;
    color: var(--muted);
    font-size: 12px;
  }
  .setup-hint button {
    padding: 7px 10px;
    background: transparent;
    border: 0;
    color: #8dcaff;
  }
  .season {
    display: grid;
    grid-template-columns: 210px minmax(280px, 1fr) minmax(240px, 300px);
    gap: 28px;
    padding: 28px;
    flex: 1;
    align-items: start;
  }
  .season aside {
    min-width: 0;
  }
  .season-title {
    font-size: 24px;
    font-weight: 500;
    margin: 3px 0 24px;
    padding: 0 8px 18px;
    border-bottom: 1px solid #ffffff40;
  }
  .season aside button {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 10px;
    min-height: 70px;
    border: 0;
    border-bottom: 1px solid var(--line);
    border-radius: 0;
    background: transparent;
    text-align: left;
  }
  .season aside button.selected {
    background: var(--blue);
    border-bottom-color: transparent;
    border-radius: 5px;
  }
  .season aside button:not(.selected):hover {
    background: #ffffff09;
  }
  .chapter-number {
    font-size: 13px;
    color: var(--muted);
    min-width: 20px;
  }
  .season aside button.selected .chapter-number {
    color: #e0f0ff;
  }
  .season aside strong {
    font-size: 15px;
    font-weight: 400;
    display: block;
    line-height: 1.3;
  }
  .season aside small {
    display: block;
    font-size: 11px;
    margin-top: 4px;
    color: #9aabc2;
  }
  .season aside .selected small {
    color: #d7eaff;
  }
  .access {
    margin-left: auto;
    display: flex;
    color: #93a6bf;
    padding-left: 4px;
  }
  .chapter {
    min-width: 0;
  }
  .chapter-cover {
    height: 205px;
    background-size: cover;
    background-position: center;
    display: flex;
    flex-direction: column;
    justify-content: end;
    padding: 22px;
    border: 0;
    border-radius: 5px;
    overflow: hidden;
  }
  .chapter-cover h2 {
    font-size: 34px;
    color: white;
  }
  .chapter-cover .eyebrow {
    color: #e5edfb;
    margin-bottom: 6px;
    letter-spacing: 0.04em;
    font-size: 12px;
  }
  .synopsis {
    padding: 18px 2px 20px;
    color: #d3deec;
    font-size: 13px;
  }
  .fixture {
    display: flex;
    gap: 15px;
    align-items: center;
    width: 100%;
    text-align: left;
    min-height: 78px;
    padding: 16px 18px;
    margin: 0;
    background: transparent;
    border: 0;
    border-bottom: 1px solid var(--line);
    border-radius: 0;
  }
  .fixture.selected {
    background: var(--blue);
    border-bottom-color: transparent;
    border-radius: 5px;
  }
  .fixture:not(.selected):hover {
    background: #ffffff08;
  }
  .fixture strong {
    display: block;
    font-size: 18px;
    font-weight: 400;
    line-height: 1.25;
  }
  .fixture small {
    display: block;
    margin-top: 6px;
    font-size: 12px;
  }
  .fixture.selected small {
    color: #e0ecff;
  }
  .marker {
    min-width: 30px;
    font-size: 18px;
    color: #e1eafa;
    border-right: 1px solid #ffffff30;
    padding-right: 14px;
  }
  .fixture.selected .marker {
    color: white;
  }
  .brief {
    background: none;
    border: 0;
    padding: 0 0 0 3px;
    min-width: 0;
  }
  .brief-label {
    font-size: 22px;
    color: var(--text);
    padding: 3px 0 18px;
    border-bottom: 1px solid #ffffff40;
    margin-bottom: 27px;
  }
  .brief h2 {
    font-size: 27px;
    margin-bottom: 16px;
  }
  .objectives {
    list-style: none;
    padding: 0;
    margin: 0 0 24px;
    color: #d9e3f1;
    font-size: 14px;
    line-height: 1.6;
  }
  .objectives li + li {
    margin-top: 6px;
  }
  .match-facts {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    padding: 23px 0;
    margin-bottom: 24px;
    border-top: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
    font-size: 15px;
  }
  .match-facts span + span::before {
    content: "·";
    color: var(--muted);
    margin-right: 8px;
  }
  .brief .check {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 13px;
    line-height: 1.5;
    margin: 10px 0 24px;
  }
  .check input {
    appearance: auto;
    display: block;
    width: 18px;
    height: 18px;
    flex-shrink: 0;
    padding: 0;
    margin: 0;
    accent-color: var(--blue);
  }
  .brief .play {
    width: 100%;
    padding: 15px 10px;
    font-size: 20px;
    text-transform: uppercase;
    letter-spacing: 0.02em;
    margin-bottom: 20px;
  }
  .match-details {
    font-size: 12px;
    color: var(--muted);
  }
  .match-details summary {
    cursor: pointer;
    padding: 5px 0;
    color: #a3b8d1;
  }
  .match-details p {
    font-size: 12px;
    margin-top: 12px;
  }
  .fixture-details {
    display: grid;
    grid-template-columns: max-content minmax(0, 1fr);
    gap: 10px 14px;
    margin: 16px 0 0;
    line-height: 1.5;
  }
  .fixture-details dt {
    color: #9aabc2;
  }
  .fixture-details dd {
    margin: 0;
    color: var(--text);
    overflow-wrap: anywhere;
  }
  .teammate {
    margin-bottom: 5px;
  }
  .lock {
    margin: 10px 0;
    color: #efc487;
    font-size: 13px;
  }
  .epilogue {
    padding: 20px 0;
    border-top: 1px solid var(--line);
  }
  .epilogue h3 {
    margin-bottom: 14px;
  }
  .workshop-banner {
    min-height: 200px;
    padding: 34px;
    background-size: cover;
    background-position: center;
    display: flex;
    flex-direction: column;
    justify-content: center;
    margin: 12px 28px 0;
    border-radius: 5px;
    overflow: hidden;
  }
  .workshop-banner p {
    max-width: 440px;
    margin-top: 12px;
    color: #d8e4f4;
  }
  .workshop-banner .eyebrow {
    margin-top: 0;
    margin-bottom: 10px;
  }
  .workshop-banner h2 {
    font-size: 30px;
  }
  .workshop-body {
    padding: 28px;
    flex: 1;
  }
  .flow {
    margin-bottom: 32px;
    padding-bottom: 30px;
    border-bottom: 1px solid var(--line);
  }
  .flow > div:first-child {
    max-width: 720px;
    margin-bottom: 22px;
  }
  .flow h2 {
    margin-bottom: 12px;
  }
  .flow-modes,
  .modules {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 18px;
  }
  .flow-modes article,
  .module {
    background: #102740;
    padding: 22px;
    display: flex;
    flex-direction: column;
    border: 1px solid #ffffff12;
    border-radius: 5px;
    min-width: 0;
  }
  .flow-modes article.equipped {
    border-color: #4aa6fa;
  }
  .flow-modes h3 {
    margin-bottom: 10px;
  }
  .flow-modes article {
    min-height: 210px;
  }
  .illustrated-card {
    position: relative;
    isolation: isolate;
    overflow: hidden;
  }
  .illustrated-card .card-art {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: 0.32;
    pointer-events: none;
    z-index: -2;
  }
  .illustrated-card::after {
    content: "";
    position: absolute;
    inset: 0;
    background: linear-gradient(90deg, #0b1c33ef, #0b1c3370 65%, #0b1c3320);
    pointer-events: none;
    z-index: -1;
  }
  .module .effect,
  .flow-modes .effect {
    display: block;
    font-size: 23px;
    font-weight: 500;
    color: #f1f7ff;
    margin: 15px 0;
  }
  .equipment-summary {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 20px;
    margin-bottom: 24px;
    font-size: 13px;
  }
  .equipment-summary span {
    color: var(--muted);
  }
  .equipment-summary strong {
    font-weight: 400;
  }
  .module-top {
    display: flex;
    justify-content: space-between;
    gap: 10px;
  }
  .module-top > span {
    font-size: 10px;
    color: var(--muted);
    letter-spacing: 0.06em;
  }
  .module h3 {
    margin: 4px 0 12px;
  }
  .module p {
    font-size: 13px;
  }
  .module .small {
    font-size: 12px;
  }
  .module-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    margin-top: auto;
    padding-top: 16px;
    font-size: 12px;
    color: var(--muted);
  }
  .module-actions button {
    background: #0b1d33c9;
    font-size: 12px;
  }
  .unavailable {
    opacity: 0.7;
  }
  .settings {
    display: grid;
    grid-template-columns: 1fr 1.2fr;
    gap: 45px;
    padding: 35px 32px;
    flex: 1;
  }
  .settings article {
    min-width: 0;
  }
  .settings h2 {
    margin-bottom: 26px;
  }
  .settings h3 {
    margin: 28px 0 10px;
  }
  .settings .small {
    margin: 10px 0 20px;
  }
  .club-name-actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
  }
  .settings .history {
    grid-column: 1/-1;
    border-top: 1px solid var(--line);
    padding-top: 22px;
  }
  .history p {
    font-size: 13px;
    margin-bottom: 8px;
  }
  .history strong {
    color: var(--text);
    font-weight: 400;
  }
  footer {
    margin-top: auto;
    padding: 26px 28px 18px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 20px;
    flex-wrap: wrap;
    font-size: 11px;
    color: #9aacc4;
  }
  .loading {
    padding: 30px;
  }
  @media (min-width: 1550px) {
    .masthead {
      padding: 26px 38px;
    }
    .season {
      grid-template-columns: 260px minmax(350px, 1fr) 335px;
      gap: 35px;
      padding: 35px 38px;
    }
    .chapter-cover {
      height: 245px;
    }
    .fixture {
      min-height: 88px;
    }
    .season aside button {
      min-height: 78px;
    }
    .season aside strong {
      font-size: 17px;
    }
    .brief h2 {
      font-size: 30px;
    }
    .brief-label {
      font-size: 25px;
    }
    .match-facts {
      font-size: 17px;
    }
    .objectives {
      font-size: 16px;
    }
  }
  @media (max-width: 1100px) {
    .masthead {
      gap: 22px;
      padding: 22px;
    }
    .wordmark {
      padding-right: 18px;
    }
    .masthead nav {
      gap: 4px;
    }
    .masthead nav button {
      font-size: 16px;
      padding: 8px 13px;
    }
    .totals {
      gap: 16px;
      font-size: 14px;
    }
    .totals > span + span {
      padding-left: 16px;
    }
    .season {
      grid-template-columns: 170px minmax(250px, 1fr) 235px;
      gap: 20px;
      padding: 22px;
    }
    .season aside strong {
      font-size: 13px;
    }
    .season aside button {
      padding: 14px 7px;
      gap: 8px;
    }
    .chapter-cover {
      height: 185px;
    }
    .chapter-cover h2 {
      font-size: 29px;
    }
    .brief h2 {
      font-size: 24px;
    }
    .match-facts {
      font-size: 13px;
    }
    .fixture strong {
      font-size: 16px;
    }
    .modules {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .club-title {
      max-width: 150px;
    }
    .setup-hint {
      margin: 0 22px;
    }
  }
  @media (max-width: 850px) {
    .masthead {
      gap: 18px;
    }
    .totals {
      font-size: 13px;
    }
    .club-title {
      max-width: 120px;
    }
    .masthead nav button {
      font-size: 15px;
      padding: 8px 10px;
    }
    .wordmark {
      font-size: 24px;
      padding-right: 15px;
    }
    .season {
      grid-template-columns: 165px minmax(0, 1fr);
    }
    .brief {
      grid-column: 2;
      padding-top: 18px;
    }
    .season aside {
      grid-row: 1/3;
    }
    .flow-modes {
      grid-template-columns: 1fr;
    }
    .flow-modes article {
      min-height: 180px;
    }
    .settings {
      gap: 28px;
      padding: 28px;
    }
    .brief-label {
      margin-bottom: 20px;
    }
    .match-facts {
      margin-bottom: 18px;
    }
    .brief .play {
      max-width: 360px;
    }
  }
  @media (max-width: 600px) {
    .masthead {
      padding: 20px;
      gap: 18px;
    }
    .wordmark {
      border: 0;
      padding: 0;
      margin-right: auto;
    }
    .masthead nav {
      order: 3;
      width: 100%;
      justify-content: space-between;
    }
    .masthead nav button {
      flex: 1;
      font-size: 17px;
    }
    .totals {
      gap: 12px;
      margin-left: 0;
    }
    .totals > span + span {
      padding-left: 12px;
    }
    .totals small {
      font-size: 12px;
    }
    .club-title {
      max-width: 110px;
    }
    .season {
      display: flex;
      flex-direction: column;
      padding: 20px;
      gap: 26px;
    }
    .season aside {
      width: 100%;
    }
    .season-title {
      font-size: 21px;
      margin-bottom: 12px;
    }
    .season aside button {
      min-height: 55px;
    }
    .season aside strong {
      font-size: 15px;
    }
    .season aside small {
      display: inline;
      margin-left: 8px;
    }
    .chapter,
    .brief {
      width: 100%;
    }
    .chapter-cover {
      height: 190px;
    }
    .fixture {
      padding: 15px 12px;
    }
    .brief .play {
      max-width: none;
    }
    .modules,
    .settings {
      grid-template-columns: 1fr;
    }
    .settings .history {
      grid-column: 1;
    }
    .equipment-summary {
      align-items: start;
      flex-direction: column;
      gap: 9px;
    }
    .workshop-banner {
      margin: 4px 20px 0;
      padding: 25px;
      min-height: 190px;
    }
    .workshop-banner h2 {
      font-size: 26px;
    }
    .workshop-body {
      padding: 25px 20px;
    }
    .settings {
      padding: 25px 20px;
    }
    .setup-hint {
      margin: 0 20px;
      align-items: start;
    }
    .setup-hint button {
      white-space: nowrap;
    }
    footer {
      padding: 25px 20px;
      gap: 14px;
    }
    .match-details p {
      font-size: 13px;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    button {
      transition: none;
    }
  }

  /* Keep RLBot's shell and theme intact; scope menu surfaces to Story Mode. */
  .career .fixture {
    background: transparent;
  }
  .career .fixture.selected {
    background: var(--blue);
  }
  .career .fixture:not(.selected):hover {
    background: #ffffff08;
  }
  .career .masthead nav button {
    background: transparent;
  }
  .career .masthead nav button.active {
    background: var(--blue);
  }
  .career .setup-hint button {
    background: transparent;
  }

  /* Workshop cards: stronger contrast on hover and keyboard focus. */
  .illustrated-card {
    transition:
      border-color 0.15s ease,
      background-color 0.15s ease;
  }
  .illustrated-card::after {
    transition: background-color 0.15s ease;
  }
  .career .illustrated-card:hover,
  .career .illustrated-card:focus-within {
    border-color: #329bff;
    background-color: #142f4f;
  }
  .career .illustrated-card:hover::after,
  .career .illustrated-card:focus-within::after {
    background-color: #0b1c33cf;
  }
  @media (prefers-reduced-motion: reduce) {
    .illustrated-card,
    .illustrated-card::after {
      transition: none;
    }
  }

  .series-panel {
    padding: 14px 0 18px;
    display: flex;
    flex-direction: column;
    gap: 9px;
    color: var(--muted);
    font-size: 13px;
  }
  .series-panel strong {
    font-size: 22px;
    color: var(--text);
    font-weight: 500;
  }
  .series-panel small {
    font-size: 12px;
  }
  .series-reset {
    align-self: start;
    background: transparent;
    border: 0;
    color: #9acbff;
    padding: 4px 0;
    font-size: 12px;
  }

  .recruitment {
    padding: 28px;
    flex: 1;
  }
  .recruitment-heading {
    display: flex;
    gap: 24px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 28px;
  }
  .recruitment-heading h2 {
    margin-bottom: 12px;
  }
  .recruitment-heading > span {
    color: var(--muted);
    white-space: nowrap;
    font-size: 13px;
  }
  .recruitment-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 22px;
  }
  .recruit-card {
    overflow: hidden;
    border: 1px solid #ffffff1a;
    border-radius: 5px;
    background: #102740;
    display: flex;
    flex-direction: column;
    min-width: 0;
    transition: border-color 0.15s ease;
  }
  .recruit-card:hover,
  .recruit-card:focus-within {
    border-color: #329bff;
  }
  .recruit-card.recruited {
    border-color: #3b78a9;
  }
  .driver-portrait {
    display: block;
    width: 100%;
    height: auto;
    aspect-ratio: 3 / 2;
    object-fit: cover;
    object-position: center top;
    flex-shrink: 0;
  }
  .recruit-content {
    padding: 22px;
    display: flex;
    flex-direction: column;
    flex: 1;
  }
  .recruit-content h3 {
    margin-bottom: 12px;
    font-size: 23px;
  }
  .recruit-content > p {
    font-size: 13px;
    margin-bottom: 12px;
  }
  .recruit-content .recruit-level {
    color: #c9e3ff;
  }
  .recruit-content details {
    font-size: 12px;
    margin: 2px 0 18px;
    color: var(--muted);
  }
  .recruit-content summary {
    cursor: pointer;
    color: #a9c9e9;
  }
  .recruit-content details p {
    font-size: 12px;
    margin-top: 10px;
  }
  .recruit-actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
    align-items: center;
    margin-top: auto;
    padding-top: 12px;
    color: #a9c9e9;
    font-size: 13px;
  }
  .lineup-driver {
    display: flex;
    gap: 12px;
    align-items: center;
    margin: 8px 0 4px;
  }
  .lineup-driver > img {
    width: 48px;
    height: 56px;
    object-fit: cover;
    border-radius: 4px;
  }
  .lineup-driver label {
    min-width: 0;
    flex: 1;
  }
  .lineup-driver select {
    margin-bottom: 0;
    padding: 9px;
  }
  .career button.recruit-link {
    background: transparent;
    border: 0;
    color: #9acbff;
    padding: 9px 0;
    font-size: 12px;
    display: block;
    margin-bottom: 12px;
  }
  .club-roster {
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
    margin: 16px 0;
  }
  .club-roster > div {
    display: flex;
    gap: 10px;
    align-items: center;
    font-size: 14px;
  }
  .club-roster img {
    width: 44px;
    height: 48px;
    object-fit: cover;
    border-radius: 4px;
  }
  @media (max-width: 1100px) {
    .recruitment-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .masthead {
      gap: 20px;
    }
    .masthead nav {
      gap: 4px;
    }
    .masthead nav button {
      padding: 8px 11px;
      font-size: 16px;
    }
  }
  @media (max-width: 600px) {
    .recruitment {
      padding: 22px 20px;
    }
    .recruitment-grid {
      grid-template-columns: 1fr;
    }
    .recruitment-heading {
      align-items: start;
      flex-direction: column;
      gap: 14px;
    }
    .masthead nav {
      flex-wrap: wrap;
    }
    .masthead nav button {
      flex: 1 1 100px;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .recruit-card {
      transition: none;
    }
  }

  .vanilla-controls {
    margin: 24px 28px 0;
  }
  .vanilla-switch {
    display: flex;
    justify-content: space-between;
    gap: 24px;
    align-items: center;
    padding: 18px 20px;
    background: #ffffff06;
    border: 1px solid var(--line);
    border-radius: 5px;
    cursor: pointer;
  }
  .vanilla-switch > span {
    display: flex;
    flex-direction: column;
    gap: 7px;
  }
  .vanilla-switch strong {
    font-size: 16px;
    font-weight: 500;
  }
  .vanilla-switch small {
    font-size: 12px;
  }
  .vanilla-switch input {
    appearance: none;
    position: relative;
    width: 50px;
    height: 28px;
    flex-shrink: 0;
    margin: 0;
    padding: 0;
    background: #35465d;
    border: 1px solid #ffffff30;
    border-radius: 20px;
    cursor: pointer;
  }
  .vanilla-switch input::after {
    content: "";
    position: absolute;
    width: 20px;
    height: 20px;
    left: 3px;
    top: 3px;
    border-radius: 50%;
    background: #f2f6ff;
    transition: transform 0.15s ease;
  }
  .vanilla-switch input:checked {
    background: var(--blue);
    border-color: var(--blue);
  }
  .vanilla-switch input:checked::after {
    transform: translateX(22px);
  }
  .vanilla-switch input:disabled {
    cursor: not-allowed;
  }
  .vanilla-note {
    font-size: 12px;
    color: #b9d9fa;
    padding: 12px 2px 0;
    max-width: 850px;
  }
  @media (max-width: 600px) {
    .vanilla-controls {
      margin: 22px 20px 0;
    }
    .vanilla-switch {
      padding: 16px;
      gap: 16px;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .vanilla-switch input::after {
      transition: none;
    }
  }

  .fixture-status {
    display: inline-flex;
    justify-content: center;
    align-items: center;
    width: 23px;
    min-height: 23px;
    line-height: 1;
    font-size: 17px;
  }
  .fixture-status.done {
    color: #8df3b0;
    background: #103c29;
    border: 1px solid #65d894;
    border-radius: 4px;
    font-weight: 600;
  }
  .fixture-status.failed {
    color: #ff9a9a;
    background: #421f2a;
    border: 1px solid #ed777f;
    border-radius: 4px;
    font-weight: 500;
  }
  .fixture.selected .fixture-status.done {
    color: #a5ffc5;
    background: #0d3a29;
    border-color: #a5ffc5;
  }
  .fixture.selected .fixture-status.failed {
    color: #ffc0c0;
    background: #4b1d29;
    border-color: #ffb0b0;
  }
</style>
