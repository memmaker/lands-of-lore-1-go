package input

// scanKeys turns a scan code into a key number (C: cs:1c3h).
var scanKeys = [128]int{
	127, 110, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 16,
	17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 43, 58, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 1, 44, 29, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 57, 100, 60, 61, 30, 112, 113, 114, 115, 116,
	117, 118, 119, 120, 121, 90, 125, 91, 96, 101, 105, 92, 97, 102, 106, 93,
	98, 103, 99, 104, 127, 127, 127, 122, 123,
}

// extendedScans and extendedKeys: the 0xE0-prefixed scan codes and their key numbers
// (C: cs:88h, cs:98h).
var extendedScans = [...]int{56, 29, 82, 83, 75, 71, 79, 72, 80, 73, 81, 77, 53, 28, 55, 70}
var extendedKeys = [...]int{62, 64, 75, 76, 79, 80, 81, 83, 84, 85, 86, 89, 95, 108, 124, 0}

// grey and twins: keys remap gives their twins' numbers (C: cs:98h, cs:0a8h).
var grey = extendedKeys
var twins = [...]int{60, 58, 99, 104, 92, 91, 93, 96, 98, 101, 103, 102, 55, 43, 124, 0}

// plainASCII and shiftedASCII: what the typing keys (below 62) type (C: cs:353h, cs:391h).
var plainASCII = [62]int{
	0, '`', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '=', 0, 8,
	9, 'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '[', ']', '\\', 0, 'a',
	's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', ';', '\'', 0, 13, 0, '-', 'z', 'x',
	'c', 'v', 'b', 'n', 'm', ',', '.', '/', 0, 0, 0, 0, 0, ' ',
}
var shiftedASCII = [62]int{
	0, '~', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '_', '+', 0, 8,
	9, 'Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P', '{', '}', '|', 0, 'A',
	'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L', ':', '"', 0, 13, 0, '-', 'Z', 'X',
	'C', 'V', 'B', 'N', 'M', '<', '>', '?', 0, 0, 0, 0, 0, ' ',
}

// navASCII: the codes of keys 75-109 with Num Lock off (C: cs:44bh).
var navASCII = [keyLastNav - keyFirstNav + 1]int{
	174, 173, 0, 0, 181, 185, 177, 0, 184, 176, 183, 175, 0, 0, 179, 0,
	185, 181, 177, 0, '/', 184, 180, 176, 174, '*', 183, 179, 175, 173, '-', '+',
	0, 13, 0,
}

// letterKeys: a bit per key number for the letters, which Ctrl turns into control
// codes (C: cs:21ch).
var letterKeys = [keyBits / bitsPerByte]byte{0, 0, 254, 135, 255, 192, 31}

// silentKeys: keys the character readers skip (C: cs:26ah; the last entry was never
// matched).
var silentKeys = [...]int{30, 44, 57, 58, 60, 62, 64, 90, 128, 128, 128}
