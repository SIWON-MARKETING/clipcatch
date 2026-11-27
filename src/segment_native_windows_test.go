package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPreviewNativeCrossPC(t *testing.T) {
	if os.Getenv("CLIPCATCH_SEGMENT_NATIVE_TEST") != "1" {
		t.Skip("opt-in native Windows media test")
	}
	if runtime.GOOS != "windows" {
		t.Fatal("Native Windows is required")
	}
	ff, output := os.Getenv("CLIPCATCH_QA_FFMPEG"), os.Getenv("CLIPCATCH_QA_OUTPUT")
	if ff == "" || output == "" {
		t.Fatal("Missing real tool/evidence paths")
	}
	font, ok := installedPopFont()
	if !ok {
		t.Fatal("Official user-profile font was not detected")
	}
	fontData, err := os.ReadFile(font)
	if err != nil {
		t.Fatal(err)
	}
	fontHash := sha256.Sum256(fontData)
	if hex.EncodeToString(fontHash[:]) != "7e8c6c5d7bdf4b9e3f171b8e5577ab36cc780735d978db08006fae8306e6bae2" {
		t.Fatal("Different font selected")
	}
	previous := appChannel
	appChannel = "preview"
	defer func() { appChannel = previous }()
	root := t.TempDir()
	// Hosted Windows uses C: for temporary data and D: for this checkout.
	if strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(output)) {
		t.Fatal("Cross-drive regression test requires different volumes")
	}
	source := filepath.Join(root, "synthetic-with-audio.mp4")
	run := func(args ...string) []byte {
		t.Helper()
		data, err := exec.Command(ff, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("Real FFmpeg failed: %v %s", err, data)
		}
		return data
	}
	run("-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc2=size=640x360:rate=30",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000", "-t", "1.2", "-c:v", "libx264",
		"-preset", "ultrafast", "-pix_fmt", "yuv420p", "-c:a", "aac", source)
	a := newApp(filepath.Join(root, "data"), filepath.Join(output, "native-output"))
	if err := os.MkdirAll(a.dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		spec segmentSpec
	}{
		{"section", segmentSpec{Kind: "clip", Start: 0, End: 2}},
		{"carousel-basic", segmentSpec{Kind: "carousel", Start: 0, End: 2, Caption: "본문 테스트\n두 번째 줄"}},
		{"carousel-edited", segmentSpec{Kind: "carousel", Start: 0, End: 2, Caption: "본문 테스트\n두 번째 줄", Advanced: true, Size: 96, Color: "#ffff55", X: 50, Y: 75, Fit: "contain"}},
	}
	var reports []map[string]any
	var frames []string
	for i, c := range cases {
		if err := c.spec.validate(); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		var file string
		if c.spec.Kind == "clip" {
			file, err = a.renderSegmentClip(ctx, c.name, ff, source, &c.spec, 1.2, true)
		} else {
			file, err = a.renderSegmentCarousel(ctx, c.name, ff, source, &c.spec, 1.2, true)
		}
		if err != nil {
			cancel()
			t.Fatalf("%s: %v", c.name, err)
		}
		if err := verifySegmentMP4(ctx, ff, file, 1.2, true); err != nil {
			cancel()
			t.Fatal(err)
		}
		cancel()
		probe, err := exec.Command(filepath.Join(filepath.Dir(ff), "ffprobe.exe"), "-v", "error", "-show_entries", "stream=codec_name,codec_type,width,height,profile", "-of", "json", file).Output()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(probe), `"codec_name": "h264"`) || !strings.Contains(string(probe), `"codec_name": "aac"`) {
			t.Fatalf("Wrong codec: %s", probe)
		}
		if i > 0 && (!strings.Contains(string(probe), `"width": 1080`) || !strings.Contains(string(probe), `"height": 1350`)) {
			t.Fatalf("Wrong carousel size: %s", probe)
		}
		pcm := run("-hide_banner", "-loglevel", "error", "-i", file, "-vn", "-t", "0.2", "-ac", "1", "-ar", "8000", "-f", "s16le", "-")
		audible := false
		for _, b := range pcm {
			if b != 0 {
				audible = true
				break
			}
		}
		if len(pcm) < 100 || !audible {
			t.Fatal("No audible decoded AAC samples")
		}
		if i > 0 {
			png := run("-hide_banner", "-loglevel", "error", "-i", file, "-frames:v", "1", "-f", "image2pipe", "-vcodec", "png", "-")
			h := sha256.Sum256(png)
			frames = append(frames, hex.EncodeToString(h[:]))
			if err := os.WriteFile(filepath.Join(output, c.name+".png"), png, 0600); err != nil {
				t.Fatal(err)
			}
		}
		reports = append(reports, map[string]any{"case": c.name, "h264_aac": true, "audible_samples": len(pcm), "edited": c.spec.Advanced, "font_size": c.spec.Size, "x": c.spec.X, "y": c.spec.Y})
	}
	if len(frames) != 2 || frames[0] == frames[1] {
		t.Fatal("Edited style did not change exported pixels")
	}
	data, err := json.MarshalIndent(map[string]any{"native_windows": true, "cross_drive_export": true, "real_ffmpeg": true, "same_official_pop_font": true, "basic_and_edited_exports_differ": true, "figma_asset_equivalence_tested": false, "cases": reports}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "native-media-results.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	// Fonts stay in the disposable profile and are never uploaded as evidence.
}
