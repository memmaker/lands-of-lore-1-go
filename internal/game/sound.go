package game

// Sound and music are not ported yet (CHECKLIST: Sound and music). These are the entry
// points the game calls; until the sound milestone they are silent, so the rest of the
// game can run.

// stopSounds stops the music and effects (C: fn_067a_0003).
func (g *Game) stopSounds() {}

// stopMusic stops the music (C: fn_067a_023d).
func (g *Game) stopMusic() {}

// playMusic starts track n, -1 asks for the current one (C: fn_067a_01d1).
func (g *Game) playMusic(n int) int {
	old := g.musicTrack
	if n != -1 {
		g.musicTrack = n
	}
	return old
}

// loadSoundFile loads the effects of a level (C: fn_067a_084d).
func (g *Game) loadSoundFile(name string, n int) {}

// soundLoaded says whether effect n is loaded (C: fn_067a_0649).
func (g *Game) soundLoaded(n int) bool { return false }

// queueSpeechTrack selects a music track for later (C: fn_067a_003e).
func (g *Game) queueTrack(n int) {}

// playSound plays effect sound (C: fn_067a_0401).
func (g *Game) playSound(sound, volume int) {}

// playSoundAt plays a sound from a block, louder the nearer (C: fn_067a_02f5).
func (g *Game) playSoundAt(sound, block int) {}

// playSoundEffect plays a digital effect through the sound driver (C: fn_17d6_0006).
func (g *Game) playSoundEffect(n int) {}

// playDelayedSounds plays the effects held back while delaySounds was set; the
// count is never reset (sic), so earlier ones play again (C: fn_067a_02c7).
func (g *Game) playDelayedSounds() {}
