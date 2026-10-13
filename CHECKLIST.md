# Lands of Lore: Go port — checklist

Rules in ~/notes/c-to-go.md, conventions in PORTING.md.

- [x] Repo skeleton: go.mod, cmd/lol, host (Ebiten lockstep), gameimport (-import incl. GOG GAME.DAT ISO)
- [ ] Data layer: PAK archives, LCW/RLE decompression, CPS, shapes, fonts, palettes, WSA
  - [x] FILEDATA.FDT index, PAK members, loose-file override (data.Files)
  - [x] LCW (format80) and RLE decoders
  - [x] PAK members: offsets/sizes re-read from each archive's own directory (FDT values are stale, s034)
  - [x] CPS (o2ded), XOR delta (s062), WSA decode and frame seek (o2dfd; drawing waits for gfx)
  - [ ] shapes, fonts, palette files
- Milestone 1 (2026-10-13): data layer reads PAK members and decodes LCW/RLE; the game timer is 60 Hz (ovr_2e13_0057(g_295c, 60)); the Westwood graphics library is fully analysed (see PORTING.md notes), gfx package is next.
- [ ] Graphics: Westwood library drawing (pages, shape blitter, text, palette fades, mouse cursor)
- [ ] Start-up and main menu (o2db0, o2d50, o2d56)
- [ ] Intro (TIM/WSA sequences), character selection
- [ ] Scripts: EMC interpreter and opcodes (s018, s019), TIM (s001, s002)
- [ ] Levels, 3D view renderer (o2d2c, o2cda), movement, automap
- [ ] Items, inventory, characters, portraits, stats screen
- [ ] Monsters, combat, magic
- [ ] Dialogue/scenes, text window, speech
- [ ] Save/load
- [ ] Sound and music (AdLib .ADL, VOC)
- [ ] Ending
- [ ] Parity checks against DOSBox-X
- [ ] wasm build, goplay registration
