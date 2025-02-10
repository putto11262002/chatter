package chatter

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// StatisFS is a wrapper around http.FileSystem that adds etag support, cache control support and fallback file.
// It can be used to serve React build files.
type StaticFS struct {
	http.FileSystem
	etags map[string]string
	// a map of globs to cache control headers
	cacheControl map[string]string
	fallbackFile string
}

// Open returns the file if found. Otherwise, it returns index.html.
func (fs StaticFS) Open(name string) (http.File, error) {
	f, err := fs.FileSystem.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to open index.html instead.
			return fs.FileSystem.Open(fs.fallbackFile)
		}
		return nil, err
	}
	return f, nil
}

// NewStaticFS returns a new StaticFS
func NewStaticFS(fs fs.FS, fallback string, cacheControl map[string]string) (*StaticFS, error) {
	// check if fallback exists
	if _, err := fs.Open(fallback); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("fallback file %s does not exist", fallback)
		}
		return nil, fmt.Errorf("opening fallback file %s: %w", fallback, err)
	}

	etags, err := calculateEtags(fs)
	if err != nil {
		return nil, fmt.Errorf("calculating etags: %w", err)
	}
	cc, err := expendCacheControl(fs, cacheControl)
	if err != nil {
		return nil, fmt.Errorf("expanding cache control paths: %w", err)
	}

	return &StaticFS{FileSystem: http.FS(fs), etags: etags, cacheControl: cc, fallbackFile: fallback}, nil
}

func calculateEtags(fsys fs.FS) (map[string]string, error) {
	etags := make(map[string]string)
	hasher := sha1.New()
	return etags, fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fp := p
		f, err := fsys.Open(fp)
		if err != nil {
			return fmt.Errorf("opening %s: %w", fp, err)
		}
		defer f.Close()
		_, err = io.Copy(hasher, f)
		defer hasher.Reset()
		if err != nil {
			return fmt.Errorf("hashing %s: %w", fp, err)
		}
		etags[fp] = fmt.Sprintf("%x", hasher.Sum(nil))
		return nil
	})
}

func expendCacheControl(fsys fs.FS, cacheControl map[string]string) (map[string]string, error) {
	expanded := make(map[string]string)

	return expanded, fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fp := p
		for glob, cc := range cacheControl {

			if matched, err := filepath.Match(glob, fp); err == nil && matched {
				expanded[fp] = cc
				return nil
			} else if err != nil {
				return fmt.Errorf("matching %s: %w", fp, err)
			}
		}
		return nil
	})

}

func (fs StaticFS) EtagMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// strip the leading slash
			if len(path) > 0 && path[0] == '/' {
				path = path[1:]
			}
			// check if the match exists if not set to fallback
			if _, ok := fs.etags[path]; !ok {
				path = fs.fallbackFile
			}

			if matched := r.Header.Get("If-None-Match"); matched != "" {
				if etag, ok := fs.etags[path]; ok && matched == etag {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}

			if etag, ok := fs.etags[path]; ok {
				w.Header().Set("Etag", etag)
				// always revalidate the cache
				if cc, ok := fs.cacheControl[path]; ok {
					w.Header().Set("Cache-Control", cc)
				}
			}

			next.ServeHTTP(w, r)
		})

	}
}
