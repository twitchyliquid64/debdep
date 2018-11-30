package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"

	version "github.com/knqyf263/go-deb-version"
	"github.com/twitchyliquid64/debdep"
	"github.com/twitchyliquid64/debdep/deb"
)

func multitargetInstallGraph(pkgs, installed *debdep.PackageInfo, targets []string) ([]debdep.Operation, error) {
	var debOps []debdep.Operation
	for _, pkg := range targets {
		isInstalled, err := installed.HasPackage(deb.Requirement{Kind: deb.PackageRelationRequirement, Package: pkg})
		if err != nil {
			return nil, err
		}
		if !isInstalled {
			graph, err := pkgs.InstallGraph(pkg, installed)
			if err != nil {
				return nil, err
			}
			newOps := graph.Unroll()
			for _, op := range newOps {
				installed.AddPkg(&deb.Paragraph{
					Values: map[string]string{
						"Package": op.Package,
						"Version": op.Version.String(),
					},
				})
			}
			debOps = append(debOps, newOps...)
		}
	}
	return debOps, nil
}

type downloadWork struct {
	OutPath string
	Package string
	Version version.Version
}

func downloadFile(url, outPath string) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0655)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r.Body)
	return err
}

func downloadWorker(wg *sync.WaitGroup, work chan downloadWork, pkgs *debdep.PackageInfo) {
	defer wg.Done()
	for dl := range work {
		fmt.Printf("Downloading: %v (%v)\n", dl.Package, dl.Version.String())
		url, err := pkgs.FetchPath(dl.Package, dl.Version)
		if err != nil {
			fmt.Printf("Error!: %v\n", err)
			continue
		}

		if err := downloadFile(url, path.Join(dl.OutPath, path.Base(url))); err != nil {
			fmt.Printf("Error!: %v\n", err)
			continue
		}
	}
}

func downloadPriorityDeps(pkgs, installed *debdep.PackageInfo, priority, outPath string) error {
	var packages []string
	if priority == "essential" {
		packages = pkgs.GetAllEssential()
	} else {
		packages = pkgs.GetAllByPriority(priority)
	}

	debOps, err := multitargetInstallGraph(pkgs, installed, packages)
	if err != nil {
		return err
	}

	workChan := make(chan downloadWork)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go downloadWorker(&wg, workChan, pkgs)
	}
	for _, op := range debOps {
		workChan <- downloadWork{
			OutPath: outPath,
			Package: op.Package,
			Version: op.Version,
		}
	}
	close(workChan)
	wg.Wait()

	return nil
}
