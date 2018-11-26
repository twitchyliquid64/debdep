package debdep

import (
	"errors"
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
	PreDep  bool
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
		w.Write([]byte("["))
		if o.PreDep {
			w.Write([]byte("*"))
		} else {
			w.Write([]byte(" "))
		}
		w.Write([]byte("] "))
		w.Write([]byte(o.Package + " (" + o.Version.String() + ")\n"))
	}

	return nil
}

func (o *Operation) Unroll() []Operation {
	var out []Operation
	switch o.Kind {
	case DebPackageInstallOp:
		out = append(out, *o)
	case CompositeDependencyOp:
		for _, c := range o.DependentOperations {
			out = append(out, c.Unroll()...)
		}
	}

	return out
}

type coveredDeps struct {
	Requirements []deb.Requirement
	Packages     []struct {
		Name    string
		Version string
	}
}

func (p *PackageInfo) InstallGraph(target string, installed *PackageInfo) (*Operation, error) {
	var coveredDeps coveredDeps
	return p.buildInstallGraph(target, &coveredDeps, installed)
}

func (p *PackageInfo) buildInstallGraph(target string, coveredDeps *coveredDeps, installed *PackageInfo) (*Operation, error) {
	pkg, err := p.FindLatest(target)
	if err != nil {
		return nil, err
	}
	vers, err := pkg.Version()
	if err != nil {
		return nil, err
	}

	out := &Operation{Kind: CompositeDependencyOp}

	// Apply pre-depends, and mark them as such.
	preDeps, err := pkg.BinaryPreDepends()
	if err != nil {
		return nil, err
	}
	if preDeps.Kind != deb.AndCompositeRequirement || len(preDeps.Children) > 0 {
		op, err := p.buildInstallGraphRequirement(coveredDeps, installed, preDeps, deb.Requirement{}, true)
		if err != nil {
			return nil, err
		}
		out.DependentOperations = append(out.DependentOperations, op)
	}

	deps, err := pkg.BinaryDepends()
	if err != nil {
		return nil, err
	}
	op, err := p.buildInstallGraphRequirement(coveredDeps, installed, deps, deb.Requirement{}, false)
	if err != nil {
		return nil, err
	}
	out.DependentOperations = append(out.DependentOperations, op)
	out.DependentOperations = append(out.DependentOperations, &Operation{
		Kind:    DebPackageInstallOp,
		Package: pkg.Name(),
		Version: vers,
	})
	return out, nil
}

func (p *PackageInfo) buildInstallGraphRequirement(coveredDeps *coveredDeps, installed *PackageInfo, req deb.Requirement, parent deb.Requirement, isPreDep bool) (out *Operation, err error) {
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
			op, err := p.buildInstallGraphRequirement(coveredDeps, installed, dep, req, isPreDep)
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

		if checkSetCoveredPackage(coveredDeps, selected.Name(), v.String()) {
			return &Operation{Kind: CompositeDependencyOp}, nil
		}

		// Apply pre-depends, and mark them as such.
		preDeps, err := selected.BinaryPreDepends()
		if err != nil {
			return nil, err
		}
		var preOps *Operation
		if preDeps.Kind != deb.AndCompositeRequirement || len(preDeps.Children) > 0 {
			preOps, err = p.buildInstallGraphRequirement(coveredDeps, installed, preDeps, req, true)
			if err != nil {
				return nil, err
			}
		}

		nextDeps, err := selected.BinaryDepends()
		if err != nil {
			return nil, err
		}

		nextOps, err := p.buildInstallGraphRequirement(coveredDeps, installed, nextDeps, req, false)
		if err != nil {
			return nil, err
		}

		if nextOps.Kind == CompositeDependencyOp && len(nextOps.DependentOperations) == 0 && preOps == nil {
			return &Operation{
				Kind:    DebPackageInstallOp,
				Package: selected.Name(),
				Version: v,
				PreDep:  isPreDep,
			}, nil
		} else {
			if preOps == nil {
				return &Operation{
					Kind: CompositeDependencyOp,
					DependentOperations: []*Operation{
						nextOps,
						{
							Kind:    DebPackageInstallOp,
							Package: selected.Name(),
							Version: v,
							PreDep:  isPreDep,
						},
					},
				}, nil
			} else {
				return &Operation{
					Kind: CompositeDependencyOp,
					DependentOperations: []*Operation{
						preOps,
						nextOps,
						{
							Kind:    DebPackageInstallOp,
							Package: selected.Name(),
							Version: v,
							PreDep:  isPreDep,
						},
					},
				}, nil
			}
		}

	case deb.OrCompositeRequirement:
		for _, candidateDep := range req.Children {
			op, err := p.buildInstallGraphRequirement(coveredDeps, installed, candidateDep, req, isPreDep)
			if err != nil {
				if _, wasDep := err.(ErrDependency); wasDep {
					continue
				}
				return nil, err
			}
			return op, nil
		}
		return nil, errors.New("no package meeting any requirement available")

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

func checkSetCoveredDependency(coveredDeps *coveredDeps, req deb.Requirement) bool {
	for _, covered := range coveredDeps.Requirements {
		if covered.Equal(&req) {
			return true
		}
	}
	coveredDeps.Requirements = append(coveredDeps.Requirements, req)
	return false
}

func checkSetCoveredPackage(coveredDeps *coveredDeps, pkg, version string) bool {
	for _, covered := range coveredDeps.Packages {
		if covered.Name == pkg && covered.Version == version {
			return true
		}
	}
	coveredDeps.Packages = append(coveredDeps.Packages, struct {
		Name    string
		Version string
	}{
		Name:    pkg,
		Version: version,
	})
	return false
}
