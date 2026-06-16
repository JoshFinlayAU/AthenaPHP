// Package walker enumerates the PHP files of a project that should be encoded,
// skipping Composer's vendor tree and other directories that must stay readable.
package walker

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Options controls traversal.
type Options struct {
	// Exts is the set of file extensions (lowercase, with dot) to encode.
	Exts []string
	// SkipDirs are directory base names pruned anywhere in the tree.
	SkipDirs []string
	// SkipSuffixes are filename suffixes excluded from encoding even when the
	// extension matches, e.g. ".blade.php" (Blade templates are read as text by
	// Laravel's compiler, not via the Zend compiler, so must stay cleartext).
	SkipSuffixes []string
	// Extra are additional relative paths (files or dirs) to exclude.
	Extra []string
}

// DefaultOptions returns sensible defaults: encode .php, skip vendor & VCS dirs,
// and never encode Blade templates.
func DefaultOptions() Options {
	return Options{
		Exts:         []string{".php"},
		SkipDirs:     []string{"vendor", ".git", ".svn", ".hg", "node_modules"},
		SkipSuffixes: []string{".blade.php"},
	}
}

func (o Options) hasExt(name string) bool {
	lower := strings.ToLower(name)
	for _, s := range o.SkipSuffixes {
		if strings.HasSuffix(lower, strings.ToLower(s)) {
			return false
		}
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, e := range o.Exts {
		if ext == e {
			return true
		}
	}
	return false
}

func (o Options) isSkipDir(base string) bool {
	for _, d := range o.SkipDirs {
		if base == d {
			return true
		}
	}
	return false
}

// ShouldEncode reports whether the project-relative (slash-separated) path rel
// is a file that must be encoded: matching extension, not under a skipped dir,
// not explicitly excluded.
func (o Options) ShouldEncode(rel string) bool {
	if !o.hasExt(rel) {
		return false
	}
	parts := strings.Split(rel, "/")
	for _, p := range parts[:len(parts)-1] {
		if o.isSkipDir(p) {
			return false
		}
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	for _, e := range o.Extra {
		if filepath.ToSlash(filepath.Clean(e)) == clean {
			return false
		}
	}
	return true
}

// WalkAll returns every file under root (project-relative, slash-separated),
// without pruning. Used when mirroring a full tree into an output directory.
func WalkAll(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

// Walk returns project-relative paths (slash-separated) of files to encode.
// root is walked recursively; vendor/ and other SkipDirs are pruned.
func Walk(root string, o Options) ([]string, error) {
	extra := make(map[string]bool, len(o.Extra))
	for _, e := range o.Extra {
		extra[filepath.ToSlash(filepath.Clean(e))] = true
	}

	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if o.isSkipDir(d.Name()) || extra[rel] {
				return fs.SkipDir
			}
			return nil
		}
		if extra[rel] {
			return nil
		}
		if o.hasExt(d.Name()) {
			out = append(out, rel)
		}
		return nil
	})
	return out, err
}
