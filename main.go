// Command go-fz walks a directory tree and reports the largest files and
// largest directories found, up to a configurable depth.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileEntry struct {
	path string
	size int64
}

type dirNode struct {
	path   string
	parent string
	depth  int
	size   int64 // cumulative, filled in after aggregation
}

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

	files, dirs, err := scan(absRoot, exclude)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmtSize := func(n int64) string {
		if *humanFlag {
			return humanSize(n)
		}
		return fmt.Sprintf("%d", n)
	}

	if *showDirs {
		var list []*dirNode
		for _, d := range dirs {
			if d.depth <= *maxDepth {
				list = append(list, d)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].size > list[j].size })
		if len(list) > *topN {
			list = list[:*topN]
		}
		fmt.Printf("Largest directories (depth <= %d):\n", *maxDepth)
		for _, d := range list {
			rel, _ := filepath.Rel(absRoot, d.path)
			if rel == "." {
				rel = "."
			}
			fmt.Printf("  %10s  %s\n", fmtSize(d.size), rel)
		}
		fmt.Println()
	}

	if *showFiles {
		sort.Slice(files, func(i, j int) bool { return files[i].size > files[j].size })
		list := files
		if len(list) > *topN {
			list = list[:*topN]
		}
		fmt.Println("Largest files:")
		for _, f := range list {
			rel, _ := filepath.Rel(absRoot, f.path)
			fmt.Printf("  %10s  %s\n", fmtSize(f.size), rel)
		}
	}
}

// scan walks root, returning every regular file found and every directory
// (with cumulative size and depth relative to root). Directories whose base
// name is in exclude are skipped entirely.
func scan(root string, exclude map[string]bool) ([]fileEntry, map[string]*dirNode, error) {
	dirs := make(map[string]*dirNode)
	var dirOrder []string
	var files []fileEntry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if path != root && exclude[d.Name()] {
				return fs.SkipDir
			}
			parent := filepath.Dir(path)
			depth := 0
			if path != root {
				rel, relErr := filepath.Rel(root, path)
				if relErr == nil {
					depth = strings.Count(rel, string(filepath.Separator)) + 1
				}
			}
			dirs[path] = &dirNode{path: path, parent: parent, depth: depth}
			dirOrder = append(dirOrder, path)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		size := info.Size()
		files = append(files, fileEntry{path: path, size: size})

		parent := filepath.Dir(path)
		if node, ok := dirs[parent]; ok {
			node.size += size
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Aggregate child sizes into parents. Descendants always precede their
	// ancestors when dirOrder (pre-order) is walked in reverse.
	for i := len(dirOrder) - 1; i >= 0; i-- {
		node := dirs[dirOrder[i]]
		if node.path == root {
			continue
		}
		if parent, ok := dirs[node.parent]; ok {
			parent.size += node.size
		}
	}

	return files, dirs, nil
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	units := "KMGTPE"
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), units[exp])
}
