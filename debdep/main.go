package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/twitchyliquid64/debdep"
)

func main() {
	flag.Parse()

	switch flag.Arg(0) {
	case "calculate-deps":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "USAGE: %s calculate-deps <package-name>\n", os.Args[0])
			os.Exit(1)
		}

		pkgs, err := debdep.Packages(true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Read %d packages.\n", len(pkgs.Packages))

		pkg, err := pkgs.InstallGraph(flag.Arg(1), &debdep.PackageInfo{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		pkg.PrettyWrite(os.Stdout, 1)

	case "bootstrap-sequence":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "USAGE: %s calculate-deps <package-name>\n", os.Args[0])
			os.Exit(1)
		}

		pkgs, err := debdep.Packages(true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("# Read %d packages.\n", len(pkgs.Packages))

		pkg, err := pkgs.InstallGraph(flag.Arg(1), &debdep.PackageInfo{})
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

	case "check-dist":
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

	default:
		fmt.Printf("Unknown command: %q\n", flag.Arg(0))
		os.Exit(1)
	}
}
