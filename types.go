package main

import (
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// Image represents a docker image with a specific tag.
//
// Digest is the manifest digest the tag resolves to; we keep it as v1.Hash
// rather than a full remote.Descriptor because the only thing the GC logic
// needs from the descriptor is the digest (used for identity comparisons
// between tags and for the Delete/Tag wire calls).
type Image struct {
	Repository string    // Name of repository to which image belongs
	Tag        string    // Image's tag
	Time       time.Time // Creation time of the image (from the image config's `created` field)
	Digest     v1.Hash   // Manifest digest the tag currently points at
}

// ImageByDate represents an array of images
// sorted by creation date
type ImageByDate []Image

func (ibd ImageByDate) Len() int           { return len(ibd) }
func (ibd ImageByDate) Swap(i, j int)      { ibd[i], ibd[j] = ibd[j], ibd[i] }
func (ibd ImageByDate) Less(i, j int) bool { return ibd[i].Time.Before(ibd[j].Time) }
