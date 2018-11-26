package deb

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

var multilineFields = map[string]bool{
	"Description":      true,
	"Files":            true,
	"Changes":          true,
	"Package-List":     true,
	"MD5Sum":           true,
	"Checksums-Sha1":   true,
	"SHA1":             true,
	"Checksums-Sha256": true,
	"SHA256":           true,
}

// Decoder is used to parse debian control files, and package lists.
type Decoder struct {
	r *bufio.Reader
}

// Decode reads the next paragraph of metadata from the reader.
func (d *Decoder) Decode(out *Paragraph) error {
	lastKey := ""
	lastMultiline := false

	for {
		orig, err := d.r.ReadString('\n')
		if err != nil {
			return err
		}
		line := strings.TrimSpace(orig)

		if line == "" {
			if !out.dirty {
				return io.EOF
			}
			return nil
		}

		if orig[0] == ' ' || orig[0] == '\t' {
			if lastMultiline {
				out.Values[lastKey] += line
				continue
			}
			// I dont think is is correct syntax but lots of packages seem to do this.
			out.Values[lastKey] += orig
		} else {
			out.dirty = true
			if out.Values == nil {
				out.Values = map[string]string{}
			}

			i := strings.Index(line, ":")
			if i < 0 {
				return fmt.Errorf("expected colon in line %q", line)
			}
			out.Values[line[:i]] = strings.TrimLeftFunc(line[i+2:], unicode.IsSpace)
			lastKey = line[:i]
			lastMultiline = multilineFields[lastKey]
		}
	}

	return nil
}

// NewDecoder returns a decoder for reading debian control files and
// other package metadata files.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: bufio.NewReader(r),
	}
}

func consumeWhitespace(r *bufio.Reader) error {
	for {
		inRune, _, err := r.ReadRune()
		if err != nil {
			return err
		}
		if unicode.IsSpace(inRune) {
			continue
		}
		return r.UnreadRune()
	}
}

func readPkgName(r *bufio.Reader) (string, error) {
	out := ""
	for {
		inRune, _, err := r.ReadRune()
		if err != nil {
			return out, err
		}
		if unicode.IsSpace(inRune) || inRune == ',' || inRune == '|' || inRune == '(' {
			return out, r.UnreadRune()
		}
		out += string(inRune)
	}
	return "", nil
}

// parseRelation parses a single package name & optional version constraint.
func parseRelation(r *bufio.Reader) (string, *VersionConstraint, error) {
	if err := consumeWhitespace(r); err != nil {
		return "", nil, err
	}
	name, err := readPkgName(r)
	if err != nil && err != io.EOF {
		return name, nil, err
	}
	consumeWhitespace(r)
	next, _, err := r.ReadRune()
	if err != nil && err != io.EOF {
		return name, nil, err
	}
	if next != '(' {
		r.UnreadRune()
		return name, nil, err
	}

	var versionConst VersionConstraint
	constraint, err := r.ReadString(' ')
	if err != nil {
		return name, nil, fmt.Errorf("error when expected ' ': %v", err)
	}
	constraint = strings.Trim(constraint, " ")
	switch constraint {
	case ConstraintGreaterThan, ConstraintLessThan, ConstraintEquals, ConstraintGreaterEquals, ConstraintLessThanEquals:
		versionConst.ConstraintRelation = ConstraintRelation(constraint)
	default:
		return name, nil, fmt.Errorf("expected relation, got %q", constraint)
	}

	vers, err := r.ReadString(')')
	if err != nil {
		return name, nil, fmt.Errorf("error when expected ')': %v", err)
	}
	versionConst.Version = strings.Trim(vers, ") ")
	return name, &versionConst, nil
}

// parseRelationSpec parses a group (between commas) of relation constraints.
func parseRelationSpec(r *bufio.Reader) (out Requirement, err error) {
	defer func() {
		if len(out.Children) == 1 && (out.Kind == AndCompositeRequirement || out.Kind == OrCompositeRequirement) {
			out = out.Children[0]
		}
	}()

	for {
		name, versionConst, err := parseRelation(r)
		if err != nil {
			if err != io.EOF || name == "" {
				return out, err
			}
		}
		spec := Requirement{
			Kind:    PackageRelationRequirement,
			Package: name,
		}
		if versionConst != nil {
			spec.VersionConstraint = versionConst
		}

		consumeWhitespace(r)
		next, _, err := r.ReadRune()
		if err != nil && err != io.EOF {
			return out, err
		}
		switch next {
		case '|':
			// This is a bunch of different packages where any one can satisfy
			// the requirement. We track this by setting out.Kind differently,
			// and instead of returning from this function we put the whole
			// lot into a OrCompositeRequirement.
			if out.Kind != OrCompositeRequirement {
				out = Requirement{
					Kind:     OrCompositeRequirement,
					Children: []Requirement{spec},
				}
			} else {
				out.Children = append(out.Children, spec)
			}

		default:
			fallthrough
		case ',':
			if out.Kind != OrCompositeRequirement {
				return spec, nil
			} else {
				out.Children = append(out.Children, spec)
				return out, nil
			}
		}
	}

	return out, nil
}

// ParsePackageRelations takes a string of package/version contraints, and parses
// them into a tree of Requirement structures.
func ParsePackageRelations(in string) (out Requirement, err error) {
	defer func() {
		if (out.Kind == AndCompositeRequirement || out.Kind == OrCompositeRequirement) && len(out.Children) == 1 {
			out = out.Children[0]
		}
	}()

	r := bufio.NewReader(strings.NewReader(in))
	out = Requirement{
		Kind: AndCompositeRequirement,
	}

	for {
		spec, err := parseRelationSpec(r)
		if err != nil {
			if err == io.EOF {
				// We track if there is one final requirement or not by
				// checking spec.Kind, as AndCompositeRequirement should never
				// happen (not possible ATM to have nested AND conditionals).
				// We should probably use a dirty flag, a nil pointer, or something else.
				if spec.Kind != AndCompositeRequirement {
					out.Children = append(out.Children, spec)
				}
				return out, nil
			}
			return out, err
		}
		out.Children = append(out.Children, spec)
	}

	return out, nil
}
