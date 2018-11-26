package deb

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

var basicPkg = `Package: fonts-sil-abyssinica
Status: install ok installed
Priority: optional
Section: fonts
Installed-Size: 2208
Maintainer: Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>
Architecture: all
Multi-Arch: foreign
Version: 1.500-1
Replaces: ttf-sil-abyssinica (<< 1.200-1)
Pre-Depends: dpkg (>= 1.15.6~)
Suggests: fontconfig, libgraphite3, pango-graphite
Breaks: ttf-sil-abyssinica (<< 1.200-1)
Description: smart Unicode font for Ethiopian and Erythrean scripts (Amharic et al.)
 The Ethiopic script is used for writing many of the languages of Ethiopia
 and Eritrea. Ethiopic (U+1200..U+137F) was added to Unicode 3.0. Ethiopic
 Supplement (U+1380..U+139F) and Ethiopic Extended (U+2D80..U+2DDF) were
 added to Unicode 4.1. Abyssinica SIL supports all Ethiopic characters which
 are in Unicode including the Unicode 4.1 extensions. Some languages of
 Ethiopia are not yet able to be fully represented in Unicode and, where
 necessary, non-Unicode characters were included in the Private Use Area.
 .
 Please read the documentation to see what ranges are supported
 and for more about the various features of the font.
 .
 Abyssinica SIL is a TrueType font with "smart font" capabilities added using
 the Graphite, OpenType(r), and AAT font technologies. This means that
 complex typographic issues such as the placement of multiple diacritics or
 the formation of ligatures are handled by the font, provided you are
 running an application that provides an adequate level of support for one
 of these smart font technologies.
 .
 This release is a regular typeface, with no bold or italic version
 available or planned.
 .
 More font sources are available in the source package and on the
 project website. Webfont versions and examples are also available.
Original-Maintainer: Debian Fonts Task Force <pkg-fonts-devel@lists.alioth.debian.org>
Homepage: http://scripts.sil.org/AbyssinicaSIL

`

func TestBasic(t *testing.T) {
	decoder := NewDecoder(strings.NewReader(basicPkg))
	var p Paragraph
	if err := decoder.Decode(&p); err != nil {
		t.Fatalf("Decode() returned err: %v", err)
	}
	if p.Values["Homepage"] != "http://scripts.sil.org/AbyssinicaSIL" {
		t.Errorf("Homepage = %q, wanted %q", p.Values["Homepage"], "http://scripts.sil.org/AbyssinicaSIL")
	}
	if p.Values["Version"] != "1.500-1" {
		t.Errorf("Version = %q, wanted %q", p.Values["Version"], "1.500-1")
	}
	if len(p.Values["Description"]) != 1304 {
		t.Errorf("len(Description) = %d, wanted %d", len(p.Values["Description"]), 1304)
	}

	if err := decoder.Decode(&p); err != io.EOF {
		t.Fatalf("Decode() returned err: %v, wanted io.EOF", err)
	}
}

var longDepends = `libamd2 (>= 1:4.5.2), libavcodec58 | libavcodec-extra58, libavformat58, libavutil56, libblas3 | libblas.so.3, libbtf1 (>= 1:4.5.2), libc6 (>= 2.15), libccolamd2 (>= 1:4.5.2), libcholmod3 (>= 1:4.5.2), libcolamd2 (>= 1:4.5.2), libcxsparse3 (>= 1:4.5.2), libgcc1 (>= 1:4.0), libjpeg62-turbo (>= 1.3.1), libklu1 (>= 1:4.5.2), liblapack3 | liblapack.so.3, libldl2 (>= 1:4.5.2), libopencv-calib3d3.2, libopencv-contrib3.2, libopencv-core3.2, libopencv-features2d3.2, libopencv-flann3.2, libopencv-highgui3.2, libopencv-imgcodecs3.2, libopencv-imgproc3.2, libopencv-ml3.2, libopencv-objdetect3.2, libopencv-photo3.2, libopencv-shape3.2, libopencv-stitching3.2, libopencv-superres3.2, libopencv-video3.2, libopencv-videoio3.2, libopencv-videostab3.2, libopencv-viz3.2, libspqr2 (>= 1:5.2.0+dfsg), libstdc++6 (>= 5.2), libswscale5 (>= 7:4.0), libumfpack5 (>= 1:4.5.2), libwxbase3.0-0v5 (>= 3.0.4+dfsg), libwxgtk3.0-0v5 (>= 3.0.4+dfsg), zlib1g (>= 1:1.2.3.4)`

func TestComplexDepends(t *testing.T) {
	spec, err := ParsePackageRelations(longDepends)
	if err != nil {
		t.Fatalf("ParsePackageRelations() returned err: %v", err)
	}
	if len(spec.Children) != 41 {
		t.Errorf("Expected 41 children, got %d", len(spec.Children))
	}
}

func TestParseDependsSimple(t *testing.T) {
	spec, err := ParsePackageRelations("libamd2 , libavcodec58")
	if err != nil {
		t.Fatalf("ParsePackageRelations() returned err: %v", err)
	}
	if !reflect.DeepEqual(spec, Requirement{
		Kind: AndCompositeRequirement,
		Children: []Requirement{
			{
				Kind:    PackageRelationRequirement,
				Package: "libamd2",
			},
			{
				Kind:    PackageRelationRequirement,
				Package: "libavcodec58",
			},
		},
	}) {
		t.Errorf("Incorrect parse: %+v", spec)
	}
}

func TestParseDependsSimpleVersions(t *testing.T) {
	spec, err := ParsePackageRelations("libamd2 (>= 1:4.5.2), libc6 (>= 2.15)")
	if err != nil {
		t.Fatalf("ParsePackageRelations() returned err: %v", err)
	}

	if len(spec.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(spec.Children))
	}
	if spec.Children[0].VersionConstraint == nil || spec.Children[0].VersionConstraint.Version != "1:4.5.2" || spec.Children[0].VersionConstraint.ConstraintRelation != ConstraintGreaterEquals {
		t.Errorf("First version constraint incorrect: %v", spec.Children[0].VersionConstraint)
	}
	if spec.Children[1].VersionConstraint == nil || spec.Children[1].VersionConstraint.Version != "2.15" || spec.Children[1].VersionConstraint.ConstraintRelation != ConstraintGreaterEquals {
		t.Errorf("Second version constraint incorrect: %v", spec.Children[1].VersionConstraint)
	}
}

func TestParseDependsSimpleOrWithVersion(t *testing.T) {
	spec, err := ParsePackageRelations("libamd2 (>= 1:4.5.2) | libc6")
	if err != nil {
		t.Fatalf("ParsePackageRelations() returned err: %v", err)
	}

	if spec.Kind != OrCompositeRequirement || len(spec.Children) != 2 {
		t.Fatalf("Expected 2 children on OR root, got %v with len = %d", spec.Kind, len(spec.Children))
	}
	if spec.Children[0].VersionConstraint == nil || spec.Children[0].VersionConstraint.Version != "1:4.5.2" || spec.Children[0].VersionConstraint.ConstraintRelation != ConstraintGreaterEquals {
		t.Errorf("First version constraint incorrect: %v", spec.Children[0].VersionConstraint)
	}
	if spec.Children[1].VersionConstraint != nil || spec.Children[1].Package != "libc6" {
		t.Errorf("Second version constraint incorrect: %v", spec.Children[1].VersionConstraint)
	}
}

func TestParseDependsSimpleVersions2(t *testing.T) {
	spec, err := ParsePackageRelations("kek (<< 1.7), meep (= 1.3.2)")
	if err != nil {
		t.Fatalf("ParsePackageRelations() returned err: %v", err)
	}

	if len(spec.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(spec.Children))
	}
	if spec.Children[0].VersionConstraint == nil || spec.Children[0].VersionConstraint.Version != "1.7" || spec.Children[0].VersionConstraint.ConstraintRelation != ConstraintLessThan {
		t.Errorf("First version constraint incorrect: %v", spec.Children[0].VersionConstraint)
	}
	if spec.Children[1].VersionConstraint == nil || spec.Children[1].VersionConstraint.Version != "1.3.2" || spec.Children[1].VersionConstraint.ConstraintRelation != ConstraintEquals {
		t.Errorf("Second version constraint incorrect: %v", spec.Children[1].VersionConstraint)
	}
}

func TestParseDependsOrVersions(t *testing.T) {
	spec, err := ParsePackageRelations("libamd2 (= 1:4.5.2), libkek | libc6 (>= 2.15), bruv")
	if err != nil {
		t.Fatalf("ParsePackageRelations() returned err: %v", err)
	}

	if len(spec.Children) != 3 {
		t.Fatalf("Expected 3 children, got %d", len(spec.Children))
	}
	if spec.Children[0].VersionConstraint == nil || spec.Children[0].VersionConstraint.Version != "1:4.5.2" || spec.Children[0].VersionConstraint.ConstraintRelation != ConstraintEquals {
		t.Errorf("First version constraint incorrect: %v", spec.Children[0].VersionConstraint)
	}
	if spec.Children[1].Kind != OrCompositeRequirement {
		t.Errorf("Expected 2nd constraint to be type OrCompositeRequirement, got %v", spec.Children[1].Kind)
	}
	if len(spec.Children[1].Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(spec.Children[1].Children))
	}
	sub := spec.Children[1].Children
	if sub[0].Package != "libkek" || sub[1].Package != "libc6" {
		t.Errorf("Incorrect package names: %+v", sub)
	}
	if sub[0].VersionConstraint != nil {
		t.Errorf("First version constraint incorrect: %v", spec.Children[1])
	}
	if sub[1].VersionConstraint == nil || sub[1].VersionConstraint.Version != "2.15" || sub[1].VersionConstraint.ConstraintRelation != ConstraintGreaterEquals {
		t.Errorf("Second version constraint incorrect: %v", spec.Children[1])
	}

	if spec.Children[2].Package != "bruv" {
		t.Errorf("Last package wrong: %+v", spec.Children[2])
	}
}
