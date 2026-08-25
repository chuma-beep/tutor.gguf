// Package assets carries the small text corpora that ship inside the tutor
// binary so a fresh install can index without extra downloads.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:rosen
var embedded embed.FS

// Rosen returns the embedded Rosen discrete-math solutions (book exercises,
// README, and license texts).
func Rosen() fs.FS {
	sub, err := fs.Sub(embedded, "rosen")
	if err != nil {
		panic(err) // compile-time guaranteed by //go:embed all:rosen
	}
	return sub
}
