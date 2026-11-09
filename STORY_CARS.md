# Character cars

Each named Story driver keeps the same body across fixtures, difficulty settings and teams. This applies to recruited teammates and campaign rivals. The player's own car is untouched.

These are presentation choices, not an aggression or skill setting. The bot remains KilobyteReduced. Bodies are selected from Epic's **Octane hitbox** group; equal hitbox classification is not a claim that every visual model or wheel placement is identical.

## Recruits

| Driver | Car | Visual direction |
| --- | --- | --- |
| Morgan Vale | Backfire | A mechanic's old project car |
| Casey Reed | Fennec | Direct, competitive, understated |
| Jamie Cole | Scarab | An unconventional choice for the club joker |
| Rowan Hayes | Twinzer | Flashier, more restless |
| Lena Cross | Takumi | Compact and restrained |
| Avery Stone | Dingo | A confident, squared-off presence |
| Silas Ward | Octane ZSR | Precise, serious competition styling |
| Mara Voss | Jackal | A sharper silhouette |
| Viktor Rook | Road Hog XL | Heavy, imposing presence |

## Rivals

| Driver | Car |
| --- | --- |
| Logan Price | Octane |
| Jules Mercer | Takumi RX-T |
| Finn Archer | Zippy |
| Skye Bennett | Vulcan |
| Blake Porter | Road Hog |
| Maya Torres | Mudcat GXT |
| Oscar Lane | Grog |
| Zoe Walker | Fennec ZR-F |
| Noah Carter | Gizmo |
| Tess Moreno | Harbinger GXT |
| Luca Hayes | Outlaw GXT |
| Aria Knox | Proteus |

## Catalogue and implementation

The 31-body curated catalogue and product IDs are in [story_cars.go](story_cars.go). It includes additional Octane-group options for later assignments, while excluding console-exclusive bodies and real-world licensed cars. It is not intended to include every licensed Octane-group release.

Hitbox classification: [Epic Games — Rocket League Car Hitboxes](https://www.epicgames.com/help/c-37599050/c-32343914/a20257614?lang=en-US), checked 9 November 2026. Product IDs are checked against the existing `frontend/src/assets/items.csv` body catalogue.

Assignments are applied to copied launch configurations for both teams. The installed bot TOMLs are never edited. Wheels, boost and team colours are retained; body-specific decals, body/decal paint and engine overrides are cleared to avoid incompatible Octane cosmetics.

Automated checks verify the body IDs, roster/rival coverage, both launch teams, and isolation from the human and shared bot loadout. Actual rendering and availability of every body still require in-game verification; no live match is restarted by those tests.
