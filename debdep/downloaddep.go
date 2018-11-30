package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

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

func downloadFile(tr *http.Transport, url, outPath, md5 string) error {
	client := &http.Client{
		Transport: tr,
		Timeout:   45 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if md5 != "" {
		req.Header.Add("ETag", md5)
	}

	r, err := client.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	switch r.StatusCode {
	case http.StatusNotModified:
		fmt.Printf("[304] %s is not modified, skipping", path.Base(outPath))
		return nil
	case http.StatusOK:
	default:
		return fmt.Errorf("unexpected response code '%d' (%s)", r.StatusCode, r.Status)
	}

	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0655)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r.Body)
	return err
}

func md5IfExists(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil)[:16])
}

func downloadWorker(wg *sync.WaitGroup, work chan downloadWork, pkgs *debdep.PackageInfo, tr *http.Transport) {
	defer wg.Done()
	for dl := range work {
		fmt.Printf("Downloading: %v (%v)\n", dl.Package, dl.Version.String())
		url, err := pkgs.FetchPath(dl.Package, dl.Version)
		if err != nil {
			fmt.Printf("[%s] Error!: %v\n", dl.Package, err)
			continue
		}

		outPath := path.Join(dl.OutPath, path.Base(url))
		if err := downloadFile(tr, url, outPath, md5IfExists(outPath)); err != nil {
			fmt.Printf("[%s] Error!: %v\n", dl.Package, err)
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

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          3,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
		DisableCompression:    false,
	}
	workChan := make(chan downloadWork)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go downloadWorker(&wg, workChan, pkgs, tr)
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
