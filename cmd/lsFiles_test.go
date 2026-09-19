package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/octocov/coverage"
)

func TestLsFilesRows(t *testing.T) {
	tests := []struct {
		name           string
		files          coverage.FileCoverages
		scope          string
		want           []lsFilesRow
		wantUnresolved []string
	}{
		{
			// The report a repository produces names its files in whatever shape the tool
			// that wrote it chose, and NormalizePaths is what reconciles that to the
			// repository. Every entry it resolved is listed, whatever the others look like.
			"every resolved file is listed at the root",
			coverage.FileCoverages{
				{File: "example.com/m/cmd/app/main.go", NormalizedPath: "cmd/app/main.go", Covered: 1, Total: 1},
				{File: "example.com/m/lib.go", NormalizedPath: "lib.go", Covered: 1, Total: 2},
			},
			".",
			[]lsFilesRow{
				{path: "cmd/app/main.go", covered: 1, total: 1},
				{path: "lib.go", covered: 1, total: 2},
			},
			nil,
		},
		{
			// Sharing a basename with an unrelated file used to decide, through the order
			// paths sorted in on disk, which entries survived. Nothing about one entry may
			// move another.
			"a file sharing its basename with another does not displace it",
			coverage.FileCoverages{
				{File: "example.com/m/cmd/app/main.go", NormalizedPath: "cmd/app/main.go", Covered: 1, Total: 1},
				{File: "example.com/m/_examples/demo/main.go", NormalizedPath: "_examples/demo/main.go", Covered: 0, Total: 1},
				{File: "example.com/m/lib.go", NormalizedPath: "lib.go", Covered: 1, Total: 2},
			},
			".",
			[]lsFilesRow{
				{path: "cmd/app/main.go", covered: 1, total: 1},
				{path: "_examples/demo/main.go", covered: 0, total: 1},
				{path: "lib.go", covered: 1, total: 2},
			},
			nil,
		},
		{
			"a subdirectory lists what sits in it, relative to it",
			coverage.FileCoverages{
				{File: "cmd/app/main.go", NormalizedPath: "cmd/app/main.go", Covered: 1, Total: 1},
				{File: "cmd/app/run/run.go", NormalizedPath: "cmd/app/run/run.go", Covered: 2, Total: 4},
				{File: "lib.go", NormalizedPath: "lib.go", Covered: 1, Total: 2},
			},
			"cmd/app",
			[]lsFilesRow{
				{path: "main.go", covered: 1, total: 1},
				{path: "run/run.go", covered: 2, total: 4},
			},
			nil,
		},
		{
			// A prefix comparison on the raw string reads cmd/apple as sitting in cmd/app.
			"a sibling whose name starts with the scope is not in it",
			coverage.FileCoverages{
				{File: "cmd/app/main.go", NormalizedPath: "cmd/app/main.go", Covered: 1, Total: 1},
				{File: "cmd/apple/main.go", NormalizedPath: "cmd/apple/main.go", Covered: 0, Total: 1},
			},
			"cmd/app",
			[]lsFilesRow{
				{path: "main.go", covered: 1, total: 1},
			},
			nil,
		},
		{
			// A directory does not contain itself, and listing such an entry relative to the
			// scope would leave it with no name at all.
			"a path equal to the scope is not below it",
			coverage.FileCoverages{
				{File: "cmd/app", NormalizedPath: "cmd/app", Covered: 1, Total: 1},
				{File: "cmd/app/main.go", NormalizedPath: "cmd/app/main.go", Covered: 1, Total: 1},
			},
			"cmd/app",
			[]lsFilesRow{
				{path: "main.go", covered: 1, total: 1},
			},
			nil,
		},
		{
			// Nothing on disk answered to it, so the path it is listed under is the one the
			// report wrote. Losing it silently is what sent this reading of ls-files wrong.
			"an entry no path was found for is still listed at the root",
			coverage.FileCoverages{
				{File: "example.com/m/gone.go", Covered: 0, Total: 3},
				{File: "lib.go", NormalizedPath: "lib.go", Covered: 1, Total: 2},
			},
			".",
			[]lsFilesRow{
				{path: "example.com/m/gone.go", covered: 0, total: 3},
				{path: "lib.go", covered: 1, total: 2},
			},
			nil,
		},
		{
			"an entry no path was found for is reported below the root",
			coverage.FileCoverages{
				{File: "example.com/m/gone.go", Covered: 0, Total: 3},
				{File: "cmd/app/main.go", NormalizedPath: "cmd/app/main.go", Covered: 1, Total: 1},
			},
			"cmd/app",
			[]lsFilesRow{
				{path: "main.go", covered: 1, total: 1},
			},
			[]string{"example.com/m/gone.go"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotUnresolved := lsFilesRows(tt.files, tt.scope)
			if diff := cmp.Diff(got, tt.want, cmp.AllowUnexported(lsFilesRow{})); diff != "" {
				t.Error(diff)
			}
			if diff := cmp.Diff(gotUnresolved, tt.wantUnresolved); diff != "" {
				t.Error(diff)
			}
		})
	}
}
