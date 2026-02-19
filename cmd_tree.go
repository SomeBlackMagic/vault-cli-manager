package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudfoundry-community/vaultkv"
	fmt "github.com/jhunt/go-ansi"
	"github.com/starkandwayne/safe/rc"
	"github.com/starkandwayne/safe/vault"
)

func registerTreeCommands(r *Runner, opt *Options) {
	r.Dispatch("versions", &Help{
		Summary: "Print information about the versions of one or more paths",
		Usage:   "safe versions PATH [PATHS...]",
		Type:    NonDestructiveCommand,
	}, func(command string, args ...string) error {
		rc.Apply(opt.UseTarget)
		v := connect(true)

		if len(args) == 0 {
			return fmt.Errorf("No paths given")
		}

		for i := range args {
			_, _, version := vault.ParsePath(args[i])
			if version > 0 {
				return fmt.Errorf("Specifying version to versions is not supported")
			}
			versions, err := v.Client().Versions(args[i])
			if vaultkv.IsNotFound(err) {
				err = vault.NewSecretNotFoundError(args[i])
			}
			if err != nil {
				return err
			}

			if len(args) > 1 {
				fmt.Printf("@B{%s}:\n", args[i])
			}

			tbl := table{}

			tbl.setHeader("version", "status", "created at")

			for j := range versions {
				//Destroyed needs to be first because things can come back as both deleted _and_ destroyed.
				// destroyed is objectively more interesting.
				statusString := "@G{alive}"
				if versions[j].Destroyed {
					statusString = "@R{destroyed}"
				} else if versions[j].Deleted {
					statusString = "@Y{deleted}"
				}

				createdAtString := "unknown"

				if !versions[j].CreatedAt.IsZero() {
					createdAtString = versions[j].CreatedAt.Local().Format(time.RFC822)
				}

				tbl.addRow(
					fmt.Sprintf("%d", versions[j].Version),
					fmt.Sprintf(statusString),
					createdAtString,
				)
			}

			tbl.print()

			if len(args) > 1 && i != len(args)-1 {
				fmt.Printf("\n")
			}
		}

		return nil
	})

	r.Dispatch("ls", &Help{
		Summary: "Print the keys and sub-directories at one or more paths",
		Usage:   "safe ls [-1|-q] [PATH ...]",
		Type:    NonDestructiveCommand,
		Description: `
	Specifying the -1 flag will print one result per line.
	Specifying the -q flag will show secrets which have been marked as deleted.
`,
	}, func(command string, args ...string) error {
		rc.Apply(opt.UseTarget)
		v := connect(true)
		display := func(paths []string) {
			if opt.List.Single {
				for _, s := range paths {
					if strings.HasSuffix(s, "/") {
						fmt.Printf("@B{%s}\n", s)
					} else {
						fmt.Printf("@G{%s}\n", s)
					}
				}
			} else {
				for _, s := range paths {
					if strings.HasSuffix(s, "/") {
						fmt.Printf("@B{%s}  ", s)
					} else {
						fmt.Printf("@G{%s}  ", s)
					}
				}
				fmt.Printf("\n")
			}
		}

		if len(args) == 0 {
			args = []string{"/"}
		}

		for _, path := range args {
			var paths []string
			if path == "" || path == "/" {
				generics, err := v.Mounts("generic")
				if err != nil {
					return err
				}
				kvs, err := v.Mounts("kv")
				if err != nil {
					return err
				}

				paths = append(generics, kvs...)
			} else {
				var err error
				paths, err = v.List(path)
				if err != nil {
					return err
				}
			}

			filteredPaths := []string{}
			if !opt.List.Quick {
				for i := range paths {
					if !strings.HasSuffix(paths[i], "/") {
						fullpath := path + "/" + vault.EscapePathSegment(paths[i])
						mountVersion, err := v.MountVersion(fullpath)
						if err != nil {
							return err
						}

						if mountVersion == 2 {
							_, err := v.Read(fullpath)
							if err != nil {
								if vault.IsNotFound(err) {
									continue
								}

								return err
							}
						}
					}
					filteredPaths = append(filteredPaths, paths[i])
				}
			} else {
				filteredPaths = paths
			}

			sort.Strings(filteredPaths)

			if len(args) != 1 {
				fmt.Printf("@C{%s}:\n", path)
			}
			display(filteredPaths)
			if len(args) != 1 {
				fmt.Printf("\n")
			}
		}
		return nil
	})

	r.Dispatch("tree", &Help{
		Summary: "Print a tree listing of one or more paths",
		Usage:   "safe tree [-d|-q|--keys] [PATH ...]",
		Type:    NonDestructiveCommand,
		Description: `
Walks the hierarchy of secrets stored underneath a given path, listing all
reachable name/value pairs and displaying them in a tree format.  If '-d' is
given, only the containing folders will be printed; this more concise output
can be useful when you're trying to get your bearings. If '-q' is given, safe
will not inspect each key in a v1 v2 mount backend to see if it has been marked
as deleted. This may cause keys which would 404 in an attempt to read them to
appear in the tree, but is often considerably quicker for larger vaults. This
flag does nothing for kv v1 mounts.
`,
	}, func(command string, args ...string) error {
		rc.Apply(opt.UseTarget)
		if opt.Tree.HideLeaves && opt.Tree.ShowKeys {
			return fmt.Errorf("Cannot specify both -d and --keys at the same time")
		}
		if len(args) == 0 {
			args = append(args, "secret")
		}
		r1, _ := regexp.Compile("^ ")
		r2, _ := regexp.Compile("^└")
		v := connect(true)
		for i, path := range args {
			secrets, err := v.ConstructSecrets(path, vault.TreeOpts{
				FetchKeys:           opt.Tree.ShowKeys,
				AllowDeletedSecrets: opt.Tree.Quick,
			})

			if err != nil {
				return err
			}
			lines := strings.Split(secrets.Draw(path, fmt.CanColorize(os.Stdout), !opt.Tree.HideLeaves), "\n")
			if i > 0 {
				lines = lines[1:] // Drop root '.' from subsequent paths
			}
			if i < len(args)-1 {
				lines = lines[:len(lines)-1]
			}
			for _, line := range lines {
				if i < len(args)-1 {
					line = r1.ReplaceAllString(r2.ReplaceAllString(line, "├"), "│")
				}
				fmt.Printf("%s\n", line)
			}
		}
		return nil
	})

	r.Dispatch("paths", &Help{
		Summary: "Print all of the known paths, one per line",
		Usage:   "safe paths [-q|--keys] PATH [PATH ...]",
		Type:    NonDestructiveCommand,
		Description: `
Walks the hierarchy of secrets stored underneath a given path, listing all
reachable name/value pairs and displaying them in a list. If '-q' is given,
safe will not inspect each key in a v1 v2 mount backend to see if it has been
marked as deleted. This may cause keys which would 404 in an attempt to read
them to appear in the tree, but is often considerably quicker for larger
vaults. This flag does nothing for kv v1 mounts.
`}, func(command string, args ...string) error {
		rc.Apply(opt.UseTarget)
		if len(args) < 1 {
			args = append(args, "secret")
		}
		v := connect(true)
		for _, path := range args {
			secrets, err := v.ConstructSecrets(path, vault.TreeOpts{
				FetchKeys:           opt.Paths.ShowKeys,
				AllowDeletedSecrets: opt.Paths.Quick,
				SkipVersionInfo:     !opt.Paths.ShowKeys,
			})
			if err != nil {
				return err
			}

			fmt.Printf(strings.Join(secrets.Paths(), "\n"))
			fmt.Printf("\n")
		}
		return nil
	})
}
