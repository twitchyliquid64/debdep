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
