package game

import (
	"fmt"
	"strings"

	"github.com/memmaker/lands-of-lore-1-go/internal/clock"
	"github.com/memmaker/lands-of-lore-1-go/internal/sound"
)

// The music and the effects (C: s003, with s030 and s032 the driver front ends). The
// AdLib card plays both from one sound file: LORE<file><song>.ADL with the music on,
// LORESFX.ADL with it off. The digital effects (VOC) and speech go through the Sound
// Blaster mixer when the digital driver is set up (C: g_3749).

const (
	firstMusicTrack = 250 // track numbers from here are music (C: n -= 250)
	effectsFile     = "LORESFX"
	soundFileExt    = ".ADL"
	soundBufferSize = 26000 // C: g_4d62, 0x6590 bytes

	noSoundFile    = -1
	effectsLoaded  = -2 // LORESFX is loaded (C: g_1e90 == -2)
	fadeTrack      = 1  // the track that fades the music out (C: fn_178d_01b0)
	stopTrack      = 0
	stopPause      = 5 // passes of the library's delay after the stop track (C: fn_2120_0007(5))
	fadePause      = 3
	fullVolume     = 255
	silentVolume   = 0
	maxSoundEffect = 250 // C: fn_17d6_0019's n < 250
	adlibOnlyCount = 1   // with the effects off only effects 0 and 1 still play

	musicVolumeScale, musicVolumeBase   = 145, 110 // the driver volume of a slider value
	effectVolumeScale, effectVolumeBase = 200, 55

	maxDelayedSounds = 10 // C: g_2148[10]
	farEffect        = 15 // effects farther away than this many blocks are not heard
	nearEffect       = 2
	effectSteps      = 3 // walls on this many steps toward the sound halve it
	volumeShift      = 4 // the distance's volume is in 16ths
	repeatEffect     = 1 // effect 1 does not repeat itself (C: g_1eaa)
	alternateEffect  = 168
	alternateFor     = 167
	noEffect         = -1
)

// Driver volumes of the slider values (C: o2d56, fn_067a_01d1).
func musicDriverVolume(v int) int  { return v*musicVolumeScale/percent + musicVolumeBase }
func effectDriverVolume(v int) int { return v*effectVolumeScale/percent + effectVolumeBase }

// soundState is s003's (C: g_1e90..g_1eaa, g_2146..g_215c).
type soundState struct {
	file, song      int  // the sound file loaded (C: g_1e90, g_1e91)
	musicPlaying    bool // C: g_1e92
	pending         int  // the effect waiting to be played (C: g_1e96)
	pendingVolume   int  // C: g_1e98
	lastEffect      int  // C: g_1eaa
	delayed         []delayedSound
	delayedOverflow bool
}

type delayedSound struct{ sound, block int }

// initSound sets s003's starting values.
func (g *Game) initSound() {
	g.snd = soundState{file: noSoundFile, song: noSoundFile, lastEffect: noEffect}
	g.musicTrack = -1
	g.lastSpeech = notSpeaking
	g.audio = sound.New()
}

// StartAudio opens the audio stream; tests and scripted runs leave it closed.
func (g *Game) StartAudio() { g.audio.Start() }

// readSoundFile reads a sound file of the slot, empty when it is missing or too big for
// what is left of the buffer (C: ovr_2db9_0039).
func (g *Game) readSoundFile(name string, room int) []byte {
	b, err := g.Files.Read(name + soundFileExt)
	if err != nil || len(b)+len(name)+len(soundFileExt)+1 > room {
		return nil
	}
	return b
}

// stopSounds unloads the sound files (C: fn_067a_0003).
func (g *Game) stopSounds() {
	g.stopSoundEffects()
	g.audio.Load(sound.Effects, nil)
	g.stopSong()
	g.audio.Load(sound.Music, nil)
	g.snd.file, g.snd.song = noSoundFile, noSoundFile
}

// queueTrack loads the sound file of music track n, or LORESFX with the music off
// (C: fn_067a_003e).
func (g *Game) queueTrack(n int) {
	if g.options&optionMusic == 0 {
		if g.snd.file == effectsLoaded {
			return
		}
		g.unloadSoundFiles()
		g.loadSoundFiles(effectsFile, false)
		g.snd.file = effectsLoaded
		g.snd.song = int(musicTracks[n-firstMusicTrack].song) // fix: the original indexed by n itself
		return
	}
	t := musicTracks[n-firstMusicTrack]
	if t.file == g.snd.file && int(t.song) == g.snd.song {
		g.stopMusic()
		return
	}
	g.stopMusic()
	g.unloadSoundFiles()
	g.loadSoundFiles(fmt.Sprintf("LORE%02d%c", t.file, t.song), true)
	g.snd.file, g.snd.song = t.file, int(t.song)
}

func (g *Game) unloadSoundFiles() {
	g.stopSoundEffects()
	g.audio.Load(sound.Effects, nil)
	g.stopSong()
	g.audio.Load(sound.Music, nil)
}

// loadSoundFiles loads a file into the effects slot and, with music, the same file into
// the music slot, which shares it (C: fn_17f8_0003, fn_17c5_0004).
func (g *Game) loadSoundFiles(name string, music bool) {
	b := g.readSoundFile(name, soundBufferSize)
	g.audio.Load(sound.Effects, b)
	if music && soundBufferSize-len(b) > 1 {
		g.audio.Load(sound.Music, b)
	}
}

// playMusic starts music track n and returns the one before; -1 asks for the current
// one (C: fn_067a_01d1).
func (g *Game) playMusic(n int) int {
	if n == -1 {
		return g.musicTrack
	}
	old := g.musicTrack
	g.musicTrack = n
	if g.options&optionMusic != 0 {
		g.queueTrack(n)
		g.stopSong()
		g.snd.musicPlaying = true
		g.playSong(musicTracks[n-firstMusicTrack].track)
		g.setMusicVolume(musicDriverVolume(g.musicVolume))
	}
	return old
}

// stopMusic fades the music out (C: fn_067a_023d).
func (g *Game) stopMusic() {
	if g.options&optionMusic != 0 {
		if g.snd.musicPlaying {
			g.fadeSong()
			clock.Delay(fadePause)
		}
		g.snd.musicPlaying = false
	}
}

// playSoundQueued plays an effect from a block, or with delaySounds set keeps it for
// playDelayedSounds (C: fn_067a_027f).
func (g *Game) playSoundQueued(n, block int) {
	if !g.delaySounds {
		g.playSoundAt(n, block)
		return
	}
	if len(g.snd.delayed) < maxDelayedSounds {
		g.snd.delayed = append(g.snd.delayed, delayedSound{n, block})
		return
	}
	// the original printed "OVERFLOWED QUEUED DELAY Sound Effect" to its debug output
}

// playDelayedSounds plays the effects held back while delaySounds was set; the
// list is never emptied (sic), so earlier ones play again (C: fn_067a_02c7).
func (g *Game) playDelayedSounds() {
	for _, d := range g.snd.delayed {
		g.playSoundAt(d.sound, d.block)
	}
}

// playSoundAt plays the effect waiting, then makes effect n the one waiting, its volume
// from the distance to block and the walls on the way; block 0 is the party's. Unless
// the scene is to be drawn, it is played at once (C: fn_067a_02f5).
func (g *Game) playSoundAt(n, block int) {
	if g.options&optionSoundEffects == 0 {
		return
	}
	if g.snd.pending > 0 {
		g.playSoundVolume(g.snd.pending, g.snd.pendingVolume, false)
	}
	d := 0 // sic: the original used the distance unset for block 0
	if block != 0 {
		d = blockDistance(g.currentBlock, block)
		if d > farEffect {
			g.snd.pending = 0
			return
		}
	}
	g.snd.pending = n
	g.snd.pendingVolume = (farEffect - d) << volumeShift
	if g.currentBlock != block {
		b := g.currentBlock
		for steps := effectSteps; b != block && steps > 0; steps-- {
			dir := directionTo(b&columnMask, b>>5, block&columnMask, block>>5)
			b += blockSteps8[dir]
			if b != block && g.wallsBlock(b, 0, 1) {
				g.snd.pendingVolume >>= 1
			}
		}
	}
	if n != 0 && !g.sceneUpdateRequired {
		g.playSoundAt(0, 0)
	}
}

// blockSteps8 are the block offsets of the 8 directions (C: g_1e9a).
var blockSteps8 = [8]int{-32, -31, 1, 33, 32, 31, -1, -33}

// playSound plays effect n at full volume; force plays it with the effects off
// (C: fn_067a_0401).
func (g *Game) playSound(n, force int) { g.playSoundVolume(n, fullVolume, force != 0) }

// playSoundVolume plays effect n: its sample through the digital mixer if it has one
// and a channel is free, else its AdLib program (C: fn_067a_0417).
func (g *Game) playSoundVolume(n, vol int, force bool) {
	if !force && g.options&optionSoundEffects == 0 {
		return
	}
	if n == repeatEffect && (g.snd.lastEffect == noEffect || g.snd.lastEffect == repeatEffect) {
		return
	}
	g.snd.lastEffect = n
	e := soundEffects[n]
	priority := -e.volume
	if e.volume > 0 {
		priority = e.volume * vol >> 8
	}
	if g.playDigitalEffect(n, priority, vol) {
		return
	}
	if e.volume < 0 {
		vol = fullVolume
	}
	if n == alternateEffect {
		n = alternateFor
	}
	g.playAdlibEffect(n, vol)
}

// playDigitalEffect plays effect n's sample through the mixer; false if no channel would
// take it or it has no sample (C: fn_16cf_01c8, fn_067a_0649, fn_16cf_0022).
func (g *Game) playDigitalEffect(n, priority, vol int) bool {
	if !g.audio.CanPlayDigital(priority) {
		return false
	}
	s := g.effectSample(n)
	if s == nil {
		return false
	}
	v := effectDriverVolume(g.effectsVolume)
	if vol != fullVolume {
		v = v * vol >> 8
	}
	if g.digitalEffectsOn || g.digitalSpeechOn {
		g.audio.PlayDigital(s, priority, v)
	}
	return true
}

// effectSample is effect n's sample, loaded on first use from "<name>.VOC"; names that
// start with X or EM have none (C: fn_067a_0649). fix: the original kept the samples in
// a 10000+ byte cache that dropped the oldest; these stay loaded.
func (g *Game) effectSample(n int) *sound.Sample {
	if n < 0 || n >= maxSoundEffect || g.Config.Speech == 0 {
		return nil
	}
	v := soundEffects[n].voc
	if v == noEffect {
		return nil
	}
	if s, ok := g.samples[n]; ok {
		return s
	}
	var s *sound.Sample
	name := g.vocNames[v]
	if name != "" && name[0] != 'X' && !strings.HasPrefix(name, "EM") {
		if b, err := g.Files.Read(name + ".VOC"); err == nil {
			s = sound.ConvertVOC(b)
		}
	}
	if g.samples == nil {
		g.samples = map[int]*sound.Sample{}
	}
	g.samples[n] = s
	return s
}

// playSoundEffect plays effect n through the effects driver at full volume
// (C: fn_17d6_0006).
func (g *Game) playSoundEffect(n int) { g.playAdlibEffect(n, fullVolume) }

// playAdlibEffect plays effect n's program at a volume (C: fn_17d6_0019 with the
// AdLib's effects driver).
func (g *Game) playAdlibEffect(n, vol int) {
	if n < 0 || n >= maxSoundEffect || (!g.effectsOn && n > adlibOnlyCount) {
		return
	}
	g.audio.Play(sound.Effects, n, vol)
}

// stopSoundEffects stops the effects: the AdLib plays its stop track (C: fn_17d6_0123).
func (g *Game) stopSoundEffects() {
	if !g.audio.Loaded(sound.Effects) {
		return
	}
	if g.audio.Play(sound.Effects, stopTrack, silentVolume) {
		clock.Delay(stopPause)
	}
}

// playSong starts music track n (C: fn_178d_0091, fn_178d_00a4).
func (g *Game) playSong(n int) {
	if n < 0 || n > maxSoundEffect || !g.musicOn {
		return
	}
	if g.audio.Play(sound.Music, n, fullVolume) {
		clock.Delay(stopPause)
	}
}

// stopSong stops the music with its stop track (C: fn_178d_015e).
func (g *Game) stopSong() {
	if g.audio.Play(sound.Music, stopTrack, silentVolume) {
		clock.Delay(stopPause)
	}
}

// fadeSong fades the music out with its fade track (C: fn_178d_01b0).
func (g *Game) fadeSong() { g.audio.Play(sound.Music, fadeTrack, silentVolume) }

// songPlaying says whether the music plays (C: fn_178d_0038).
func (g *Game) songPlaying() bool { return g.audio.Playing() }

// setMusicVolume sets the driver's music volume; 0 also keeps the control programs from
// starting (C: ovr_2db9_0048, fn_16bf_0058).
func (g *Game) setMusicVolume(v int) {
	v = max(0, min(v, fullVolume))
	g.audio.SetMusicVolume(v)
	g.setMusicAllowed(v != 0)
}

// setMusicAllowed lets the music play or stops it (C: fn_16bf_0058).
func (g *Game) setMusicAllowed(on bool) {
	g.musicOn = on
	if !on {
		g.stopSong()
	}
	g.audio.Block(false, !on)
}

// setEffectVolume sets the driver's effects volume; 0 also stops the effects
// (C: ovr_2db9_002a, fn_16bf_0001).
func (g *Game) setEffectVolume(v int) {
	v = max(0, min(v, fullVolume))
	g.audio.SetEffectVolume(v)
	g.effectsOn = v != 0
	if v == 0 {
		g.stopSoundEffects()
	}
	g.audio.Block(true, v == 0)
}

// loadSoundFile names the sample file of effect n for the level (C: fn_067a_084d).
func (g *Game) loadSoundFile(name string, n int) {
	if name == "" || n < 0 || n > maxSoundEffect {
		return
	}
	if v := soundEffects[n].voc; v != noEffect {
		g.vocNames[v] = name
		delete(g.samples, n) // fix: the cache kept the old sample until it was dropped
	}
}

// soundLoaded says whether effect n has a sample (C: fn_067a_0649).
func (g *Game) soundLoaded(n int) bool { return g.effectSample(n) != nil }

// nextSoundEffect is the effect after n that has a sample, skipping effect 47; it goes
// back to 11 at the end of the table (C: fn_067a_08c2).
func (g *Game) nextSoundEffect(n int) int {
	for {
		n++
		if n == skippedTestEffect {
			n++
		}
		v := soundEffects[n].voc
		if v == endOfEffects {
			return firstTestEffect
		}
		if v < 0 || v >= len(g.vocNames) {
			continue // fix: the original read the name before the table
		}
		if s := g.vocNames[v]; s != "" && s[0] != 'X' && s != "EMPTY" {
			return n
		}
	}
}

const (
	skippedTestEffect = 47
	endOfEffects      = 199
	firstTestEffect   = 11
)
