package debdep

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/twitchyliquid64/debdep/deb"

	version "github.com/knqyf263/go-deb-version"
)

type ErrDependency struct {
	DependencyPackage string
	RequiredByPackage string
	RequiredByVersion string
	VersionConstraint *deb.VersionConstraint
}

func (e ErrDependency) Error() string {
	base := fmt.Sprintf("package %q (%s) required %q", e.RequiredByPackage, e.RequiredByVersion, e.DependencyPackage)
	if e.RequiredByPackage == "" {
		base = fmt.Sprintf("required package %q", e.DependencyPackage)
	}

	if e.VersionConstraint == nil {
		return base + " was not found"
	} else {
		base += fmt.Sprintf(" with version %s %q, but it was not found", e.VersionConstraint.ConstraintRelation, e.VersionConstraint.Version)
		return base
	}
}

type OperationKind uint8

func (o OperationKind) String() string {
	switch o {
	case DebPackageInstallOp:
		return "package-dep"
	case CompositeDependencyOp:
		return "composite"
	}
	return "?OperationKind?"
}

// Different OperationKind's.
const (
	DebPackageInstallOp OperationKind = iota
	CompositeDependencyOp
)

// Operation represents an operation in a tree of dependencies/operations.
type Operation struct {
	Kind                OperationKind
	DependentOperations []*Operation

	Package string
	Version version.Version
}

func (o *Operation) PrettyWrite(w io.Writer, depth int) error {
	for i := 0; i < depth; i++ {
		w.Write([]byte(" "))
	}
	w.Write([]byte(o.Kind.String() + ": "))

	switch o.Kind {
	case CompositeDependencyOp:
		w.Write([]byte("\n"))
		for _, dep := range o.DependentOperations {
			dep.PrettyWrite(w, depth+1)
		}
	case DebPackageInstallOp:
		w.Write([]byte(o.Package + " (" + o.Version.String() + ")\n"))
	}

	return nil
}

func (p *PackageInfo) InstallGraph(target string, installed *PackageInfo) (*Operation, error) {
	var coveredDeps []deb.Requirement
	return p.buildInstallGraph(target, &coveredDeps, installed)
}

func (p *PackageInfo) buildInstallGraph(target string, coveredDeps *[]deb.Requirement, installed *PackageInfo) (*Operation, error) {
	pkg, err := p.FindLatest(target)
	if err != nil {
		return nil, err
	}
	vers, err := pkg.Version()
	if err != nil {
		return nil, err
	}
	deps, err := pkg.BinaryDepends()
	if err != nil {
		return nil, err
	}

	op, err := p.buildInstallGraphRequirement(coveredDeps, installed, deps, deb.Requirement{})
	if err != nil {
		return nil, err
	}
	return &Operation{
		Kind: CompositeDependencyOp,
		DependentOperations: []*Operation{
			op,
			&Operation{
				Kind:    DebPackageInstallOp,
				Package: pkg.Name(),
				Version: vers,
			},
		},
	}, nil
}

func (p *PackageInfo) buildInstallGraphRequirement(coveredDeps *[]deb.Requirement, installed *PackageInfo, req deb.Requirement, parent deb.Requirement) (out *Operation, err error) {
	defer func() {
		if err == nil && out.Kind == CompositeDependencyOp && len(out.DependentOperations) == 1 {
			out = out.DependentOperations[0]
		}
	}()

	if checkSetCoveredDependency(coveredDeps, req) {
		return &Operation{Kind: CompositeDependencyOp}, nil
	}

	switch req.Kind {
	case deb.AndCompositeRequirement:
		var ops []*Operation
		for _, dep := range req.Children {
			op, err := p.buildInstallGraphRequirement(coveredDeps, installed, dep, req)
			if err != nil {
				return nil, err
			}
			ops = append(ops, op)
		}
		return &Operation{
			Kind:                CompositeDependencyOp,
			DependentOperations: ops,
		}, nil

	case deb.PackageRelationRequirement:
		var selected *deb.Paragraph
		if req.VersionConstraint == nil {
			latest, err := p.FindLatest(req.Package)
			if err != nil {
				if err == os.ErrNotExist {
					return nil, ErrDependency{
						DependencyPackage: req.Package,
						RequiredByPackage: parent.Package,
					}
				}
				return nil, err
			}
			selected = latest
		} else {
			pkg, err := p.FindWithVersionConstraint(req.Package, req.VersionConstraint)
			if err != nil {
				if err == os.ErrNotExist {
					return nil, ErrDependency{
						DependencyPackage: req.Package,
						RequiredByPackage: parent.Package,
						VersionConstraint: req.VersionConstraint,
					}
				}
				return nil, err
			}
			selected = pkg
		}

		v, err := selected.Version()
		if err != nil {
			return nil, err
		}

		nextDeps, err := selected.BinaryDepends()
		if err != nil {
			return nil, err
		}

		nextOps, err := p.buildInstallGraphRequirement(coveredDeps, installed, nextDeps, req)
		if err != nil {
			return nil, err
		}

		if nextOps.Kind == CompositeDependencyOp && len(nextOps.DependentOperations) == 0 {
			return &Operation{
				Kind:    DebPackageInstallOp,
				Package: selected.Name(),
				Version: v,
			}, nil
		} else {
			return &Operation{
				Kind: CompositeDependencyOp,
				DependentOperations: []*Operation{
					nextOps,
					{
						Kind:    DebPackageInstallOp,
						Package: selected.Name(),
						Version: v,
					},
				},
			}, nil
		}

	default:
		return nil, fmt.Errorf("cannot process requirement type %d", req.Kind)
	}
	return nil, nil
}

// FindAll returns all packages with a given name, by version.
func (p *PackageInfo) FindAll(target string) (map[version.Version]*deb.Paragraph, error) {
	pkgs, ok := p.Packages[target]
	if !ok {
		return nil, os.ErrNotExist
	}
	return pkgs, nil
}

// FindLatest returns the latest version of the package with the given name.
func (p *PackageInfo) FindLatest(target string) (*deb.Paragraph, error) {
	pkgs, err := p.FindAll(target)
	if err != nil {
		return nil, err
	}

	vers := make([]version.Version, len(pkgs))
	count := 0
	for v, _ := range pkgs {
		vers[count] = v
		count++
	}

	sort.Slice(vers, func(i, j int) bool {
		return vers[i].LessThan(vers[j])
	})

	return pkgs[vers[len(vers)-1]], nil
}

// FindWithVersionConstraint tries to find a version of the package that satisfies the
// given version constraint.
func (p *PackageInfo) FindWithVersionConstraint(target string, constraint *deb.VersionConstraint) (*deb.Paragraph, error) {
	pkgs, err := p.FindAll(target)
	if err != nil {
		return nil, err
	}
	v, err := version.NewVersion(constraint.Version)
	if err != nil {
		return nil, err
	}

	getSorted := func() []version.Version {
		vers := make([]version.Version, len(pkgs))
		count := 0
		for v, _ := range pkgs {
			vers[count] = v
			count++
		}

		sort.Slice(vers, func(i, j int) bool {
			return vers[i].LessThan(vers[j])
		})
		return vers
	}

	switch constraint.ConstraintRelation {
	case deb.ConstraintEquals:
		pkg, ok := pkgs[v]
		if !ok {
			return nil, os.ErrNotExist
		}
		return pkg, nil

	case deb.ConstraintLessThan:
		sorted := getSorted()
		for i := len(sorted) - 1; i >= 0; i-- {
			if sorted[i].LessThan(v) {
				return pkgs[sorted[i]], nil
			}
		}

	case deb.ConstraintGreaterThan:
		sorted := getSorted()
		for i := len(sorted) - 1; i >= 0; i-- {
			if sorted[i].GreaterThan(v) {
				return pkgs[sorted[i]], nil
			}
		}

	case deb.ConstraintGreaterEquals:
		sorted := getSorted()
		for i := len(sorted) - 1; i >= 0; i-- {
			if sorted[i].GreaterThan(v) || sorted[i].Equal(v) {
				return pkgs[sorted[i]], nil
			}
		}

	case deb.ConstraintLessThanEquals:
		sorted := getSorted()
		for i := len(sorted) - 1; i >= 0; i-- {
			if sorted[i].LessThan(v) || sorted[i].Equal(v) {
				return pkgs[sorted[i]], nil
			}
		}
	}

	return nil, os.ErrNotExist
}

func checkSetCoveredDependency(coveredDeps *[]deb.Requirement, req deb.Requirement) bool {
	for _, covered := range *coveredDeps {
		if covered.Equal(&req) {
			return true
		}
	}
	t := append(*coveredDeps, req)
	*coveredDeps = t
	return false
}
