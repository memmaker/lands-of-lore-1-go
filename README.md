# lands-of-lore-1-go

Lands of Lore: The Throne of Chaos (Westwood, 1993, DOS) ported to Go and Ebiten from the
byte-exact decompilation in [lands-of-lore-1-re](https://github.com/memmaker/lands-of-lore-1-re).
Bring your own game files:

    go run ./cmd/lol -import setup_lands_of_lore_-_the_throne_of_chaos_1.02d_(28044).exe -data game
    go run ./cmd/lol -data game

`-import` takes a folder, zip, CD image (ISO/BIN/CUE) or the GOG installer.

Work in progress: see CHECKLIST.md.
