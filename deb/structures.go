package deb

import (
	"errors"

	version "github.com/knqyf263/go-deb-version"
)

// Paragraph represents the metadata associated with a debian package.
type Paragraph struct {
	dirty  bool
	Values map[string]string
}

// Name returns the package name, or the empty string.
func (p *Paragraph) Name() string {
	n, _ := p.Values["Package"]
	return n
}

// Version returns the parsed version of the package.
func (p *Paragraph) Version() (version.Version, error) {
	v, ok := p.Values["Version"]
	if !ok {
		return version.Version{}, errors.New("no version specified in package")
	}
	return version.NewVersion(v)
}

// BinaryDepends returns a requirements graph representing the binary
// dependencies of the package.
func (p *Paragraph) BinaryDepends() (Requirement, error) {
	dep, ok := p.Values["Depends"]
	if !ok {
		return Requirement{}, nil
	}
	return ParsePackageRelations(dep)
}

// RequirementKind disambiguates nodes in the requirements tree.
type RequirementKind uint8

// Kinds of requirements.
const (
	AndCompositeRequirement RequirementKind = iota
	OrCompositeRequirement
	PackageRelationRequirement
)

func (r RequirementKind) String() string {
	switch r {
	case AndCompositeRequirement:
		return "AND"
	case OrCompositeRequirement:
		return "OR"
	case PackageRelationRequirement:
		return "Relation constraint"
	}
	return "?"
}

// Requirement represents a tree of constraints on a package.
type Requirement struct {
	Kind     RequirementKind
	Children []Requirement

	// Applicable if Kind == PackageRelationRequirement
	Package           string
	VersionConstraint *VersionConstraint
}

func (r *Requirement) Equal(b *Requirement) bool {
	if r.Kind != b.Kind {
		return false
	}
	if r.Kind == PackageRelationRequirement {
		if r.Package != b.Package {
			return false
		}
		hasVers := r.VersionConstraint != nil
		if hasVers != (b.VersionConstraint != nil) {
			return false
		}
		if hasVers {
			if r.VersionConstraint.ConstraintRelation != b.VersionConstraint.ConstraintRelation {
				return false
			}
			if r.VersionConstraint.Version != b.VersionConstraint.Version {
				return false
			}
		}
	} else {
		if len(r.Children) != len(b.Children) {
			return false
		}
		for i, _ := range r.Children {
			if !r.Children[i].Equal(&b.Children[i]) {
				return false
			}
		}
	}
	return true
}

// ConstraintRelation describes the kind of operation
// by which a package version is constrained.
type ConstraintRelation string

// Kinds of version contraint relationships.
const (
	ConstraintGreaterThan    = ">>"
	ConstraintLessThan       = "<<"
	ConstraintEquals         = "="
	ConstraintGreaterEquals  = ">="
	ConstraintLessThanEquals = "<="
)

// VersionConstraint describes versioning constraints applied to
// package relation.
type VersionConstraint struct {
	ConstraintRelation ConstraintRelation
	Version            string
}
