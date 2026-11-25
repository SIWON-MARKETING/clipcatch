package main

import (
    "strings"
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
