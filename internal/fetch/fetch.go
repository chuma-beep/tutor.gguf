// Package fetch provides stdlib-only downloads with progress reporting and
// tar.gz/zip extraction, used by `tutor setup` to provision models,
// llama-server binaries, and corpus data.
package fetch

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// EnsureFile downloads url to dest unless dest already exists non-empty.
// It reports whether a download actually happened.
func EnsureFile(ctx context.Context, url, dest string) (bool, error) {
	if info, err := os.Stat(dest); err == nil && info.Size() > 0 {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return false, fmt.Errorf("create dir for %s: %w", dest, err)
	}

	tmp := dest + ".part"
	defer os.Remove(tmp) // no-op after successful rename

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	out, err := os.Create(tmp)
	if err != nil {
		return false, fmt.Errorf("create %s: %w", tmp, err)
	}
	progress := &progressWriter{total: resp.ContentLength}
	if _, err := io.Copy(io.MultiWriter(out, progress), resp.Body); err != nil {
		out.Close()
		return false, fmt.Errorf("download %s: %w", url, err)
	}
	if err := out.Close(); err != nil {
		return false, err
	}
	fmt.Println()
	if err := os.Rename(tmp, dest); err != nil {
		return false, err
	}
	return true, nil
}

type progressWriter struct {
	total int64
	n     int64
	last  int64
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.n += int64(len(b))
	if p.total > 0 && p.n-p.last > p.total/50 && p.n < p.total {
		p.last = p.n
		fmt.Printf("\r    %3d%%  (%s / %s)", p.n*100/p.total, human(p.n), human(p.total))
	}
	return len(b), nil
}

func human(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// ExtractTarGz extracts src into destDir. When keep is non-nil, entries whose
// archive-relative path fails keep(...) are skipped.
func ExtractTarGz(src, destDir string, keep func(relPath string) bool) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue // never write outside destDir
		}
		if keep != nil && !keep(name) {
			continue
		}
		target := filepath.Join(destDir, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode)
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(target), 0o755)
			_ = os.Remove(target)
			_ = os.Symlink(hdr.Linkname, target)
		}
	}
}

// CopyDir recursively copies the contents of src into dst, creating dst if
// needed. File modes are preserved.
func CopyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		return err
	})
}

// FindFileBelow returns the first regular file named name below root,
// excluding any path containing skip, or "" when absent.
func FindFileBelow(root, name, skip string) string {
	var found string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return err
		}
		if skip != "" && strings.Contains(path, skip) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && d.Name() == name {
			found = path
		}
		return nil
	})
	return found
}

// ExtractZip extracts src into destDir, setting the executable bit on entries
// that carry one in their external attributes (zip has no POSIX modes).
func ExtractZip(src, destDir string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, zf := range zr.File {
		name := filepath.Clean(zf.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}
		target := filepath.Join(destDir, name)
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			rc.Close()
			return err
		}
		mode := os.FileMode(0o644)
		if zf.Mode()&0o111 != 0 {
			mode = 0o755
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}
