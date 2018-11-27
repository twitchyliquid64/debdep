package debdep

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/twitchyliquid64/debdep/deb"

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

func repositoryPackagesReader(binary bool) (io.ReadCloser, error) {
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

// PackageInfo keeps track of package information.
type PackageInfo struct {
	BinaryPackages bool
	Packages       map[string]map[version.Version]*deb.Paragraph
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

func Packages(binary bool) (*PackageInfo, error) {
	r, err := repositoryPackagesReader(binary)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	packages := make(map[string]map[version.Version]*deb.Paragraph)

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

		if _, ok := packages[p.Name()]; !ok {
			packages[p.Name()] = make(map[version.Version]*deb.Paragraph)
		}
		vers, err := p.Version()
		if err != nil {
			return nil, err
		}
		packages[p.Name()][vers] = &p
	}
	return &PackageInfo{
		BinaryPackages: binary,
		Packages:       packages,
	}, nil
}
