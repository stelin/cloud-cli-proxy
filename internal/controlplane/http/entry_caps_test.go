package http

import "testing"

// TestDeriveEntryCapabilities 覆盖 Phase 30 D-06/D-07：
// - image_version 来自 template_image_ref 最后一个 ":" 后的 tag（无 ":" 则整串），仅 trim 空白；
// - supports_mergerfs = imageLockSupportsMergerfs || (image_version == "v3.0.0")；
// - image.lock 显式声明为 true 时优先覆盖 tag 推导，解决重建后 DB 字段未同步问题。
func TestDeriveEntryCapabilities(t *testing.T) {
	tests := []struct {
		name                      string
		ref                       string
		imageLockSupportsMergerfs bool
		wantVersion               string
		wantMergerfs              bool
	}{
		{
			name:         "v3.0.0 tag unlocks mergerfs",
			ref:          "ghcr.io/example/cloud-claude:v3.0.0",
			wantVersion:  "v3.0.0",
			wantMergerfs: true,
		},
		{
			name:         "v2.0.0 tag keeps caps false",
			ref:          "ghcr.io/example/cloud-claude:v2.0.0",
			wantVersion:  "v2.0.0",
			wantMergerfs: false,
		},
		{
			name:         "pre-release tag is not treated as v3.0.0",
			ref:          "ghcr.io/example/cloud-claude:v3.0.0-rc1",
			wantVersion:  "v3.0.0-rc1",
			wantMergerfs: false,
		},
		{
			name:         "missing colon falls back to whole string as version",
			ref:          "cloudclaude-image",
			wantVersion:  "cloudclaude-image",
			wantMergerfs: false,
		},
		{
			name:         "whitespace is trimmed",
			ref:          "  ghcr.io/example/cloud-claude:v3.0.0  ",
			wantVersion:  "v3.0.0",
			wantMergerfs: true,
		},
		{
			name:         "empty ref yields empty version and false caps",
			ref:          "",
			wantVersion:  "",
			wantMergerfs: false,
		},
		{
			name:         "registry with port does not confuse tag parsing",
			ref:          "registry.internal:5000/cloud-claude:v3.0.0",
			wantVersion:  "v3.0.0",
			wantMergerfs: true,
		},
		{
			name:                      "image.lock true overrides old tag",
			ref:                       "ghcr.io/example/cloud-claude:v2.0.0",
			imageLockSupportsMergerfs: true,
			wantVersion:               "v2.0.0",
			wantMergerfs:              true,
		},
		{
			name:                      "image.lock true works with any ref",
			ref:                       "cloudclaude-image",
			imageLockSupportsMergerfs: true,
			wantVersion:               "cloudclaude-image",
			wantMergerfs:              true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			version, mergerfs := deriveEntryCapabilities(tc.ref, tc.imageLockSupportsMergerfs)
			if version != tc.wantVersion {
				t.Errorf("image_version = %q, want %q", version, tc.wantVersion)
			}
			if mergerfs != tc.wantMergerfs {
				t.Errorf("supports_mergerfs = %v, want %v", mergerfs, tc.wantMergerfs)
			}
		})
	}
}
