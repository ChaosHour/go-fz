// Command go-fz walks a directory tree and reports the largest files and
// largest directories found, up to a configurable depth.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go-fz/internal/scan"
)

func main() {
	var (
		maxDepth   = flag.Int("depth", 3, "max depth (relative to root) of directories to report")
		topN       = flag.Int("top", 10, "number of results to show in each list")
		showFiles  = flag.Bool("files", true, "show largest files")
		showDirs   = flag.Bool("dirs", true, "show largest directories")
		excludeStr = flag.String("exclude", ".git", "comma-separated directory names to skip")
		humanFlag  = flag.Bool("human", true, "print human-readable sizes")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] [path]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Finds the largest files and largest directories under path (default \".\").\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	exclude := map[string]bool{}
	for name := range strings.SplitSeq(*excludeStr, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			exclude[name] = true
		}
	}

	files, dirs, err := scan.Scan(absRoot, exclude)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmtSize := func(n int64) string {
		if *humanFlag {
			return scan.HumanSize(n)
		}
		return fmt.Sprintf("%d", n)
	}

	if *showDirs {
		var list []*scan.DirNode
		for _, d := range dirs {
			if d.Depth <= *maxDepth {
				list = append(list, d)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Size > list[j].Size })
		if len(list) > *topN {
			list = list[:*topN]
		}
		fmt.Printf("Largest directories (depth <= %d):\n", *maxDepth)
		for _, d := range list {
			rel, _ := filepath.Rel(absRoot, d.Path)
			if rel == "." {
				rel = "."
			}
			fmt.Printf("  %10s  %s\n", fmtSize(d.Size), rel)
		}
		fmt.Println()
	}

	if *showFiles {
		sort.Slice(files, func(i, j int) bool { return files[i].Size > files[j].Size })
		list := files
		if len(list) > *topN {
			list = list[:*topN]
		}
		fmt.Println("Largest files:")
		for _, f := range list {
			rel, _ := filepath.Rel(absRoot, f.Path)
			fmt.Printf("  %10s  %s\n", fmtSize(f.Size), rel)
		}
	}
}
