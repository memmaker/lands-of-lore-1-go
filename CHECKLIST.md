# Lands of Lore: Go port — checklist

Rules in ~/notes/c-to-go.md, conventions in PORTING.md.

- [x] Repo skeleton: go.mod, cmd/lol, host (Ebiten lockstep), gameimport (-import incl. GOG GAME.DAT ISO)
- [ ] Data layer: PAK archives, LCW/RLE decompression, CPS, shapes, fonts, palettes, WSA
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
