package debdep

import (
	"testing"

	"github.com/twitchyliquid64/debdep/deb"

	version "github.com/knqyf263/go-deb-version"
)

func makePkg(t *testing.T, name string, versions []string, depends string) (out map[version.Version]*deb.Paragraph) {
	t.Helper()
	out = make(map[version.Version]*deb.Paragraph)

	for _, ver := range versions {
		v, err := version.NewVersion(ver)
		if err != nil {
			t.Fatal(err)
		}
		out[v] = &deb.Paragraph{
			Values: map[string]string{
				"Package": name,
				"Version": v.String(),
				"Depends": depends,
			},
		}
	}
	return out
}

func TestInstallGraph(t *testing.T) {
	pkgInfo := &PackageInfo{
		BinaryPackages: true,
		Packages: map[string]map[version.Version]*deb.Paragraph{
			"base": makePkg(t, "base", []string{"1.3.2", "1.9.2"}, "kek"),
			"kek":  makePkg(t, "kek", []string{"1.3.2", "1.9.2"}, ""),
		},
	}

	graph, err := pkgInfo.InstallGraph("base", nil)
	if err != nil {
		t.Fatalf("InstallGraph() returned err: %v", err)
	}
	//graph.PrettyWrite(os.Stdout, 1)
	if graph == nil {
		t.Fatalf("InstallGraph() returned nil")
	}

	if graph.Kind != CompositeDependencyOp && len(graph.DependentOperations) != 2 {
		t.Fatalf("Expected root to be Composite with 2 children, got Kind=%v & len(children) = %d", graph.Kind, len(graph.DependentOperations))
	}
	if graph.DependentOperations[0].Kind != DebPackageInstallOp || graph.DependentOperations[0].Package != "kek" || graph.DependentOperations[0].Version.String() != "1.9.2" {
		t.Errorf("First package-dep incorrect, got %+v", graph.DependentOperations[0])
	}
	if graph.DependentOperations[1].Kind != DebPackageInstallOp || graph.DependentOperations[1].Package != "base" || graph.DependentOperations[1].Version.String() != "1.9.2" {
		t.Errorf("Second package-dep incorrect, got %+v", graph.DependentOperations[1])
	}
}

func TestInstallGraphVersionConstraints(t *testing.T) {
	pkgInfo := &PackageInfo{
		BinaryPackages: true,
		Packages: map[string]map[version.Version]*deb.Paragraph{
			"base": makePkg(t, "base", []string{"1.3.2", "1.9.2"}, "kek (<< 1.7), meep (= 1.3.2), yolo"),
			"kek":  makePkg(t, "kek", []string{"1.3.2", "1.9.2"}, ""),
			"meep": makePkg(t, "meep", []string{"1.3.2", "1.9.2", "2.0.0"}, ""),
			"yolo": makePkg(t, "yolo", []string{"1"}, ""),
		},
	}

	graph, err := pkgInfo.InstallGraph("base", nil)
	if err != nil {
		t.Fatalf("InstallGraph() returned err: %v", err)
	}
	if graph == nil {
		t.Fatalf("InstallGraph() returned nil")
	}
	//graph.PrettyWrite(os.Stdout, 1)

	if graph.Kind != CompositeDependencyOp && len(graph.DependentOperations) != 3 {
		t.Fatalf("Expected root to be Composite with 2 children, got Kind=%v & len(children) = %d", graph.Kind, len(graph.DependentOperations))
	}
	if graph.DependentOperations[0].Kind != CompositeDependencyOp || len(graph.DependentOperations[0].DependentOperations) != 3 {
		t.Errorf("First composite incorrect, got %+v", graph.DependentOperations[0])
	}
	sub := graph.DependentOperations[0].DependentOperations
	if sub[0].Kind != DebPackageInstallOp || sub[0].Package != "kek" || sub[0].Version.String() != "1.3.2" {
		t.Errorf("Nested package-dep 1 incorrect, got %+v", sub[0])
	}
	if sub[1].Kind != DebPackageInstallOp || sub[1].Package != "meep" || sub[1].Version.String() != "1.3.2" {
		t.Errorf("Nested package-dep 2 incorrect, got %+v", sub[0])
	}
	if sub[2].Kind != DebPackageInstallOp || sub[2].Package != "yolo" || sub[2].Version.String() != "1" {
		t.Errorf("Nested package-dep 3 incorrect, got %+v", sub[0])
	}
	if graph.DependentOperations[1].Kind != DebPackageInstallOp || graph.DependentOperations[1].Package != "base" || graph.DependentOperations[1].Version.String() != "1.9.2" {
		t.Errorf("Third package-dep incorrect, got %+v", graph.DependentOperations[1])
	}
}

func TestInstallGraphDeepNested(t *testing.T) {
	pkgInfo := &PackageInfo{
		BinaryPackages: true,
		Packages: map[string]map[version.Version]*deb.Paragraph{
			"base":     makePkg(t, "base", []string{"1.3.2", "1.9.2"}, "kek, meep"),
			"kek":      makePkg(t, "kek", []string{"1.3.2"}, ""),
			"meep":     makePkg(t, "meep", []string{"1.9.2", "2.0.0"}, "yolo"),
			"yolo":     makePkg(t, "yolo", []string{"1"}, "swaggins"),
			"swaggins": makePkg(t, "swaggins", []string{"2"}, ""),
		},
	}

	graph, err := pkgInfo.InstallGraph("base", nil)
	if err != nil {
		t.Fatalf("InstallGraph() returned err: %v", err)
	}
	if graph == nil {
		t.Fatalf("InstallGraph() returned nil")
	}
	// graph.PrettyWrite(os.Stdout, 1)

	if graph.Kind != CompositeDependencyOp && len(graph.DependentOperations) != 2 {
		t.Fatalf("Expected root to be Composite with 2 children, got Kind=%v & len(children) = %d", graph.Kind, len(graph.DependentOperations))
	}
	if graph.DependentOperations[0].Kind != CompositeDependencyOp || len(graph.DependentOperations[0].DependentOperations) != 2 {
		t.Errorf("First composite incorrect, got %+v", graph.DependentOperations[0])
	}
	if graph.DependentOperations[1].Kind != DebPackageInstallOp || graph.DependentOperations[1].Package != "base" || graph.DependentOperations[1].Version.String() != "1.9.2" {
		t.Errorf("Last composite incorrect, got %+v", graph.DependentOperations[1])
	}
	sub := graph.DependentOperations[0].DependentOperations
	if sub[0].Kind != DebPackageInstallOp || sub[0].Package != "kek" || sub[0].Version.String() != "1.3.2" {
		t.Errorf("First package-dep incorrect, got %+v", sub[0])
	}
	if sub[1].Kind != CompositeDependencyOp || len(sub[1].DependentOperations) != 2 {
		t.Errorf("1st-level nested composite incorrect, got %+v", sub[1])
	}
	sub = sub[1].DependentOperations
	if sub[0].Kind != CompositeDependencyOp || len(sub[0].DependentOperations) != 2 {
		t.Errorf("2nd-level nested composite incorrect, got %+v", sub[0])
	}
	if sub[1].Kind != DebPackageInstallOp || sub[1].Package != "meep" || sub[1].Version.String() != "2.0.0" {
		t.Errorf("Fourth package-dep incorrect, got %+v", sub[1])
	}
	sub = sub[0].DependentOperations
	if sub[0].Kind != DebPackageInstallOp || sub[0].Package != "swaggins" || sub[0].Version.String() != "2" {
		t.Errorf("Second package-dep incorrect, got %+v", sub[0])
	}
	if sub[1].Kind != DebPackageInstallOp || sub[1].Package != "yolo" || sub[1].Version.String() != "1" {
		t.Errorf("Third package-dep incorrect, got %+v", sub[1])
	}
}

func TestInstallGraphLoop(t *testing.T) {
	pkgInfo := &PackageInfo{
		BinaryPackages: true,
		Packages: map[string]map[version.Version]*deb.Paragraph{
			"base": makePkg(t, "base", []string{"1.3.2", "1.9.2"}, "kek"),
			"kek":  makePkg(t, "kek", []string{"1.3.2"}, "meep"),
			"meep": makePkg(t, "meep", []string{"1.3.2"}, "kek"),
		},
	}

	graph, err := pkgInfo.InstallGraph("base", nil)
	if err != nil {
		t.Fatalf("InstallGraph() returned err: %v", err)
	}
	if graph == nil {
		t.Fatalf("InstallGraph() returned nil")
	}
	// graph.PrettyWrite(os.Stdout, 1)

	if graph.DependentOperations[0].DependentOperations[0].Package != "meep" {
		t.Error("Expected first pkg to be meep")
	}
	if graph.DependentOperations[0].DependentOperations[1].Package != "kek" {
		t.Error("Expected second pkg to be kek")
	}
	if graph.DependentOperations[1].Package != "base" {
		t.Error("Expected third pkg to be base")
	}
}

func TestInstallGraphOrRequirements(t *testing.T) {
	pkgInfo := &PackageInfo{
		BinaryPackages: true,
		Packages: map[string]map[version.Version]*deb.Paragraph{
			"base": makePkg(t, "base", []string{"1.3.2", "1.9.2"}, "kek (>> 2.0.0) | meep"),
			"kek":  makePkg(t, "kek", []string{"1.3.2"}, ""),
			"meep": makePkg(t, "meep", []string{"1.3.2"}, ""),
		},
	}

	graph, err := pkgInfo.InstallGraph("base", nil)
	if err != nil {
		t.Fatalf("InstallGraph() returned err: %v", err)
	}
	if graph == nil {
		t.Fatalf("InstallGraph() returned nil")
	}
	//graph.PrettyWrite(os.Stdout, 1)

	if graph.Kind != CompositeDependencyOp && len(graph.DependentOperations) != 2 {
		t.Fatalf("Expected root to be Composite with 2 children, got Kind=%v & len(children) = %d", graph.Kind, len(graph.DependentOperations))
	}
	if graph.DependentOperations[0].Kind != DebPackageInstallOp || graph.DependentOperations[0].Package != "meep" || graph.DependentOperations[0].Version.String() != "1.3.2" {
		t.Errorf("First package-dep incorrect, got %+v", graph.DependentOperations[0])
	}
	if graph.DependentOperations[1].Kind != DebPackageInstallOp || graph.DependentOperations[1].Package != "base" || graph.DependentOperations[1].Version.String() != "1.9.2" {
		t.Errorf("Second package-dep incorrect, got %+v", graph.DependentOperations[1])
	}
}

func TestInstallGraphUnsatisfiedMissing(t *testing.T) {
	pkgInfo := &PackageInfo{
		BinaryPackages: true,
		Packages: map[string]map[version.Version]*deb.Paragraph{
			"base": makePkg(t, "base", []string{"1.3.2"}, "missing"),
		},
	}

	graph, err := pkgInfo.InstallGraph("base", nil)
	if err == nil {
		t.Fatal("InstallGraph() returned nil instead of error")
	}
	if graph != nil {
		t.Fatalf("InstallGraph() returned non-nil")
	}
	info, ok := err.(ErrDependency)
	if !ok {
		t.Fatalf("error was not type ErrDependency, got %t", err)
	}
	if info.DependencyPackage != "missing" || info.VersionConstraint != nil {
		t.Errorf("Error parameters incorrect, got %+v", info)
	}
}

func TestInstallGraphUnsatisfiedNoVersion(t *testing.T) {
	pkgInfo := &PackageInfo{
		BinaryPackages: true,
		Packages: map[string]map[version.Version]*deb.Paragraph{
			"base":     makePkg(t, "base", []string{"1.3.2"}, "swaggins (>> 2.0.0)"),
			"swaggins": makePkg(t, "swaggins", []string{"1.3.2"}, "swaggins (>> 2.0.0)"),
		},
	}

	graph, err := pkgInfo.InstallGraph("base", nil)
	if err == nil {
		t.Fatal("InstallGraph() returned nil instead of error")
	}
	if graph != nil {
		t.Fatalf("InstallGraph() returned non-nil")
	}
	info, ok := err.(ErrDependency)
	if !ok {
		t.Fatalf("error was not type ErrDependency, got %t", err)
	}
	if info.DependencyPackage != "swaggins" || info.VersionConstraint == nil || info.VersionConstraint.ConstraintRelation != deb.ConstraintGreaterThan || info.VersionConstraint.Version != "2.0.0" {
		t.Errorf("Error parameters incorrect, got %v", info)
	}
}
