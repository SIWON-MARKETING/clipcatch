package main

import (
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestPortablePreviewHasOwnAddressAndFiles(t *testing.T) {
    old := appChannel
    defer func() { appChannel = old }()
    root := t.TempDir()
    t.Setenv("LOCALAPPDATA", root)
    t.Setenv("CLIPCATCH_SAVEDIR", filepath.Join(root, "stable-unchanged"))
    t.Setenv("CLIPCATCH_TEST2_SAVEDIR", filepath.Join(root, "test2-unchanged"))
    t.Setenv("CLIPCATCH_PREVIEW_SAVEDIR", "")

    appChannel = "preview"
    if !isPreviewBuild() || !isTrialBuild() {
        t.Fatal("preview must be a trial but separate from TEST2")
    }
    if runtimeAppID() != "clipcatch-preview-v2" || runtimePort() != "18768" || runtimeDataName() != "ClipCatch-Preview-V2" {
        t.Fatalf("preview identity wrong: %s %s %s", runtimeAppID(), runtimePort(), runtimeDataName())
    }
    if runtimeSaveDir(root) != filepath.Join(root, "Downloads", "ClipCatch-Preview-V2") {
        t.Fatal("preview reused a stable or TEST2 save path")
    }
    if strings.Contains(strings.ToLower(installedExePath()), "test2") || !strings.Contains(strings.ToLower(installedExePath()), "preview") {
        t.Fatal("preview updater directory overlaps TEST2")
    }
    if installAndRelaunch() {
        t.Fatal("preview must never invoke install-and-relaunch")
    }
    if err := validateBuildIdentity(buildIdentity{App: runtimeAppID(), Channel: appChannel, Version: "2.0.1-test.8"}); err != nil {
        t.Fatal("preview build identity rejected", err)
    }
    a := newApp(filepath.Join(root, "preview"), filepath.Join(root, "save"))
    h := httptest.NewRecorder()
    a.mux.ServeHTTP(h, httptest.NewRequest("GET", "/api/health", nil))
    if h.Code != 200 || !strings.Contains(h.Body.String(), "clipcatch-preview-v2") {
        t.Fatalf("preview health missing: %d %s", h.Code, h.Body.String())
    }
    if check, _ := a.beginAppUpdateCheck(false, false); check != nil {
        t.Fatal("preview must not attempt in-app upgrades to TEST2 or stable releases")
    }
    if a.updateRequiredLocked() {
        t.Fatal("preview must never gate new downloads for released stable binaries")
    }

    // Switch back in the same test; production and older TEST2 remain unchanged.
    appChannel = "stable"
    if runtimePort() != "8765" || runtimeDataName() == "ClipCatch-Preview-V2" {
        t.Fatal("production startup identity changed")
    }
    appChannel = "test"
    if runtimePort() != "18766" || runtimeAppID() != "clipcatch-test2" {
        t.Fatal("older TEST2 startup identity changed")
    }
    if _, err := os.Stat(filepath.Join(root, "stable-unchanged")); !os.IsNotExist(err) {
        t.Fatal("preview modified stable directory")
    }
}
