// Package scan walks a directory tree and reports the largest files and
// largest directories found, up to a configurable depth.
package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type FileEntry struct {
	Path string
	Size int64
}

type DirNode struct {
	Path   string
	Parent string
	Depth  int
	Size   int64 // cumulative, filled in after aggregation
}

// Scan walks root, returning every regular file found and every directory
// (with cumulative size and depth relative to root). Directories whose base
// name is in exclude are skipped entirely.
func Scan(root string, exclude map[string]bool) ([]FileEntry, map[string]*DirNode, error) {
	dirs := make(map[string]*DirNode)
	var dirOrder []string
	var files []FileEntry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			if d == nil {
				// Failed to stat path itself (most commonly root): nothing
				// to continue walking, so propagate the error.
				return err
			}
			if d.IsDir() {
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
			dirs[path] = &DirNode{Path: path, Parent: parent, Depth: depth}
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
		files = append(files, FileEntry{Path: path, Size: size})

		parent := filepath.Dir(path)
		if node, ok := dirs[parent]; ok {
			node.Size += size
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
		if node.Path == root {
			continue
		}
		if parent, ok := dirs[node.Parent]; ok {
			parent.Size += node.Size
		}
	}

	return files, dirs, nil
}

func HumanSize(n int64) string {
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
