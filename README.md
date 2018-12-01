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


**Additional arguments**

 * `--packages_file` - Path to the available packages file, will be used instead of reading the package list from the web repository.
 * `--installed_file` - Path to the status file, which details all installed packages. This is typically `/var/lib/dpkg/status`. If specified, packages which
 are already installed will not be included in the dependency graph.


 **download-pkg-info**

 Downloads the package information file to the given path.


 **download-priority-deps**

 Downloads all packages with the given priority (or, if *essential*, all essential packages), and downloads
 their dependencies (including pre-dependencies).

 ```shell
 > go run github.com/twitchyliquid64/debdep/debdep download-priority-deps essential ./

 Downloading: libaudit-common (1:2.8.4-2)
 Downloading: gcc-8-base (8.2.0-9)
 Downloading: libgcc1 (1:8.2.0-9)
 Downloading: libc6 (2.27-8)
 Downloading: libcap-ng0 (0.7.9-1)
 Downloading: libaudit1 (1:2.8.4-2)
 Downloading: libbz2-1.0 (1.0.6-9)
 Downloading: liblzma5 (5.2.2-1.3)
 Downloading: libpcre3 (2:8.39-11)
 Downloading: libselinux1 (2.8-1+b1)
 Downloading: zlib1g (1:1.2.11.dfsg-1)
 Downloading: libattr1 (1:2.4.47-2+b2)
 Downloading: libacl1 (2.2.52-3+b1)
 Downloading: tar (1.30+dfsg-3)
 Downloading: dpkg (1.19.2)
 Downloading: perl-base (5.28.0-4)
 Downloading: debconf (1.5.69)
 Downloading: libpam0g (1.1.8-3.8)
 Downloading: libdb5.3 (5.3.28+dfsg1-0.2)
 Downloading: libpam-modules-bin (1.1.8-3.8)
 Downloading: libpam-modules (1.1.8-3.8)
 Downloading: libpam-runtime (1.1.8-3.8)
 Downloading: login (1:4.5-1.1)
 Downloading: debianutils (4.8.6)
 Downloading: diffutils (1:3.6-1)
 Downloading: libgpg-error0 (1.32-3)
 Downloading: libgcrypt20 (1.8.4-3)
 Downloading: liblz4-1 (1.8.2-1)
 Downloading: libsystemd0 (239-13)
 Downloading: bsdutils (1:2.32.1-0.1)
 Downloading: libuuid1 (2.32.1-0.1)
 Downloading: libblkid1 (2.32.1-0.1)
 Downloading: libmount1 (2.32.1-0.1)
 Downloading: libsmartcols1 (2.32.1-0.1)
 Downloading: libtinfo6 (6.1+20181013-1)
 Downloading: libudev1 (239-13)
 Downloading: libfdisk1 (2.32.1-0.1)
 Downloading: libncursesw6 (6.1+20181013-1)
 Downloading: fdisk (2.32.1-0.1)
 Downloading: util-linux (2.32.1-0.1)
 Downloading: coreutils (8.30-1)
 Downloading: libdebconfclient0 (0.245)
 Downloading: base-passwd (3.5.45)
 Downloading: libgmp10 (2:6.1.2+dfsg-3)
 Downloading: libmpfr6 (4.0.1-1)
 Downloading: readline-common (7.0-5)
 Downloading: libreadline7 (7.0-5)
 Downloading: libsigsegv2 (2.12-2)
 Downloading: gawk (1:4.2.1+dfsg-1)
 Downloading: base-files (10.1)
 Downloading: bash (4.4.18-3.1)
 Downloading: gzip (1.9-2.1)
 Downloading: grep (3.1-2)
 Downloading: hostname (3.21)
 Downloading: libc-bin (2.27-8)
 Downloading: findutils (4.6.0+git+20181018-1)
 Downloading: ncurses-bin (6.1+20181013-1)
 Downloading: ncurses-base (6.1+20181013-1)
 Downloading: init-system-helpers (1.56)
 Downloading: sysvinit-utils (2.92~beta-2)
 Downloading: sed (4.5-2)
 Downloading: dash (0.5.10.2-1)
 ```

**download-specific-deps sub-command**

This command downloads all the packages specified, and all their dependencies.

This is intended to be used with `--installed_file`, which will avoid downloading
dependencies which are already installed.

```shell
./debdep download-specific-deps "apt screen htop" /var/my_debs
```

**calculate-deps sub-command**

This command shows the dependency tree for a given package.

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

**bootstrap-sequence sub-command**

This command shows an ordered list of packages that must be installed to install the given package.

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

The asterisks symbolize pre-dependencies.

**all-priority**

Lists all packages with a given priority (also works with the special-case of `Essential: yes`)

```shell
> debdep all-priority essential

000 libc-bin
001 sed
002 util-linux
003 gzip
004 ncurses-bin
005 bash
006 diffutils
007 bsdutils
008 base-passwd
009 perl-base
010 tar
011 grep
012 debianutils
013 login
014 coreutils
015 sysvinit-utils
016 findutils
017 dash
018 base-files
019 hostname
020 dpkg
021 ncurses-base
022 init-system-helpers
```

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
