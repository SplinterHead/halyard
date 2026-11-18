package agent

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
)

func TestBuildImageInfosDeduplicatesSameIDRepoTag(t *testing.T) {
	sharedID := "sha256:2fe266df07a8c9044ec2461381a9b14dd4f382d1ea5e429da2b19ce532798f83"
	otherID := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	summaryList := []image.Summary{
		{
			ID:       sharedID,
			RepoTags: []string{"bitping/bitpingd:latest", "bitping/bitpingd:latest@sha256:2fe266df07a8c9044ec2461381a9b14dd4f382d1ea5e429da2b19ce532798f83"},
			Size:     60100000,
			Created:  1700000000,
		},
		{
			ID:          sharedID,
			RepoTags:    nil,
			RepoDigests: []string{"bitping/bitpingd@sha256:2fe266df07a8c9044ec2461381a9b14dd4f382d1ea5e429da2b19ce532798f83"},
			Size:        60100000,
			Created:     1700000000,
		},
		{
			ID:       sharedID,
			RepoTags: []string{"bitping/bitpingd:1.0.0"},
			Size:     60100000,
			Created:  1700000000,
		},
		{
			ID:       otherID,
			RepoTags: []string{"bitping/bitpingd:latest"},
			Size:     55000000,
			Created:  1700000000,
		},
	}

	containers := []types.Container{
		{
			ID:      "c1",
			ImageID: sharedID,
			Image:   "bitping/bitpingd:latest@sha256:2fe266df07a8c9044ec2461381a9b14dd4f382d1ea5e429da2b19ce532798f83",
		},
	}

	got := buildImageInfos(summaryList, containers, "x86_64")
	if len(got) != 3 {
		t.Fatalf("expected 3 deduplicated image entries, got %d: %+v", len(got), got)
	}

	if got[0].ID != sharedID || got[0].Repository != "bitping/bitpingd" || got[0].Tag != "latest" || !got[0].InUse {
		t.Fatalf("unexpected first image entry: %+v", got[0])
	}
	if got[1].ID != sharedID || got[1].Repository != "bitping/bitpingd" || got[1].Tag != "1.0.0" || !got[1].InUse {
		t.Fatalf("unexpected second image entry (distinct tag on same ID must be preserved): %+v", got[1])
	}
	if got[2].ID != otherID || got[2].Repository != "bitping/bitpingd" || got[2].Tag != "latest" || got[2].InUse {
		t.Fatalf("unexpected third image entry (distinct ID with same tag must be preserved): %+v", got[2])
	}
}
