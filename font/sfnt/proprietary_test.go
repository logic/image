// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sfnt

/*
This file contains opt-in tests for popular, high quality, proprietary fonts,
made by companies such as Adobe and Microsoft. These fonts are generally
available, but copies are not explicitly included in this repository due to
licensing differences or file size concerns. To opt-in, run:

go test golang.org/x/image/font/sfnt -args -proprietary

Not all tests pass out-of-the-box on all systems. For example, the Microsoft
Times New Roman font is downloadable gratis even on non-Windows systems, but as
per the ttf-mscorefonts-installer Debian package, this requires accepting an
End User License Agreement (EULA) and a CAB format decoder. These tests assume
that such fonts have already been installed. You may need to specify the
directories for these fonts:

go test golang.org/x/image/font/sfnt -args -proprietary -adobeDir=/foo/bar/aFonts -microsoftDir=/foo/bar/mFonts

To only run those tests for the Microsoft fonts:

go test golang.org/x/image/font/sfnt -test.run=ProprietaryMicrosoft -args -proprietary
*/

// TODO: add Apple system fonts? Google fonts (Droid? Noto?)? Emoji fonts?

// TODO: enable Apple/Microsoft tests by default on Darwin/Windows?

import (
	"errors"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var (
	proprietary = flag.Bool("proprietary", false, "test proprietary fonts not included in this repository")

	adobeDir = flag.String(
		"adobeDir",
		// This needs to be set explicitly. There is no default dir on Debian:
		// https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=736680
		//
		// Get the fonts from https://github.com/adobe-fonts, e.g.:
		//	- https://github.com/adobe-fonts/source-code-pro/releases/latest
		//	- https://github.com/adobe-fonts/source-han-sans/releases/latest
		//	- https://github.com/adobe-fonts/source-sans-pro/releases/latest
		//
		// Copy all of the TTF and OTF files to the one directory, such as
		// $HOME/adobe-fonts, and pass that as the -adobeDir flag here.
		"",
		"directory name for the Adobe proprietary fonts",
	)

	microsoftDir = flag.String(
		"microsoftDir",
		"/usr/share/fonts/truetype/msttcorefonts",
		"directory name for the Microsoft proprietary fonts",
	)
)

func TestProprietaryAdobeSourceCodeProOTF(t *testing.T) {
	testProprietary(t, "adobe", "SourceCodePro-Regular.otf", 1500, 2)
}

func TestProprietaryAdobeSourceCodeProTTF(t *testing.T) {
	testProprietary(t, "adobe", "SourceCodePro-Regular.ttf", 1500, 36)
}

func TestProprietaryAdobeSourceHanSansSC(t *testing.T) {
	testProprietary(t, "adobe", "SourceHanSansSC-Regular.otf", 65535, 2)
}

func TestProprietaryAdobeSourceSansProOTF(t *testing.T) {
	testProprietary(t, "adobe", "SourceSansPro-Regular.otf", 1800, 2)
}

func TestProprietaryAdobeSourceSansProTTF(t *testing.T) {
	testProprietary(t, "adobe", "SourceSansPro-Regular.ttf", 1800, 54)
}

func TestProprietaryMicrosoftArial(t *testing.T) {
	testProprietary(t, "microsoft", "Arial.ttf", 1200, 98)
}

func TestProprietaryMicrosoftComicSansMS(t *testing.T) {
	testProprietary(t, "microsoft", "Comic_Sans_MS.ttf", 550, 98)
}

func TestProprietaryMicrosoftTimesNewRoman(t *testing.T) {
	testProprietary(t, "microsoft", "Times_New_Roman.ttf", 1200, 98)
}

func TestProprietaryMicrosoftWebdings(t *testing.T) {
	testProprietary(t, "microsoft", "Webdings.ttf", 200, -1)
}

// testProprietary tests that we can load every glyph in the named font.
//
// The exact number of glyphs in the font can differ across its various
// versions, but as a sanity check, there should be at least minNumGlyphs.
//
// While this package is a work-in-progress, not every glyph can be loaded. The
// firstUnsupportedGlyph argument, if non-negative, is the index of the first
// unsupported glyph in the font. This number should increase over time (or set
// negative), as the TODO's in this package are done.
func testProprietary(t *testing.T, proprietor, filename string, minNumGlyphs, firstUnsupportedGlyph int) {
	if !*proprietary {
		t.Skip("skipping proprietary font test")
	}

	file, err := []byte(nil), error(nil)
	switch proprietor {
	case "adobe":
		file, err = ioutil.ReadFile(filepath.Join(*adobeDir, filename))
		if err != nil {
			t.Fatalf("%v\nPerhaps you need to set the -adobeDir=%v flag?", err, *adobeDir)
		}
	case "microsoft":
		file, err = ioutil.ReadFile(filepath.Join(*microsoftDir, filename))
		if err != nil {
			t.Fatalf("%v\nPerhaps you need to set the -microsoftDir=%v flag?", err, *microsoftDir)
		}
	default:
		panic("unreachable")
	}
	f, err := Parse(file)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ppem := fixed.Int26_6(f.UnitsPerEm())
	qualifiedFilename := proprietor + "/" + filename
	var buf Buffer

	// Some of the tests below, such as which glyph index a particular rune
	// maps to, can depend on the specific version of the proprietary font. If
	// tested against a different version of that font, the test might (but not
	// necessarily will) fail, even though the Go code is good. If so, log a
	// message, but don't automatically fail (i.e. dont' call t.Fatalf).
	gotVersion, err := f.Name(&buf, NameIDVersion)
	if err != nil {
		t.Fatalf("Name: %v", err)
	}
	wantVersion := proprietaryVersions[qualifiedFilename]
	if gotVersion != wantVersion {
		t.Logf("font version provided differs from the one the tests were written against:"+
			"\ngot  %q\nwant %q", gotVersion, wantVersion)
	}

	numGlyphs := f.NumGlyphs()
	if numGlyphs < minNumGlyphs {
		t.Fatalf("NumGlyphs: got %d, want at least %d", numGlyphs, minNumGlyphs)
	}

	iMax := numGlyphs
	if firstUnsupportedGlyph >= 0 {
		iMax = firstUnsupportedGlyph
	}
	for i, numErrors := 0, 0; i < iMax; i++ {
		if _, err := f.LoadGlyph(&buf, GlyphIndex(i), ppem, nil); err != nil {
			t.Errorf("LoadGlyph(%d): %v", i, err)
			numErrors++
		}
		if numErrors == 10 {
			t.Fatal("LoadGlyph: too many errors")
		}
	}

	for r, want := range proprietaryGlyphIndexTestCases[qualifiedFilename] {
		got, err := f.GlyphIndex(&buf, r)
		if err != nil {
			t.Errorf("GlyphIndex(%q): %v", r, err)
			continue
		}
		if got != want {
			t.Errorf("GlyphIndex(%q): got %d, want %d", r, got, want)
			continue
		}
	}

kernLoop:
	for _, tc := range proprietaryKernTestCases[qualifiedFilename] {
		var indexes [2]GlyphIndex
		for i := range indexes {
			x, err := f.GlyphIndex(&buf, tc.runes[i])
			if x == 0 && err == nil {
				err = errors.New("no glyph index found")
			}
			if err != nil {
				t.Errorf("GlyphIndex(%q): %v", tc.runes[0], err)
				continue kernLoop
			}
			indexes[i] = x
		}
		kern, err := f.Kern(&buf, indexes[0], indexes[1], tc.ppem, tc.hinting)
		if err != nil {
			t.Errorf("Kern(%q, %q, ppem=%d, hinting=%v): %v",
				tc.runes[0], tc.runes[1], tc.ppem, tc.hinting, err)
			continue
		}
		if got := Units(kern); got != tc.want {
			t.Errorf("Kern(%q, %q, ppem=%d, hinting=%v): got %d, want %d",
				tc.runes[0], tc.runes[1], tc.ppem, tc.hinting, got, tc.want)
			continue
		}
	}
}

// proprietaryVersions holds the expected version string of each proprietary
// font tested. If third parties such as Adobe or Microsoft update their fonts,
// and the tests subsequently fail, these versions should be updated too.
//
// Updates are expected to be infrequent. For example, as of 2017, the fonts
// installed by the Debian ttf-mscorefonts-installer package have last modified
// times no later than 2001.
var proprietaryVersions = map[string]string{
	"adobe/SourceCodePro-Regular.otf":   "Version 2.030;PS 1.0;hotconv 16.6.51;makeotf.lib2.5.65220",
	"adobe/SourceCodePro-Regular.ttf":   "Version 2.030;PS 1.000;hotconv 16.6.51;makeotf.lib2.5.65220",
	"adobe/SourceHanSansSC-Regular.otf": "Version 1.004;PS 1.004;hotconv 1.0.82;makeotf.lib2.5.63406",
	"adobe/SourceSansPro-Regular.otf":   "Version 2.020;PS 2.0;hotconv 1.0.86;makeotf.lib2.5.63406",
	"adobe/SourceSansPro-Regular.ttf":   "Version 2.020;PS 2.000;hotconv 1.0.86;makeotf.lib2.5.63406",

	"microsoft/Arial.ttf":           "Version 2.82",
	"microsoft/Comic_Sans_MS.ttf":   "Version 2.10",
	"microsoft/Times_New_Roman.ttf": "Version 2.82",
	"microsoft/Webdings.ttf":        "Version 1.03",
}

// proprietaryGlyphIndexTestCases hold a sample of each font's rune to glyph
// index cmap. The numerical values can be verified by running the ttx tool.
var proprietaryGlyphIndexTestCases = map[string]map[rune]GlyphIndex{
	"adobe/SourceCodePro-Regular.otf": {
		'\u0030':     877,  // U+0030 DIGIT ZERO
		'\u0041':     2,    // U+0041 LATIN CAPITAL LETTER A
		'\u0061':     28,   // U+0061 LATIN SMALL LETTER A
		'\u0104':     64,   // U+0104 LATIN CAPITAL LETTER A WITH OGONEK
		'\u0125':     323,  // U+0125 LATIN SMALL LETTER H WITH CIRCUMFLEX
		'\u01f4':     111,  // U+01F4 LATIN CAPITAL LETTER G WITH ACUTE
		'\u03a3':     623,  // U+03A3 GREEK CAPITAL LETTER SIGMA
		'\u2569':     1500, // U+2569 BOX DRAWINGS DOUBLE UP AND HORIZONTAL
		'\U0001f100': 0,    // U+0001F100 DIGIT ZERO FULL STOP
	},
	"adobe/SourceCodePro-Regular.ttf": {
		'\u0030': 877, // U+0030 DIGIT ZERO
		'\u0041': 2,   // U+0041 LATIN CAPITAL LETTER A
		'\u01f4': 111, // U+01F4 LATIN CAPITAL LETTER G WITH ACUTE
	},
	"adobe/SourceHanSansSC-Regular.otf": {
		'\u0030':     17,    // U+0030 DIGIT ZERO
		'\u0041':     34,    // U+0041 LATIN CAPITAL LETTER A
		'\u00d7':     150,   // U+00D7 MULTIPLICATION SIGN
		'\u1100':     365,   // U+1100 HANGUL CHOSEONG KIYEOK
		'\u25ca':     1254,  // U+25CA LOZENGE
		'\u2e9c':     1359,  // U+2E9C CJK RADICAL SUN
		'\u304b':     1463,  // U+304B HIRAGANA LETTER KA
		'\u4e2d':     9893,  // U+4E2D <CJK Ideograph>, 中
		'\ua960':     47537, // U+A960 HANGUL CHOSEONG TIKEUT-MIEUM
		'\ufb00':     58919, // U+FB00 LATIN SMALL LIGATURE FF
		'\uffee':     59213, // U+FFEE HALFWIDTH WHITE CIRCLE
		'\U0001f100': 59214, // U+0001F100 DIGIT ZERO FULL STOP
		'\U0001f248': 59449, // U+0001F248 TORTOISE SHELL BRACKETED CJK UNIFIED IDEOGRAPH-6557
		'\U0002f9f4': 61768, // U+0002F9F4 CJK COMPATIBILITY IDEOGRAPH-2F9F4
	},
	"adobe/SourceSansPro-Regular.otf": {
		'\u0041': 2,    // U+0041 LATIN CAPITAL LETTER A
		'\u03a3': 592,  // U+03A3 GREEK CAPITAL LETTER SIGMA
		'\u0435': 999,  // U+0435 CYRILLIC SMALL LETTER IE
		'\u2030': 1728, // U+2030 PER MILLE SIGN
	},
	"adobe/SourceSansPro-Regular.ttf": {
		'\u0041': 2,    // U+0041 LATIN CAPITAL LETTER A
		'\u03a3': 592,  // U+03A3 GREEK CAPITAL LETTER SIGMA
		'\u0435': 999,  // U+0435 CYRILLIC SMALL LETTER IE
		'\u2030': 1728, // U+2030 PER MILLE SIGN
	},

	"microsoft/Arial.ttf": {
		'\u0041':     36,   // U+0041 LATIN CAPITAL LETTER A
		'\u00f1':     120,  // U+00F1 LATIN SMALL LETTER N WITH TILDE
		'\u0401':     556,  // U+0401 CYRILLIC CAPITAL LETTER IO
		'\u200d':     745,  // U+200D ZERO WIDTH JOINER
		'\u20ab':     1150, // U+20AB DONG SIGN
		'\u2229':     320,  // U+2229 INTERSECTION
		'\u04e9':     1319, // U+04E9 CYRILLIC SMALL LETTER BARRED O
		'\U0001f100': 0,    // U+0001F100 DIGIT ZERO FULL STOP
	},
	"microsoft/Comic_Sans_MS.ttf": {
		'\u0041': 36,  // U+0041 LATIN CAPITAL LETTER A
		'\u03af': 573, // U+03AF GREEK SMALL LETTER IOTA WITH TONOS
	},
	"microsoft/Times_New_Roman.ttf": {
		'\u0041': 36,  // U+0041 LATIN CAPITAL LETTER A
		'\u0042': 37,  // U+0041 LATIN CAPITAL LETTER B
		'\u266a': 392, // U+266A EIGHTH NOTE
		'\uf041': 0,   // PRIVATE USE AREA
		'\uf042': 0,   // PRIVATE USE AREA
	},
	"microsoft/Webdings.ttf": {
		'\u0041': 0,  // U+0041 LATIN CAPITAL LETTER A
		'\u0042': 0,  // U+0041 LATIN CAPITAL LETTER B
		'\u266a': 0,  // U+266A EIGHTH NOTE
		'\uf041': 36, // PRIVATE USE AREA
		'\uf042': 37, // PRIVATE USE AREA
	},
}

type kernTestCase struct {
	ppem    fixed.Int26_6
	hinting font.Hinting
	runes   [2]rune
	want    Units
}

// proprietaryKernTestCases hold a sample of each font's kerning pairs. The
// numerical values can be verified by running the ttx tool.
var proprietaryKernTestCases = map[string][]kernTestCase{
	"microsoft/Arial.ttf": {
		{2048, font.HintingNone, [2]rune{'A', 'V'}, -152},
		// U+03B8 GREEK SMALL LETTER THETA
		// U+03BB GREEK SMALL LETTER LAMDA
		{2048, font.HintingNone, [2]rune{'\u03b8', '\u03bb'}, -39},
		{2048, font.HintingNone, [2]rune{'\u03bb', '\u03b8'}, -0},
	},
	"microsoft/Comic_Sans_MS.ttf": {
		{2048, font.HintingNone, [2]rune{'A', 'V'}, 0},
	},
	"microsoft/Times_New_Roman.ttf": {
		{768, font.HintingNone, [2]rune{'A', 'V'}, -99},
		{768, font.HintingFull, [2]rune{'A', 'V'}, -128},
		{2048, font.HintingNone, [2]rune{'A', 'A'}, 0},
		{2048, font.HintingNone, [2]rune{'A', 'T'}, -227},
		{2048, font.HintingNone, [2]rune{'A', 'V'}, -264},
		{2048, font.HintingNone, [2]rune{'T', 'A'}, -164},
		{2048, font.HintingNone, [2]rune{'T', 'T'}, 0},
		{2048, font.HintingNone, [2]rune{'T', 'V'}, 0},
		{2048, font.HintingNone, [2]rune{'V', 'A'}, -264},
		{2048, font.HintingNone, [2]rune{'V', 'T'}, 0},
		{2048, font.HintingNone, [2]rune{'V', 'V'}, 0},
		// U+0390 GREEK SMALL LETTER IOTA WITH DIALYTIKA AND TONOS
		// U+0393 GREEK CAPITAL LETTER GAMMA
		{2048, font.HintingNone, [2]rune{'\u0390', '\u0393'}, 0},
		{2048, font.HintingNone, [2]rune{'\u0393', '\u0390'}, 76},
	},
	"microsoft/Webdings.ttf": {
		{2048, font.HintingNone, [2]rune{'\uf041', '\uf042'}, 0},
	},
}
