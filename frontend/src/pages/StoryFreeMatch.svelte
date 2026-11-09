<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { App, type StoryRecruit } from "../../bindings/gui";

  let {
    paths,
    ranks,
    rankNames,
    locked,
    onlaunch,
    roster,
    portraits,
  }: {
    roster: StoryRecruit[];
    portraits: Record<string, string>;
    paths: string[];
    ranks: Record<string, number>;
    rankNames: string[];
    locked: boolean;
    onlaunch: () => Promise<unknown>;
  } = $props();

  const arenas = [
    ["BeckwithPark", "Beckwith Park"],
    ["Farmstead", "Farmstead"],
    ["UrbanCentral", "Urban Central"],
    ["Mannfield", "Mannfield"],
    ["DFHStadium", "DFH Stadium"],
    ["ChampionsField", "Champions Field"],
  ];
  const initialRank = untrack(() => {
    const rank = ranks["2v2"];
    return rank >= 1 && rank <= 22 ? rank : 7;
  });
  const storageKey = "RLSTORY_FREE_MATCH_V1";
  let size = $state(2),
    arena = $state("BeckwithPark");
  let blueDrivers = $state(["rookie", "pro"]);
  let orangeDrivers = $state(["Beast from the East", "Phoenix CS", "Fury"]);
  const driver = (id: string) => roster.find((r) => r.id === id);
  const portrait = (id: string) => portraits[driver(id)?.portrait ?? ""];
  function activeDrivers() {
    return [...blueDrivers.slice(0, size - 1), ...orangeDrivers.slice(0, size)];
  }
  function ensureUnique() {
    const seen = new Set<string>();
    for (const [ids, count] of [
      [blueDrivers, size - 1],
      [orangeDrivers, size],
    ] as [string[], number][]) {
      for (let i = 0; i < count; i++) {
        if (!driver(ids[i]) || seen.has(ids[i]))
          ids[i] = roster.find((r) => !seen.has(r.id))!.id;
        seen.add(ids[i]);
      }
    }
  }
  function changeFormat(next: number) {
    size = next;
    ensureUnique();
  }
  function usedCharacter(id: string, current: string) {
    return id !== current && activeDrivers().includes(id);
  }
  let blueRanks = $state([initialRank, initialRank]);
  let orangeRanks = $state([initialRank, initialRank, initialRank]);
  let launching = $state(false),
    error = $state("");
  const disabled = $derived(locked || launching);

  onMount(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey) ?? "null");
      const validRanks = (
        values: unknown,
        length: number,
      ): values is number[] =>
        Array.isArray(values) &&
        values.length === length &&
        values.every((v) => Number.isInteger(v) && v >= 1 && v <= 22);
      if (
        saved &&
        [1, 2, 3].includes(saved.size) &&
        arenas.some((a) => a[0] === saved.arena) &&
        validRanks(saved.blueRanks, 2) &&
        validRanks(saved.orangeRanks, 3)
      ) {
        size = saved.size;
        arena = saved.arena;
        blueRanks = saved.blueRanks;
        orangeRanks = saved.orangeRanks;
        const validDrivers = (ids: unknown, length: number): ids is string[] =>
          Array.isArray(ids) &&
          ids.length === length &&
          ids.every((id) => typeof id === "string" && driver(id));
        if (validDrivers(saved.blueDrivers, 2)) blueDrivers = saved.blueDrivers;
        if (validDrivers(saved.orangeDrivers, 3))
          orangeDrivers = saved.orangeDrivers;
        ensureUnique();
      }
    } catch {
      /* A malformed local preset should not prevent match creation. */
    }
  });

  async function launch() {
    launching = true;
    error = "";
    try {
      try {
        localStorage.setItem(
          storageKey,
          JSON.stringify({
            size,
            arena,
            blueRanks,
            orangeRanks,
            blueDrivers,
            orangeDrivers,
          }),
        );
      } catch {
        /* Storage is optional. */
      }
      await App.StartStoryFreeMatch(
        {
          size,
          arena,
          blueRanks: blueRanks.slice(0, size - 1),
          orangeRanks: orangeRanks.slice(0, size),
          blueDrivers: blueDrivers.slice(0, size - 1),
          orangeDrivers: orangeDrivers.slice(0, size),
        },
        paths,
        localStorage.getItem("MS_LAUNCHER") ?? "",
        localStorage.getItem("MS_LAUNCHER_ARG") ?? "",
      );
      await onlaunch();
    } catch (e) {
      error = String(e);
    } finally {
      launching = false;
    }
  }
</script>

<section class="free-match" aria-label="Free match">
  <header>
    <p class="eyebrow">FREE MATCH</p>
    <h2>Your match. Your line-up.</h2>
    <p>
      Set each bot's level independently. Mix stronger opponents with weaker
      teammates, or build a balanced match.
    </p>
  </header>
  <div class="setup">
    <label
      >Format<select
        aria-label="Format"
        value={size}
        onchange={(e) => changeFormat(Number(e.currentTarget.value))}
        {disabled}
      >
        <option value={1}>1v1 · Duel</option><option value={2}
          >2v2 · Doubles</option
        ><option value={3}>3v3 · Standard</option>
      </select></label
    >
    <label
      >Arena<select aria-label="Arena" bind:value={arena} {disabled}>
        {#each arenas as [id, name]}<option value={id}>{name}</option>{/each}
      </select></label
    >
  </div>
  <div class="teams">
    <section class="team blue" aria-label="Your team">
      <h3>Your team <span>{size} players</span></h3>
      <div class="player human">
        <span class="slot">01</span>
        <div>
          <strong>You</strong><small>Your car and camera settings</small>
        </div>
        <span class="human-tag">PLAYER</span>
      </div>
      {#each blueRanks.slice(0, size - 1) as rank, i}
        <div class="player">
          <span class="slot">0{i + 2}</span>
          <img class="portrait" src={portrait(blueDrivers[i])} alt="" />
          <div class="driver-info">
            <label
              ><span class="sr-only">Teammate {i + 1} character</span><select
                aria-label={`Teammate ${i + 1} character`}
                value={blueDrivers[i]}
                {disabled}
                onchange={(e) => (blueDrivers[i] = e.currentTarget.value)}
                >{#each roster as r}<option
                    value={r.id}
                    disabled={usedCharacter(r.id, blueDrivers[i])}
                    >{r.name}</option
                  >{/each}</select
              ></label
            ><small>{driver(blueDrivers[i])?.carName}</small>
          </div>
          <label
            ><span class="sr-only">{driver(blueDrivers[i])?.name} rank</span
            ><select
              value={rank}
              {disabled}
              onchange={(e) => (blueRanks[i] = Number(e.currentTarget.value))}
            >
              {#each rankNames.slice(1) as name, j}<option value={j + 1}
                  >{name}</option
                >{/each}
            </select></label
          >
        </div>
      {/each}
    </section>
    <section class="team orange" aria-label="Opponents">
      <h3>Opponents <span>{size} players</span></h3>
      {#each orangeRanks.slice(0, size) as rank, i}
        <div class="player">
          <span class="slot">0{i + 1}</span>
          <img class="portrait" src={portrait(orangeDrivers[i])} alt="" />
          <div class="driver-info">
            <label
              ><span class="sr-only">Opponent {i + 1} character</span><select
                aria-label={`Opponent ${i + 1} character`}
                value={orangeDrivers[i]}
                {disabled}
                onchange={(e) => (orangeDrivers[i] = e.currentTarget.value)}
                >{#each roster as r}<option
                    value={r.id}
                    disabled={usedCharacter(r.id, orangeDrivers[i])}
                    >{r.name}</option
                  >{/each}</select
              ></label
            ><small>{driver(orangeDrivers[i])?.carName}</small>
          </div>
          <label
            ><span class="sr-only">{driver(orangeDrivers[i])?.name} rank</span
            ><select
              value={rank}
              {disabled}
              onchange={(e) => (orangeRanks[i] = Number(e.currentTarget.value))}
            >
              {#each rankNames.slice(1) as name, j}<option value={j + 1}
                  >{name}</option
                >{/each}
            </select></label
          >
        </div>
      {/each}
    </section>
  </div>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <div class="launch-bar">
    <div>
      <p class="rules">
        Standard Rocket League rules · No equipment or Flow · No career rewards
      </p>
      <p class="hint">
        Characters change the name, portrait and car. All bots use
        KilobyteReduced's playstyle; ranks are estimates.
      </p>
    </div>
    <button class="launch" {disabled} onclick={launch}
      >{launching
        ? "Preparing…"
        : locked
          ? "Match in progress"
          : "Play free match"}</button
    >
  </div>
</section>

<style>
  .free-match {
    padding: 28px;
    flex: 1;
    color: var(--text);
  }
  header {
    max-width: 700px;
    margin-bottom: 28px;
  }
  .eyebrow {
    font-size: 11px;
    letter-spacing: 0.1em;
    color: var(--muted);
    margin: 0 0 10px;
  }
  h2 {
    font-size: 28px;
    font-weight: 500;
    margin: 0 0 12px;
  }
  header > p:last-child {
    color: var(--muted);
    font-size: 14px;
    line-height: 1.6;
    margin: 0;
  }
  .setup {
    display: flex;
    gap: 20px;
    flex-wrap: wrap;
    margin-bottom: 28px;
  }
  .setup > label {
    min-width: 180px;
    flex: 0 1 280px;
  }
  label {
    display: block;
    font-size: 12px;
    color: var(--muted);
  }
  select {
    width: 100%;
    display: block;
    margin-top: 8px;
    padding: 11px 12px;
    border: 1px solid #ffffff25;
    border-radius: 4px;
    background: #102740;
    color: var(--text);
    font: inherit;
    font-size: 14px;
  }
  select:focus-visible {
    outline: 2px solid #329bff;
    outline-offset: 2px;
  }
  .teams {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 24px;
  }
  .team {
    background: #102740;
    border: 1px solid #ffffff15;
    border-top: 3px solid #329bff;
    border-radius: 5px;
    padding: 20px;
  }
  .orange {
    border-top-color: #f39b42;
  }
  h3 {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 10px;
    font-size: 20px;
    font-weight: 500;
    margin: 0 0 14px;
  }
  h3 > span {
    font-size: 12px;
    color: var(--muted);
  }
  .player {
    display: flex;
    gap: 12px;
    align-items: center;
    min-height: 85px;
    padding: 14px 0;
    border-top: 1px solid #ffffff16;
    box-sizing: border-box;
  }
  .slot {
    font-size: 12px;
    color: var(--muted);
  }
  .portrait {
    width: 48px;
    height: 56px;
    object-fit: cover;
    object-position: center top;
    border-radius: 4px;
    flex-shrink: 0;
  }
  .player > div {
    flex: 1;
    min-width: 0;
  }
  .player strong {
    display: block;
    font-size: 15px;
    font-weight: 500;
  }
  .player small {
    display: block;
    margin-top: 6px;
    font-size: 12px;
    color: var(--muted);
  }
  .player > label {
    width: 155px;
    flex-shrink: 0;
  }
  .driver-info select {
    width: 100%;
    padding: 8px 10px;
    min-width: 0;
    font-weight: 500;
  }
  .player select {
    margin: 0;
  }
  .human-tag {
    font-size: 10px;
    letter-spacing: 0.08em;
    color: #8bc9ff;
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  .rules {
    font-size: 13px;
    color: var(--text);
    margin: 0 0 8px;
  }
  .hint {
    font-size: 12px;
    color: var(--muted);
    line-height: 1.6;
    margin: 0;
  }
  .error {
    background: #532c37;
    color: #ffe3e8;
    padding: 12px 16px;
    font-size: 13px;
  }
  .launch-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
    position: sticky;
    bottom: 0;
    z-index: 2;
    margin-top: 24px;
    padding: 20px;
    background: #0b1c33;
    border: 1px solid #ffffff20;
    border-radius: 5px;
  }
  .free-match button.launch {
    background: #f89508;
    background-color: #f89508;
    color: #102037;
    backdrop-filter: none !important;
    -webkit-backdrop-filter: none !important;
    box-shadow: none;
    isolation: auto;
    flex-shrink: 0;
    border: 0;
    border-radius: 4px;
    font-size: 18px;
    font-weight: 500;
    padding: 14px 28px;
    cursor: pointer;
  }
  .free-match button.launch:hover:not(:disabled) {
    background: #ffab28;
  }
  .free-match button.launch:focus-visible {
    outline: 3px solid #8bc9ff;
    outline-offset: 3px;
  }
  :disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
  @media (max-width: 1000px) {
    .teams {
      grid-template-columns: 1fr;
    }
    .player > label {
      width: 180px;
    }
  }
  @media (max-width: 600px) {
    .launch-bar {
      flex-direction: column;
      align-items: stretch;
      gap: 16px;
      padding: 16px;
    }
    .free-match {
      padding: 22px 20px;
    }
    .team {
      padding: 16px;
    }
    .player {
      flex-wrap: wrap;
    }
    .player > label {
      width: 100%;
      margin-left: 26px;
    }
    .setup > label {
      flex: 1 1 100%;
    }
    .launch {
      width: 100%;
    }
  }
</style>
