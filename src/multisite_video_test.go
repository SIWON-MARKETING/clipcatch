package main

import (
    "strings"
    "os"
    "path/filepath"
    "testing"
)

func TestNewPublicVideoPages(t *testing.T) {
    good := []string{
        "https://m.naver.com/shorts/?mediaId=8A94DF43BEECAA92D927198B8C5F07A0D6FB&serviceType=CLIP",
        "https://clip.naver.com/contents?recType=CLIP_PC&seedMediaId=42C87377808E4ABD5FE30AF7884589909A7F",
        "https://tv.naver.com/v/60223053",
        "https://w3.assembly.go.kr/main/player.do?menu=30&mc=345&ct1=22&ct2=439&ct3=V2&wv=1&",
    }
    for _, raw := range good {
        u, host, err := cleanURL(raw)
        if err != nil || u == "" || host == "" {
            t.Errorf("must accept exact public video page %s: %v", raw, err)
        }
    }
    // Reject non-media URLs and lookalike hosts, including local network targets.
    bad := []string{
        "https://m.naver.com/shorts/",
        "https://m.naver.com/shorts/?mediaId=not-media",
        "https://m.naver.com/other?mediaId=8A94DF43BEECAA92D927198B8C5F07A0D6FB",
        "https://clip.naver.com/@someone",
        "https://clip.naver.com/contents?seedMediaId=not-media",
        "https://evil-clip.naver.com/contents?seedMediaId=42C87377808E4ABD5FE30AF7884589909A7F",
        "https://tv.naver.com/users",
        "https://w3.assembly.go.kr/main/player.do?mc=345&ct1=22&ct2=439",
        "https://w3.assembly.go.kr/main/player.do?mc=345&ct1=22&ct2=439&ct3=%3Cscript%3E",
        "https://evilw3.assembly.go.kr/main/player.do?mc=345&ct1=22&ct2=439&ct3=V2",
        "https://w3.assembly.go.kr/admin",
        "https://u:pw@m.naver.com/shorts/?mediaId=42C87377808E4ABD5FE30AF7884589909A7F",
        "http://127.0.0.1:8765/",
    }
    for _, raw := range bad {
        if u, _, err := cleanURL(raw); err == nil {
            t.Errorf("must reject %s, accepted %s", raw, u)
        }
    }
}

func TestNewVideoPageKeepsRequiredParameters(t *testing.T) {
    raw := "https://w3.assembly.go.kr/main/player.do?menu=30&mc=345&ct1=22&ct2=439&ct3=V2&wv=1&utm_source=share"
    clean, _, err := cleanURL(raw)
    if err != nil || strings.Contains(clean, "utm_source") || !strings.Contains(clean, "ct3=V2") || !strings.Contains(clean, "wv=1") {
        t.Fatalf("page parameters changed: %s, err=%v", clean, err)
    }
}

func TestTrialExtensionIsolatedFromStableSource(t *testing.T) {
    original := appChannel
    defer func(){ appChannel = original }()
    appChannel = "stable"
    stable := newApp(t.TempDir(), t.TempDir())
    stableFolder, err := stable.installBrowserExtension()
    if err != nil { t.Fatal(err) }
    stableWorker, err := os.ReadFile(filepath.Join(stableFolder, "background.js"))
    if err != nil { t.Fatal(err) }
    if !strings.Contains(string(stableWorker), "channel:'stable'") && !strings.Contains(string(stableWorker), "channel: 'stable'") {
        // Existing stable worker must retain the real production pairing checks.
        if !strings.Contains(string(stableWorker), "'stable'") { t.Fatal("stable extension identity overwritten") }
    }
    appChannel = "test"
    trial := newApp(t.TempDir(), t.TempDir())
    trialFolder, err := trial.installBrowserExtension()
    if err != nil { t.Fatal(err) }
    trialWorker, err := os.ReadFile(filepath.Join(trialFolder, "background.js"))
    if err != nil { t.Fatal(err) }
    if !strings.Contains(string(trialWorker), "'clipcatch-test2'") || !strings.Contains(string(trialWorker), "'test'") {
        t.Fatal("trial extension did not receive its own pairing identity")
    }
    trialPopup, err := os.ReadFile(filepath.Join(trialFolder, "popup.js"))
    if err != nil || !strings.Contains(string(trialPopup), "hello?.channel!=='test'") {
        t.Fatal("trial popup still uses the production channel")
    }
    manifest, err := os.ReadFile(filepath.Join(trialFolder, "manifest.json"))
    if err != nil || !strings.Contains(string(manifest), "ClipCatch Browser TEST2") {
        t.Fatal("trial browser extension is not identified as TEST2")
    }
}
