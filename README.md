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

*Calculate-deps* sub-command:

```shell

./debdep calculate-deps screen

Read 55937 packages.
 composite:
  composite:
   composite:
    composite:
     composite:
      package-dep: gcc-8-base (8.2.0-9)
      package-dep: libc6 (2.27-8)
     package-dep: libgcc1 (1:8.2.0-9)
    package-dep: libc6 (2.27-8)
   composite:
    composite:
     composite:
      composite:
       package-dep: libaudit-common (1:2.8.4-2)
       composite:
       composite:
        package-dep: libc6 (2.27-8)
        package-dep: libcap-ng0 (0.7.9-1)
      package-dep: libaudit1 (1:2.8.4-2)
     composite:
     package-dep: debconf (1.5.69)
    package-dep: libpam0g (1.1.8-3.8)
   composite:
    package-dep: libc6 (2.27-8)
    package-dep: libtinfo6 (6.1+20181013-1)
   composite:
    package-dep: libc6 (2.27-8)
    package-dep: libutempter0 (1.1.6-3)
  package-dep: screen (4.6.2-3)
```

*bootstrap-sequence* sub-command:

```shell

./debdep bootstrap-sequence screen

# Read 55937 packages.
000 gcc-8-base 8.2.0-9
001 libc6 2.27-8
002 libgcc1 1:8.2.0-9
003 libc6 2.27-8
004 libaudit-common 1:2.8.4-2
005 libc6 2.27-8
006 libcap-ng0 0.7.9-1
007 libaudit1 1:2.8.4-2
008 debconf 1.5.69
009 libpam0g 1.1.8-3.8
010 libc6 2.27-8
011 libtinfo6 6.1+20181013-1
012 libc6 2.27-8
013 libutempter0 1.1.6-3
014 screen 4.6.2-3
```

### In Go:

Setup (optional):

```go

// These are defaults at time of writing
debdep.SetCodename("buster")
debdep.SetDistribution("testing")
debdep.SetArch("amd64")
```

Read & Parse all packages in the Debian repositories.

```go

pkgs, err := debdep.Packages(true)
if err != nil {
  fmt.Fprintf(os.Stderr, "Error: %v\n", err)
  os.Exit(1)
}
fmt.Printf("Read %d packages.\n", len(pkgs.Packages))
```

Compute an install graph (assuming no packages installed)

```go

pkg, err := pkgs.InstallGraph("screen", &debdep.PackageInfo{})
if err != nil {
  fmt.Fprintf(os.Stderr, "Error: %v\n", err)
  os.Exit(1)
}
pkg.PrettyWrite(os.Stdout, 1) // Pretty-print the graph.
```

## Known issues

 * If the sub-dependency chains up to a dependency that is already in the resolved install graph, but with a different set of requirements, it will be duplicated.

## TODO
 * Fix known issues.
 * Support specifying which packages are already installed.
