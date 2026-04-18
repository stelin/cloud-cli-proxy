package http

import "testing"

// TestDeriveEntryCapabilities 覆盖 Phase 30 D-06/D-07：
// - image_version 来自 template_image_ref 最后一个 ":" 后的 tag（无 ":" 则整串），仅 trim 空白；
// - supports_mutagen / supports_mergerfs 当且仅当 image_version == "v3.0.0" 时为 true；
// - 本阶段禁止引入任何额外的 tag 对照表。
func TestDeriveEntryCapabilities(t *testing.T) {
	tests := []struct {
		name            string
		ref             string
		wantVersion     string
		wantMutagen     bool
		wantMergerfs    bool
	}{
		{
			name:         "v3.0.0 tag unlocks mutagen and mergerfs",
			ref:          "ghcr.io/example/cloud-claude:v3.0.0",
			wantVersion:  "v3.0.0",
			wantMutagen:  true,
			wantMergerfs: true,
		},
		{
			name:         "v2.0.0 tag keeps caps false",
			ref:          "ghcr.io/example/cloud-claude:v2.0.0",
			wantVersion:  "v2.0.0",
			wantMutagen:  false,
			wantMergerfs: false,
		},
		{
			name:         "pre-release tag is not treated as v3.0.0",
			ref:          "ghcr.io/example/cloud-claude:v3.0.0-rc1",
			wantVersion:  "v3.0.0-rc1",
			wantMutagen:  false,
			wantMergerfs: false,
		},
		{
			name:         "missing colon falls back to whole string as version",
			ref:          "cloudclaude-image",
			wantVersion:  "cloudclaude-image",
			wantMutagen:  false,
			wantMergerfs: false,
		},
		{
			name:         "whitespace is trimmed",
			ref:          "  ghcr.io/example/cloud-claude:v3.0.0  ",
			wantVersion:  "v3.0.0",
			wantMutagen:  true,
			wantMergerfs: true,
		},
		{
			name:         "empty ref yields empty version and false caps",
			ref:          "",
			wantVersion:  "",
			wantMutagen:  false,
			wantMergerfs: false,
		},
		{
			name:         "registry with port does not confuse tag parsing",
			ref:          "registry.internal:5000/cloud-claude:v3.0.0",
			wantVersion:  "v3.0.0",
			wantMutagen:  true,
			wantMergerfs: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			version, mutagen, mergerfs := deriveEntryCapabilities(tc.ref)
			if version != tc.wantVersion {
				t.Errorf("image_version = %q, want %q", version, tc.wantVersion)
			}
			if mutagen != tc.wantMutagen {
				t.Errorf("supports_mutagen = %v, want %v", mutagen, tc.wantMutagen)
			}
			if mergerfs != tc.wantMergerfs {
				t.Errorf("supports_mergerfs = %v, want %v", mergerfs, tc.wantMergerfs)
			}
		})
	}
}
