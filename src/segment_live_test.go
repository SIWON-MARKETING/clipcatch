package main

import (
    "context"
    "encoding/json"
    "bytes"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "time"
)

// Opt-in remote smoke test uses one public National Assembly sample.
// No cookies, DRM, account login, browser sessions, or full recording download.
// It validates the same queue/yt-dlp/FFmpeg path as the Preview UI.
func TestPreviewAssemblyLiveTrim(t *testing.T) {
    if os.Getenv("CLIPCATCH_SEGMENT_LIVE_TEST")!="1" {t.Skip("public-provider live video smoke test opt-in")}
    dlp,e:=exec.LookPath("yt-dlp");if e!=nil{t.Fatal(e)}
    ff,e:=exec.LookPath("ffmpeg");if e!=nil{t.Fatal(e)}
    root:=t.TempDir()
    a:=newApp(filepath.Join(root,"data"),filepath.Join(root,"downloads"))
    a.mu.Lock()
    a.state.Phase="ready"
    a.ytDlp=dlp
    a.ffmpeg=ff
    a.mu.Unlock()
    old:=appChannel
    appChannel="preview"
    defer func(){appChannel=old}()
    url:="https://w3.assembly.go.kr/main/player.do?menu=10&mc=10&ct1=22&ct2=439&ct3=11&wv=1"
    body,_:=json.Marshal(map[string]any{
      "url":url,"start_seconds":2,"end_seconds":10,"kind":"clip"})
    w:=httptest.NewRecorder()
    a.mux.ServeHTTP(w,httptest.NewRequest("POST","/api/segment",bytes.NewReader(body)))
    if w.Code!=202{t.Fatalf("cannot queue Assembly clip: %d %s",w.Code,w.Body.String())}
    var response struct{ JobID string `json:"job_id"` }
    if e:=json.Unmarshal(w.Body.Bytes(),&response);e!=nil{t.Fatal(e)}
    if response.JobID==""{t.Fatal("queued segment job missing id")}
    ctx,cancel:=context.WithTimeout(context.Background(),150*time.Second)
    defer cancel()
    done:=make(chan struct{})
    go func(){a.runDownload(response.JobID);close(done)}()
    select{
    case <-done:
    case <-ctx.Done():
        a.cancelDownloadByID(response.JobID)
        t.Fatal("live Assembly section fetch did not complete in time")
    }
    j:=a.getJob(response.JobID)
    if j==nil||j.Status!="done"{t.Fatalf("live section failed: status=%s code=%s title=%s detail=%s",j.Status,j.ErrorCode,j.ErrorTitle,j.Detail)}
    if !nonempty(j.File){t.Fatal("successful job without a file")}
    duration,_,e:=probeSegmentVideo(context.Background(),ff,j.File)
    if e!=nil||duration>13||duration<4{t.Fatalf("invalid selected output duration: %.2f %v",duration,e)}
    t.Logf("PASS: public Assembly recording 00:00:02–00:00:10 saved as %.2f sec video, path %s",duration,filepath.Base(j.File))
}
