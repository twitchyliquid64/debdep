# debdep

Debdep understand the dependency tree in Debian packages, and can be used to parse the package list
in the debian repositories.

Functions are exposed to :

 * Parse package specs / control files
 * Parse dependency specifications into Requirements
 * Resolve the dependency graph into an ordered set of packages+versions to install

## Examples

### End-to-end: Debdep

See the main package `debdep`.

#### Installation

Make sure you have `go` installed.

```shell
go get github.com/twitchyliquid64/debdep
go build -o debdep github.com/twitchyliquid64/debdep/debdep
sudo install -v -m 0755 --strip debdep /usr/bin/debdep
```


*Calculate-deps* sub-command:

```shell

./debdep calculate-deps screen

Read 55944 packages.
 composite:
  composite:
   composite:
    composite:
     composite:
      composite:
       package-dep: [ ] gcc-8-base (8.2.0-9)
       composite:
      package-dep: [ ] libgcc1 (1:8.2.0-9)
     package-dep: [ ] libc6 (2.27-8)
    package-dep: [*] libbz2-1.0 (1.0.6-9)
   composite:
   package-dep: [*] liblzma5 (5.2.2-1.3)
   composite:
    composite:
     composite:
     package-dep: [ ] libpcre3 (2:8.39-11)
    package-dep: [*] libselinux1 (2.8-1+b1)
   package-dep: [*] zlib1g (1:1.2.11.dfsg-1)
  composite:
   composite:
    composite:
     composite:
      package-dep: [ ] libattr1 (1:2.4.47-2+b2)
      composite:
     package-dep: [*] libacl1 (2.2.52-3+b1)
    composite:
    composite:
   composite:
   package-dep: [ ] tar (1.30+dfsg-3)
  package-dep: [ ] dpkg (1.19.2)
```

*bootstrap-sequence* sub-command:

```shell

./debdep bootstrap-sequence screen

# Read 55944 packages.
000 [ ] gcc-8-base 8.2.0-9
001 [ ] libgcc1 1:8.2.0-9
002 [ ] libc6 2.27-8
003 [ ] libaudit-common 1:2.8.4-2
004 [ ] libcap-ng0 0.7.9-1
005 [ ] libaudit1 1:2.8.4-2
006 [*] libbz2-1.0 1.0.6-9
007 [*] liblzma5 5.2.2-1.3
008 [ ] libpcre3 2:8.39-11
009 [*] libselinux1 2.8-1+b1
010 [*] zlib1g 1:1.2.11.dfsg-1
011 [ ] libattr1 1:2.4.47-2+b2
012 [*] libacl1 2.2.52-3+b1
013 [ ] tar 1.30+dfsg-3
014 [*] dpkg 1.19.2
015 [*] perl-base 5.28.0-4
016 [ ] debconf 1.5.69
017 [ ] libpam0g 1.1.8-3.8
018 [ ] libtinfo6 6.1+20181013-1
019 [ ] libutempter0 1.1.6-3
020 [ ] screen 4.6.2-3

```

The asterisks symbolize pre-dependencies. These can be thought of as install 'barriers', which must be fully
installed before subsequent packages can begin to be installed.


### In Go:

Setup (optional):

```go

// These are defaults at time of writing
debdep.SetCodename("buster")
debdep.SetDistribution("testing")
debdep.SetArch("amd64")
```

Read & Parse all packages in the configured Debian repository.

```go

pkgs, err := debdep.Packages(true)
if err != nil {
  fmt.Fprintf(os.Stderr, "Error: %v\n", err)
  os.Exit(1)
}
fmt.Printf("Read %d packages.\n", len(pkgs.Packages))
```

Compute an install graph (assuming no packages installed) for the package *screen*

```go

pkg, err := pkgs.InstallGraph("screen", &debdep.PackageInfo{})
if err != nil {
  fmt.Fprintf(os.Stderr, "Error: %v\n", err)
  os.Exit(1)
}
pkg.PrettyWrite(os.Stdout, 1) // Pretty-print the graph.
```

## Known issues

 * Wierd behavior with circular dependencies - not certain the resolver is producing the correct order in this case.

## TODO

 * Fix known issues.
 * Support for source packages.
 * Support for multi-arch.
 * Verify the integrity of the remote repository.
 * Support for downloading packages.
 * Support for extracting packages into the system.
 * Implement fake-installs to /var/lib/dpkg/status for bootstrapping purposes.
