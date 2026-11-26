package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Use literal API asset URLs, independent of the application's repository
// constant: fixtures derived from that constant hid the transfer regression.
func transferredRelease(t *testing.T, trial bool) githubRelease {
	t.Helper()
	tag, asset := "v9.9.1", "ClipCatch.exe"
	if trial {
		tag, asset = "test2-v9.9.1-test.1", "ClipCatch-Test2.exe"
	}
	base := "https://github.com/SIWON-MARKETING/clipcatch/releases/download/" + tag + "/"
	data, err := json.Marshal(map[string]any{
		"tag_name": tag, "prerelease": trial,
		"assets": []map[string]any{
			{"name": asset, "size": 4000000, "browser_download_url": base + asset, "digest": "sha256:" + strings.Repeat("a", 64)},
			{"name": "SHA256SUMS", "browser_download_url": base + "SHA256SUMS"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rel githubRelease
	if err := json.Unmarshal(data, &rel); err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestTransferredReleaseAssetsStayStrict(t *testing.T) {
	for _, trial := range []bool{false, true} {
		channel := "stable"
		parse := parseGithubRelease
		if trial {
			channel, parse = "test", parseTrialRelease
		}
		t.Run(channel, func(t *testing.T) {
			old := appChannel
			appChannel = channel
			defer func() { appChannel = old }()
			got, err := parse(transferredRelease(t, trial))
			if err != nil || !releaseCacheValid(got) || !strings.HasPrefix(got.ReleaseURL, "https://github.com/SIWON-MARKETING/clipcatch/releases/tag/") {
				t.Fatalf("transferred official release rejected: %+v, %v", got, err)
			}
			for _, index := range []int{0, 1} {
				for _, replacement := range []string{
					"shindongyoon-siwon/clipcatch", "SIWON-MARKETING/other", "attacker/clipcatch",
					"SIWON-MARKETING/clipcatch.evil", "SIWON-MARKETING/clipcatch/..",
				} {
					bad := transferredRelease(t, trial)
					bad.Assets[index].BrowserDownloadURL = strings.Replace(bad.Assets[index].BrowserDownloadURL, "SIWON-MARKETING/clipcatch", replacement, 1)
					if _, err := parse(bad); err == nil {
						t.Errorf("accepted foreign/legacy asset %q", bad.Assets[index].BrowserDownloadURL)
					}
				}
				for _, suffix := range []string{"?download=1", "#asset", "/extra"} {
					bad := transferredRelease(t, trial)
					bad.Assets[index].BrowserDownloadURL += suffix
					if _, err := parse(bad); err == nil {
						t.Errorf("accepted modified asset %q", bad.Assets[index].BrowserDownloadURL)
					}
				}
			}
			if trial {
				if _, err := parseGithubRelease(transferredRelease(t, true)); err == nil {
					t.Fatal("stable accepted TEST2")
				}
			} else if _, err := parseTrialRelease(transferredRelease(t, false)); err == nil {
				t.Fatal("TEST2 accepted stable")
			}
		})
	}
}

func TestTransferLegacyHintCannotSupplyAnInstaller(t *testing.T) {
	for _, shareURL := range []string{
		"https://github.com/SIWON-MARKETING/clipcatch/releases/latest/download/ClipCatch.exe",
		"https://github.com/shindongyoon-siwon/clipcatch/releases/latest/download/ClipCatch.exe",
	} {
		data, _ := json.Marshal(buildIdentity{App: "clipcatch", Channel: "stable", Version: "9.9.1", ShareURL: shareURL})
		hint, err := parseStableUpdateHint(data)
		if err != nil || hint.Version != "9.9.1" {
			t.Fatalf("historical/current stable hint rejected: %v", err)
		}
		a := discoveryApp(t)
		a.latestRelease = discoveryRelease(t, appVersion)
		a.updateHooks.probe = func(context.Context) (updateHint, error) { return hint, nil }
		a.updateHooks.fetch = func(context.Context) (verifiedRelease, error) {
			return verifiedRelease{}, errors.New("official release unavailable")
		}
		runDiscovery(t, a)
		if a.updateRequiredLocked() || a.state.AppUpdate.Status == "available" {
			t.Fatal("hint bypassed official release verification")
		}
	}
	for _, shareURL := range []string{
		"https://github.com/attacker/clipcatch/releases/latest/download/ClipCatch.exe",
		"https://github.com/shindongyoon-siwon/clipcatch/releases/latest/download/ClipCatch-Test2.exe",
	} {
		data, _ := json.Marshal(buildIdentity{App: "clipcatch", Channel: "stable", Version: "9.9.1", ShareURL: shareURL})
		if _, err := parseStableUpdateHint(data); err == nil {
			t.Fatalf("untrusted hint accepted: %s", shareURL)
		}
	}
}

func TestTransferDiscardsLegacyInstallerCache(t *testing.T) {
	for _, cacheName := range []string{"required-update.json", "activity-update.json"} {
		t.Run(cacheName, func(t *testing.T) {
			a := discoveryApp(t)
			rel := discoveryRelease(t, "9.9.1")
			rel.BinaryURL = strings.Replace(rel.BinaryURL, "SIWON-MARKETING/clipcatch", "shindongyoon-siwon/clipcatch", 1)
			rel.SumsURL = strings.Replace(rel.SumsURL, "SIWON-MARKETING/clipcatch", "shindongyoon-siwon/clipcatch", 1)
			var data []byte
			if cacheName == "required-update.json" {
				data, _ = json.Marshal(rel)
			} else {
				now := time.Now()
				data, _ = json.Marshal(activityUpdateCache{Current: appVersion, Checked: now, Next: now.Add(time.Hour), Release: rel, Info: UpdateInfo{Status: "available"}})
			}
			if err := os.WriteFile(filepath.Join(a.dataDir, cacheName), data, 0600); err != nil {
				t.Fatal(err)
			}
			b := newApp(a.dataDir, t.TempDir())
			if b.latestRelease.Version != "" || !b.updateCheck.nextAutomatic.IsZero() {
				t.Fatal("legacy installer/cache cooldown survived transfer")
			}
		})
	}
}

func TestTransferKeepsPreviewOutsideStableAndTEST2Feeds(t *testing.T) {
	rel := transferredRelease(t, true)
	rel.TagName = "preview-v9.9.1-test.1"
	rel.Assets[0].Name = "ClipCatch-Preview-V2.exe"
	if _, err := parseGithubRelease(rel); err == nil {
		t.Fatal("stable accepted Preview")
	}
	if _, err := parseTrialRelease(rel); err == nil {
		t.Fatal("TEST2 accepted Preview")
	}
	for _, channel := range []string{"stable", "test"} {
		data, _ := json.Marshal(buildIdentity{App: "clipcatch-preview-v2", Channel: channel, Version: "9.9.1", ShareURL: stableShareURL})
		if _, err := parseStableUpdateHint(data); err == nil {
			t.Fatal("Preview identity disguised as update hint", channel)
		}
	}
}
