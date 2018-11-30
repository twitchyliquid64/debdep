package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/twitchyliquid64/debdep"
)

var (
	fetchBase         = flag.String("addr", "https://cdn-aws.deb.debian.org/debian", "Base repository URL")
	codename          = flag.String("codename", "buster", "Debian codename")
	arch              = flag.String("arch", "amd64", "Architecture")
	pkgsFromFile      = flag.String("packages_file", "", "Path to read package info from instead of fetching from remote")
	installedFromFile = flag.String("installed_file", "", "Path to read installed package info")
)

func main() {
	flag.Parse()
	debdep.SetBaseURL(*fetchBase)
	debdep.SetCodename(*codename)
	debdep.SetArch(*arch)

	var packages *debdep.PackageInfo
	var err error
	installed := &debdep.PackageInfo{}
	if *installedFromFile != "" {
		installed, err = debdep.LoadPackageInfo(*installedFromFile, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading installed packages: %v\n", err)
			os.Exit(1)
		}
	}

	if *pkgsFromFile == "" {
		packages, err = debdep.Packages(true)
	} else {
		packages, err = debdep.LoadPackageInfo(*pkgsFromFile, true)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading packages: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Read %d packages.\n", len(packages.Packages))

	switch flag.Arg(0) {
	case "all-priority":
		allPriorityCmd(packages, flag.Arg(1))

	case "calculate-deps":
		calculateDepsCommand(packages, installed, flag.Arg(1))

	case "bootstrap-sequence":
		bootstrapSequenceCmd(packages, installed, flag.Arg(1))

	case "check-dist":
		checkDistCmd()

	case "download-pkg-info":
		downloadPackageInfo(flag.Arg(1))

	case "download-priority-deps":
		downloadPriorityDeps(packages, installed, flag.Arg(1), flag.Arg(2))

	default:
		fmt.Printf("Unknown command: %q\n", flag.Arg(0))
		fmt.Println("Available commands: all-priority, calculate-deps, bootstrap-sequence, check-dist")
		os.Exit(1)
	}
}

func downloadPackageInfo(path string) {
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s download-pkg-info <output-path>\n", os.Args[0])
		os.Exit(1)
	}

	r, err := debdep.RepositoryPackagesReader(true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer r.Close()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0655)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func calculateDepsCommand(pkgs, installed *debdep.PackageInfo, pkgName string) {
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s calculate-deps <package-name>\n", os.Args[0])
		os.Exit(1)
	}

	pkg, err := pkgs.InstallGraph(pkgName, installed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating install graph: %v\n", err)
		os.Exit(1)
	}
	pkg.PrettyWrite(os.Stdout, 1)
}

func allPriorityCmd(pkgs *debdep.PackageInfo, priority string) {
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s all-priority <all-priority>\n", os.Args[0])
		os.Exit(1)
	}

	var packages []string
	if priority == "essential" {
		packages = pkgs.GetAllEssential()
	} else {
		packages = pkgs.GetAllByPriority(priority)
	}
	for i, p := range packages {
		fmt.Printf("%.03d %s\n", i, p)
	}
}

func bootstrapSequenceCmd(pkgs, installed *debdep.PackageInfo, pkgName string) {
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s bootstrap-sequence <package-name>\n", os.Args[0])
		os.Exit(1)
	}

	pkg, err := pkgs.InstallGraph(pkgName, installed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	for i, op := range pkg.Unroll() {
		marker := "[ ]"
		if op.PreDep {
			marker = "[*]"
		}
		fmt.Printf("%.03d %s %s %s\n", i, marker, op.Package, op.Version.String())
	}
}

func checkDistCmd() {
	err := debdep.CheckReleaseStatus()
	if err != nil {
		if relData, ok := err.(debdep.ReleaseInconsistency); ok {
			fmt.Printf("Configured debian state is inconsistent with the repositories!\n")
			if relData.WantDistro != "" {
				fmt.Printf("\tDistribution = %q (our setting: %q)\n", relData.GotDistro, relData.WantDistro)
			}
			if relData.WantArch != "" {
				fmt.Printf("\tArch = %q (our setting: %q)\n", relData.GotArch, relData.WantArch)
			}
			if relData.WantComponent != "" {
				fmt.Printf("\tComponent = %q (our setting: %q)\n", relData.GotComponent, relData.WantComponent)
			}
		} else {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}
