package debdep

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/twitchyliquid64/debdep/deb"
	"golang.org/x/crypto/openpgp"

	version "github.com/knqyf263/go-deb-version"
)

var (
	codename     = "buster"
	distribution = "testing" // Also, stable, unstable.
	component    = "main"    // Also, non-free, contrib etc.
	arch         = "amd64"
	fetchBase    = "https://cdn-aws.deb.debian.org/debian"
)

func SetCodename(in string) {
	codename = in
}

func SetDistribution(in string) {
	distribution = in
}

func SetComponent(in string) {
	component = in
}

func SetArch(in string) {
	arch = in
}

func SetBaseURL(in string) {
	fetchBase = in
}

func url(isBinary bool) string {
	src := "source-"
	if isBinary {
		src = "binary-"
	}
	return fetchBase + "/dists/" + codename + "/" + component + "/" + src + arch
}

// ReleaseInconsistency is returned by CheckReleaseStatus if the settings for distribution/component/arch
// do not match what the repository reports.
type ReleaseInconsistency struct {
	dirty bool

	WantDistro, GotDistro       string
	WantComponent, GotComponent string
	WantArch, GotArch           string
}

func (r ReleaseInconsistency) Error() string {
	out := ""
	if r.WantArch != "" {
		out = "inconsistent arch"
	}
	if r.WantComponent != "" {
		if out != "" {
			out += ", "
		}
		out += "inconsistent component"
	}
	if r.WantDistro != "" {
		if out != "" {
			out += ", "
		}
		out += "inconsistent distro"
	}
	return out
}

// CheckReleaseStatus returns an error if it could not connect to the repository,
// or if the repository metadata was inconsistent with settings for
// the distribution/arch/component etc.
func CheckReleaseStatus() error {
	var out ReleaseInconsistency

	req, err := http.Get(url(true) + "/Release")
	if err != nil {
		return err
	}
	defer req.Body.Close()
	in := bufio.NewReader(req.Body)
	for {
		line, _, err := in.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		spl := strings.Split(string(line), ": ")
		if len(spl) < 2 {
			return fmt.Errorf("unexpected line: %q", line)
		}

		switch spl[0] {
		case "Archive":
			if spl[1] != distribution {
				out.dirty = true
				out.WantDistro = distribution
				out.GotDistro = spl[1]
			}
		case "Component":
			if spl[1] != component {
				out.dirty = true
				out.WantComponent = component
				out.GotComponent = spl[1]
			}
		case "Architecture":
			if spl[1] != arch {
				out.dirty = true
				out.WantArch = arch
				out.GotArch = spl[1]
			}
		case "Origin", "Label", "Acquire-By-Hash":

		}
	}

	if out.dirty {
		return out
	}
	return nil
}

// PackageInfo keeps track of package information.
type PackageInfo struct {
	BinaryPackages  bool
	Packages        map[string]map[version.Version]*deb.Paragraph
	virtualPackages map[string][]*deb.Paragraph
}

// GetAllByPriority returns all packages with a given priority.
func (p *PackageInfo) GetAllByPriority(priority string) []string {
	var out []string
	for n, _ := range p.Packages {
		latest, _ := p.FindLatest(n)
		if latest.Values["Priority"] == priority {
			out = append(out, n)
		}
	}
	return out
}

// GetAllEssential returns all packages marked as essential.
func (p *PackageInfo) GetAllEssential() []string {
	var out []string
	for n, _ := range p.Packages {
		latest, _ := p.FindLatest(n)
		if latest.Values["Essential"] == "yes" {
			out = append(out, n)
		}
	}
	return out
}

// HasPackage returns true if a package meeting the given requirements
// is present. This includes virtual packages.
func (p *PackageInfo) HasPackage(req deb.Requirement) (bool, error) {
	if req.Kind != deb.PackageRelationRequirement {
		return false, errors.New("only requirement.Kind == PackageRelationRequirement supported")
	}
	if _, exists := p.Packages[req.Package]; !exists {
		if _, virtPkgExists := p.virtualPackages[req.Package]; virtPkgExists && req.VersionConstraint == nil {
			return true, nil
		}
		return false, nil
	}
	if req.VersionConstraint == nil {
		return true, nil
	} else {
		if _, err := p.FindWithVersionConstraint(req.Package, req.VersionConstraint); err != nil {
			if err == os.ErrNotExist {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

// AddPkg appends a package, overwriting any name+version combination that already exists.
func (p *PackageInfo) AddPkg(pkg *deb.Paragraph) error {
	if p.virtualPackages == nil {
		p.virtualPackages = make(map[string][]*deb.Paragraph)
	}
	if p.Packages == nil {
		p.Packages = make(map[string]map[version.Version]*deb.Paragraph)
	}
	return pkgInfoAppend(pkg, p.Packages, p.virtualPackages)
}

// FetchPath returns the URL to retrieve a package.
func (p *PackageInfo) FetchPath(pkg string, version version.Version) (string, error) {
	pkgs, ok := p.Packages[pkg]
	if !ok {
		return "", os.ErrNotExist
	}
	s, ok := pkgs[version]
	if !ok {
		return "", os.ErrNotExist
	}
	return fetchBase + s.Values["Filename"], nil
}

// readPackages consumes package info from the given reader.
func readPackages(r io.Reader, isBinaryPackages bool) (*PackageInfo, error) {
	packages := make(map[string]map[version.Version]*deb.Paragraph)
	virtualPackages := make(map[string][]*deb.Paragraph)
	d := deb.NewDecoder(r)

	for {
		var p deb.Paragraph
		err := d.Decode(&p)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if err := pkgInfoAppend(&p, packages, virtualPackages); err != nil {
			return nil, err
		}
	}
	return &PackageInfo{
		BinaryPackages:  isBinaryPackages,
		Packages:        packages,
		virtualPackages: virtualPackages,
	}, nil
}

func pkgInfoAppend(p *deb.Paragraph, packages map[string]map[version.Version]*deb.Paragraph, virtualPackages map[string][]*deb.Paragraph) error {
	if _, ok := packages[p.Name()]; !ok {
		packages[p.Name()] = make(map[version.Version]*deb.Paragraph)
	}
	vers, err := p.Version()
	if err != nil {
		return err
	}
	packages[p.Name()][vers] = p

	for _, virtualPackage := range p.Provides() {
		virtualPackages[virtualPackage] = append(virtualPackages[virtualPackage], p)
	}
	return nil
}

// LoadPackageInfo reads a file detailing packages from disk.
func LoadPackageInfo(path string, isBinaryPackages bool) (*PackageInfo, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return readPackages(r, isBinaryPackages)
}

// RepositoryPackagesReader returns a reader for package information from the
// configured remote repository.
func RepositoryPackagesReader(binary bool) (io.ReadCloser, error) {
	req, err := http.Get(url(binary) + "/Packages.gz")
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(req.Body)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// TODO: make work.
func fetchReleaseInfo(binary bool) (error, error) {
	req, err := http.Get(url(binary) + "/Release")
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	relData, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	sig, err := http.Get(url(binary) + "/Release.gpg")
	if err != nil {
		return nil, err
	}
	defer sig.Body.Close()
	sigData, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	keys, err := os.Open("/usr/share/keyrings/debian-archive-keyring.gpg")
	if err != nil {
		return nil, err
	}
	defer keys.Close()
	keyring, err := openpgp.ReadKeyRing(keys)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Keys: %+v\n", keyring)
	for i, _ := range keyring {
		fmt.Printf("\t%+v\n", keyring[i])
	}
	if _, err := openpgp.CheckDetachedSignature(keyring, bytes.NewBuffer(relData), bytes.NewBuffer(sigData)); err != nil {
		return nil, fmt.Errorf("Signature check failed: %v", err)
	}

	return nil, nil
}

// Packages returns information about packages available in the
// remote repository.
func Packages(binary bool) (*PackageInfo, error) {
	r, err := RepositoryPackagesReader(binary)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return readPackages(r, binary)
}
