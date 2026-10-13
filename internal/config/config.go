// Package config reads and writes LANDS.CFG, the set-up program's choices (o2e1e).
package config

// The file is 10 bytes: 8 obfuscated settings, the language, and a checksum of the 8
// settings. Each of the 8 has the number of bytes after it among them added, then is
// XORed with obfuscationKey; the checksum is XORed with it too.
const (
	FileName       = "LANDS.CFG"
	fileSize       = 10
	obfuscated     = 8
	languageAt     = 8
	checksumAt     = 9
	obfuscationKey = 165
)

// Video modes the game accepts (C: main's switch on g_ad41); all run as MCGA.
var videoModes = []int{7, 8, 10, 11, 12}

// Config is LANDS.CFG (C: g_ad41..g_ad49).
type Config struct {
	Video        int  // C: g_ad41
	Music        int  // music driver (C: g_ad42)
	Effects      int  // sound effects driver (C: g_ad43)
	Speech       int  // C: g_ad44
	Mouse        bool // C: g_ad45; without one the keypad moves the cursor
	DigitalSound bool // C: g_ad46; speech is switched off without it
	reserved     [2]byte
	Language     int // 0 English, 1 French, 2 German (C: g_ad49)
}

// Decode checks and unpacks the file's contents; ok is false if it is the wrong size or
// the checksum fails (C: ovr_2e1e_0025, ovr_2e1e_002f).
func Decode(b []byte) (c Config, ok bool) {
	if len(b) != fileSize {
		return c, false
	}
	var plain [fileSize]byte
	copy(plain[:], b)
	sum := byte(0)
	for k := range obfuscated {
		plain[k] = (plain[k] ^ obfuscationKey) - byte(obfuscated-1-k)
		sum += plain[k]
	}
	if plain[checksumAt] != sum^obfuscationKey {
		return c, false
	}
	return Config{
		Video:        int(plain[0]),
		Music:        int(plain[1]),
		Effects:      int(plain[2]),
		Speech:       int(plain[3]),
		Mouse:        plain[4] != 0,
		DigitalSound: plain[5] != 0,
		reserved:     [2]byte{plain[6], plain[7]},
		Language:     int(plain[languageAt]),
	}, true
}

// Encode packs a configuration the way Decode reads it (C: ovr_2e1e_002a).
func (c Config) Encode() []byte {
	b := []byte{byte(c.Video), byte(c.Music), byte(c.Effects), byte(c.Speech),
		boolByte(c.Mouse), boolByte(c.DigitalSound), c.reserved[0], c.reserved[1],
		byte(c.Language), 0}
	sum := byte(0)
	for k := range obfuscated {
		sum += b[k]
		b[k] = (b[k] + byte(obfuscated-1-k)) ^ obfuscationKey
	}
	b[checksumAt] = sum ^ obfuscationKey
	return b
}

// VideoSupported says whether main accepts the video mode.
func (c Config) VideoSupported() bool {
	for _, m := range videoModes {
		if c.Video == m {
			return true
		}
	}
	return false
}

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}
