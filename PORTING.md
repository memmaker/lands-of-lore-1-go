# Porting contract

Source of truth: `~/Projects/lands-of-lore-1-re/decomp/src` (MAIN.EXE of the GOG/CD version
1.02d, byte-exact). Original data: imported with `lol -import setup_lands_of_lore_*.exe -data game`.
Read `~/notes/c-to-go.md` first. Its rules #0–#6 are binding: behaviour parity, no far
pointers or memory emulation, no magic numbers or hex, readable names, idiomatic Go, obvious
bugs may be fixed (mark them `// fix:`).

## Packages

| package | replaces | notes |
|---|---|---|
| `internal/host` | DOS, BIOS, timer IRQ | lockstep Ebiten host: `Yield`, `Wait`, `KeyQueue`, mouse |
| `internal/gameimport` | — | `-import` (shared with other ports; only the GAME.DAT sniff was added here) |
| `cmd/lol` | MAIN.EXE start-up | flags, import, launcher |

## Conventions

- C globals become fields of typed state structs, never byte arrays addressed by offset.
  Saved files are read and written field by field in the original layout.
- Every blocking wait (page flips, delays, input polls) calls `host.Yield`/`host.Wait`.
- Every function keeps a comment `// C: sNNN fn_SSSS_OOOO` naming its original. That comment
  is the only place hex labels may appear.
- Constants get names, with the original value in decimal.
