package integration_test

import (
	"math/rand"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
)

// pushOCIImage produces an OCI-format image
// (application/vnd.oci.image.manifest.v1+json + OCIConfigJSON) rather than
// the Docker schema2 default used by testRegistry.pushImage. Kept next to
// its sole consumer instead of in the shared harness because nothing else
// needs OCI-format fixtures today.
func pushOCIImage(t *testing.T, r *testRegistry, repo, tag string, created time.Time) v1.Image {
	t.Helper()

	buf := make([]byte, 32)
	//nolint:gosec // math/rand is fine; we only need uniqueness, not secrecy.
	_, err := rand.New(rand.NewSource(nextSeed())).Read(buf)
	require.NoError(t, err, "generate random layer bytes")

	layer := static.NewLayer(buf, types.OCILayer)

	img, err := mutate.AppendLayers(empty.Image, layer)
	require.NoError(t, err, "append layer")

	img, err = mutate.CreatedAt(img, v1.Time{Time: created.UTC()})
	require.NoError(t, err, "set CreatedAt on image")

	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.ConfigMediaType(img, types.OCIConfigJSON)

	ref := r.ref(t, repo, tag)
	require.NoError(t, remote.Write(ref, img), "push OCI image %s", ref)

	mt, err := img.MediaType()
	require.NoError(t, err, "read media type")
	require.Equal(t, types.OCIManifestSchema1, mt,
		"pushed image should be OCI manifest schema 1; sanity check so the test doesn't accidentally validate schema2")

	return img
}

// TestOCIManifestCreated pins the behavioural contract that deckschrubber
// reads the Created timestamp from OCI-format image configs just as it does
// from Docker schema2 configs. The tool's tag-age classification must be
// driven by the real image creation time regardless of manifest wire format.
//
// Fixtures:
//   - old-oci:   OCI-format, CreatedAt = 100 days ago
//   - fresh-oci: OCI-format, CreatedAt =   5 days ago
//
// With -day 30 and -latest 0 the only deciding factor is the deadline.
// -dry short-circuits the delete calls so assertions run against log output.
// Expected: old-oci is "Marking tag as outdated"; fresh-oci is "Tag not
// outdated". A failure here means OCI configs are being read as the zero
// time (every tag would then be classified as outdated).
func TestOCIManifestCreated(t *testing.T) {
	r := startRegistry(t)
	repo := "app/oci"

	now := time.Now()
	pushOCIImage(t, r, repo, "old-oci", now.AddDate(0, 0, -100))
	pushOCIImage(t, r, repo, "fresh-oci", now.AddDate(0, 0, -5))

	output := r.runDeckschrubber(t,
		"-repos", "10",
		"-repo", "^app/",
		"-day", "30",
		"-latest", "0",
		"-dry",
	)

	require.Contains(t, output, `tag=old-oci`,
		"output should reference the old OCI tag; output:\n%s", output)
	require.Contains(t, output, `Marking tag as outdated`,
		"old-oci (100 days old) should be marked outdated; output:\n%s", output)

	require.Contains(t, output, `tag=fresh-oci`,
		"output should reference the fresh OCI tag; output:\n%s", output)
	require.Contains(t, output, `Tag not outdated`,
		"fresh-oci (5 days old) should NOT be marked outdated; output:\n%s", output)
}
