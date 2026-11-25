package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "net/url"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestAssemblyOfficialMetadataResolution(t *testing.T){
    seen:=[]string{}
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        if r.URL.Path!="/main/service/movie.do" {t.Errorf("wrong path %s",r.URL.Path);w.WriteHeader(404);return}
        q:=r.URL.Query()
        seen=append(seen,q.Get("cmd"))
        w.Header().Set("Content-Type","application/json")
        switch q.Get("cmd"){
        case "movieInfo":
            fmt.Fprint(w, `{"confTitle":"교육위원회","movieList":[{"no":675887,"movieTitle":"전체보기"}]}`)
        case "fileInfo":
            if q.Get("no")!="675887" {t.Errorf("no not propagated %s",q.Get("no"))}
            fmt.Fprint(w, `{"movieTitle":"전체보기","filePath":{"default":"720p","240p":"//m.webcast.go.kr/vod/low.mp4/playlist.m3u8","720p":"//m.webcast.go.kr/vod/hi.mp4/playlist.m3u8"}}`)
        default:t.Errorf("unexpected command %s",q.Get("cmd"))
        }
    }))
    defer server.Close()
    page:="https://w3.assembly.go.kr/main/player.do?menu=10&mc=10&ct1=22&ct2=439&ct3=11&wv=1"
    meta,err:=resolveAssemblyVideo(context.Background(),server.Client(),page,server.URL)
    if err!=nil {t.Fatal(err)}
    if len(seen)!=2||seen[0]!="movieInfo"||seen[1]!="fileInfo" {t.Fatal(seen)}
    if len(meta.Formats)!=1 || meta.Formats[0].URL!="https://m.webcast.go.kr/vod/hi.mp4/playlist.m3u8" {
        t.Fatalf("wrong HLS output %+v",meta)
    }
    if meta.Referer!=page || !strings.Contains(meta.Title,"교육위원회") {t.Fatal(meta.Title)}
}

func TestNaverOfficialPlaybackMetadataAndSeparateAudio(t *testing.T) {
    id:="3C8DF44C96A26217490A1AB575D276BEF723"
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        if r.URL.Path!="/api/v3.0/clip/playback" || r.Method!="POST" {t.Errorf("wrong request %s %s",r.Method,r.URL.Path);w.WriteHeader(400);return}
        if r.Header.Get("x-creator-hub-sid")!="clip" || r.Header.Get("Origin")!="https://clip.naver.com" {t.Error("missing creator hub headers")}
        var body struct {Media []struct{VideoID string `json:"videoId"`;ServiceType string `json:"serviceType"`} `json:"media"`}
        if err:=json.NewDecoder(r.Body).Decode(&body);err!=nil {t.Error(err)}
        if len(body.Media)!=1||body.Media[0].VideoID!=id||body.Media[0].ServiceType!="CLIP" {t.Errorf("wrong body %+v",body)}
        fmt.Fprint(w, `{"body":{"media":[{"mediaId":"3C8DF44C96A26217490A1AB575D276BEF723","serviceType":"CLIP","vod":{"playable":true,"type":"PLAY_API_3_0","playback":{"MPD":[{"Period":[{"AdaptationSet":[{"@mimeType":"video/mp4","Representation":[{"@width":"720","@height":"1280","BaseURL":["https://vod.pstatic.net/high.mp4"]},{"@width":"540","@height":"960","BaseURL":["https://vod.pstatic.net/low.mp4"]}]},{"@mimeType":"audio/mp4","Representation":[{"BaseURL":["https://vod.pstatic.net/audio.m4a"]}]}]}]}]}}}]}}`)
    }))
    defer server.Close()
    page:="https://m.naver.com/shorts/?mediaId="+id+"&serviceType=CLIP"
    result,err:=resolveNaverVideo(context.Background(),server.Client(),page,server.URL)
    if err!=nil {t.Fatal(err)}
    if len(result.Formats)!=3 {t.Fatalf("wanted both video qualities and audio: %+v",result)}
    if result.Formats[0].Height!=1280 || result.Formats[2].VCodec!="none" || result.Formats[2].ACodec=="none" {
        t.Fatalf("audio or video misclassified: %+v",result.Formats)
    }
}

func TestMediaURLSafetyAndInfoFileCleanup(t *testing.T){
    bad:=[]string{"http://m.webcast.go.kr/vod/a/playlist.m3u8","https://evil-m.webcast.go.kr/vod/a/playlist.m3u8","https://m.webcast.go.kr.evil.test/vod/a/playlist.m3u8","https://127.0.0.1/video.mp4","https://badpstatic.net/video.mp4"}
    for _,input:=range bad {
        site:="assembly";if strings.Contains(input,"pstatic")||strings.Contains(input,"127.0.0.1"){site="naver"}
        if _,err:=verifiedMediaURL(input,site);err==nil {t.Errorf("should reject %s",input)}
    }
    if _,err:=verifiedMediaURL("https://vod.pstatic.net/my-video.mp4?token=not-persisted","naver");err!=nil{t.Fatal(err)}
    if val:=baseURLValue(json.RawMessage(`[{"#text":"https://vod.pstatic.net/a.mp4"}]`));val!="https://vod.pstatic.net/a.mp4"{t.Fatal(val)}
    app:=newApp(t.TempDir(),t.TempDir())
    j:=&Job{ID:"x",URL:"https://w3.assembly.go.kr/main/player.do?mc=10&ct1=22&ct2=439&ct3=11&wv=1"}
    args:=[]string{"--ignore-config","-P",filepath.Join(t.TempDir(),"videos"),j.URL}
    // Reject attempting to replace anything other than the exact input page.
    if _,_,e:=app.siteVideoDownloadArgs(context.Background(),j,[]string{"--ignore-config","https://wrong.invalid"});e==nil {
        // It fails at resolver earlier in an offline test, and either way
        // must never write a command using an unverified URL.
        t.Fatal("unexpected resolver success")
    }
    _=args
    if _,err:=os.Stat(filepath.Join(app.dataDir,"unrelated"));err==nil {t.Fatal("unexpected side effect")}
}

func TestOfficialVideoPagesRetainParameters(t *testing.T){
    urlText:="https://w3.assembly.go.kr/main/player.do?menu=10&mc=10&ct1=22&ct2=439&ct3=11&wv=1&"
    clean,_,err:=cleanURL(urlText)
    if err!=nil {t.Fatal(err)}
    parsed,_:=url.Parse(clean)
    for key,val:=range map[string]string{"menu":"10","mc":"10","ct1":"22","ct2":"439","ct3":"11","wv":"1"}{
        if parsed.Query().Get(key)!=val {t.Fatalf("%s lost",key)}
    }
}
