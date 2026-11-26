package main

import (
    "context"
    "encoding/json"
    "fmt"
    "image/png"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
    "testing"
    "time"
)

func TestPreviewSegmentValidationAndArgs(t *testing.T){
    for _,tc:=range []struct{s segmentSpec;good bool}{
        {segmentSpec{Start:0,End:30,Kind:"clip"},true},
        {segmentSpec{Start:65,End:95,Kind:"carousel",Caption:"안녕하세요\n두 번째 줄"},true},
        {segmentSpec{Start:65,End:65,Kind:"clip"},false},
        {segmentSpec{Start:90,End:35,Kind:"clip"},false},
        {segmentSpec{Start:0,End:3700,Kind:"clip"},false},
        {segmentSpec{Start:0,End:35,Kind:"carousel",Caption:""},false},
        {segmentSpec{Start:1,End:35,Kind:"carousel",Caption:"본문",Advanced:true,Size:130,Color:"#ffffff",X:50,Y:70,Fit:"cover"},false},
    }{
        e:=tc.s.validate()
        if (e==nil)!=tc.good {t.Errorf("unexpected result %+v: %v",tc.s,e)}
        if tc.s.Kind=="carousel"&&tc.good && tc.s.Size!=76 {t.Errorf("basic style was not locked: %+v",tc.s)}
    }
    j:=&Job{ID:"123",URL:"https://w3.assembly.go.kr/main/player.do?mc=10&ct1=22&ct2=439&ct3=11",Segment:&segmentSpec{Start:62,End:92}}
    args:=segmentArgs(j,[]string{"--ignore-config","-o","video.%(ext)s",j.URL})
    joined:=strings.Join(args,"|")
    for _,want:=range []string{"--download-sections","*00:01:02-00:01:32","--force-keyframes-at-cuts","--remux-video","mp4","[123]"}{
        if !strings.Contains(joined,want)&&want!="[123]" {t.Errorf("missing %q in %s",want,joined)}
    }
    if args[len(args)-1]!=j.URL {t.Fatal("URL must be last for first-party Assembly resolver")}
    if strings.Contains(joined,"video.%(ext)s"){t.Fatal("trim must have its own output name")}
    if len(segmentArgs(&Job{URL:j.URL},[]string{j.URL}))!=1 {t.Fatal("full video args changed")}
}

func TestBridgeBodyDynamicPosition(t *testing.T) {
    one:=&segmentSpec{Kind:"carousel",Caption:"안녕하세요"}
    two:=&segmentSpec{Kind:"carousel",Caption:"안녕하세요\n두 번째 줄"}
    if e:=one.validate();e!=nil{t.Fatal(e)}
    if e:=two.validate();e!=nil{t.Fatal(e)}
    size1,n1,y1,p1,e:=segmentBodyLayout(one)
    if e!=nil{t.Fatal(e)}
    size2,n2,y2,p2,e:=segmentBodyLayout(two)
    if e!=nil{t.Fatal(e)}
    if size1!=76||size2!=76||p2>=p1||y2>=y1||n2>=n1 {
        t.Fatalf("body must expand upward by lines: 1(%d,%d,%d,%d) 2(%d,%d,%d,%d)",size1,n1,y1,p1,size2,n2,y2,p2)
    }
    if y2<800||y2>1250||p2<750 {t.Fatalf("body no longer lower card: y=%d photo=%d",y2,p2)}
    shade:=filepath.Join(t.TempDir(),"shade.png")
    if e:=makeSectionShade(shade,p2);e!=nil{t.Fatal(e)}
    f,e:=os.Open(shade);if e!=nil{t.Fatal(e)};defer f.Close()
    img,e:=png.Decode(f);if e!=nil{t.Fatal(e)}
    if img.Bounds().Dx()!=1080||img.Bounds().Dy()!=1350 {t.Fatal("wrong gradient output size")}
}

func TestPreviewSegmentAPIIsIsolatedAndRedditUsesExtensionOnly(t *testing.T) {
    old:=appChannel;appChannel="preview";defer func(){appChannel=old}()
    a:=newApp(t.TempDir(),t.TempDir())
    a.state.Phase="ready"
    action:=func(body string) *httptest.ResponseRecorder{
        w:=httptest.NewRecorder()
        a.mux.ServeHTTP(w,httptest.NewRequest("POST","/api/segment",strings.NewReader(body)))
        return w
    }
    valid:=`{"url":"https://w3.assembly.go.kr/main/player.do?menu=10&mc=10&ct1=22&ct2=439&ct3=11&wv=1","start_seconds":30,"end_seconds":60,"kind":"clip"}`
    w:=action(valid)
    if w.Code!=202 {t.Fatalf("could not queue valid clip: %d %s",w.Code,w.Body.String())}
    if len(a.state.Jobs)!=1||a.state.Jobs[0].Segment==nil||a.state.Jobs[0].Segment.End!=60 {
        t.Fatal("segment not added to existing worker queue")
    }
    w=action(`{"url":"https://www.reddit.com/r/aww/comments/1wilsax/puppy/","start_seconds":1,"end_seconds":5,"kind":"clip"}`)
    if w.Code!=400 {t.Fatalf("Reddit should require extension: %d",w.Code)}
    w=action(`{"url":"http://127.0.0.1:5678/secret.mp4","start_seconds":1,"end_seconds":5,"kind":"clip"}`)
    if w.Code!=400 {t.Fatalf("loopback URL must never be fetched: %d",w.Code)}
    appChannel="stable"
    stable:=newApp(t.TempDir(),t.TempDir())
    blocked:=httptest.NewRecorder()
    stable.mux.ServeHTTP(blocked,httptest.NewRequest("POST","/api/segment",strings.NewReader(valid)))
    if blocked.Code!=404 {t.Fatalf("stable must not expose test endpoint: %d",blocked.Code)}
}

func TestSegmentFFmpegLocalCarouselEndToEnd(t *testing.T) {
    if os.Getenv("CLIPCATCH_SEGMENT_MEDIA_TEST")!="1" {t.Skip("run with local FFmpeg and a test-only font")}
    if runtime.GOOS=="windows" {t.Skip("CI media fixture renders on Linux; Windows EXE separately smoke-tested")}
    ff,e:=exec.LookPath("ffmpeg");if e!=nil {t.Skip("FFmpeg unavailable")}
    font:=os.Getenv("CLIPCATCH_PREVIEW_FONT");if !validLocalFont(font){t.Skip("no test font installed")}
    root:=t.TempDir();t.Setenv("CLIPCATCH_PREVIEW_FONT",font)
    input:=filepath.Join(root,"fixture.mp4")
    cmd:=exec.Command(ff,"-v","error","-f","lavfi","-i","testsrc2=size=640x360:rate=30",
      "-f","lavfi","-i","sine=frequency=440:sample_rate=48000",
      "-t","1.2","-c:v","libx264","-preset","ultrafast","-pix_fmt","yuv420p",
      "-c:a","aac","-y",input)
    out,e:=cmd.CombinedOutput();if e!=nil{t.Fatalf("fixture: %v: %s",e,out)}
    a:=newApp(filepath.Join(root,"data"),filepath.Join(root,"output"))
    if e:=os.MkdirAll(a.dataDir,0700);e!=nil{t.Fatal(e)}
    spec:=&segmentSpec{Kind:"carousel",Caption:"TEST\nPOP BODY"}
    if e:=spec.validate();e!=nil{t.Fatal(e)}
    ctx,cancel:=context.WithTimeout(context.Background(),time.Minute);defer cancel()
    file,e:=a.renderSegmentCarousel(ctx,"test-123",ff,input,spec,1.2,true)
    if e!=nil{t.Fatalf("carousel rendering failed: %v",e)}
    if _,e:=os.Stat(file);e!=nil{t.Fatal(e)}
    probe:=exec.Command(filepath.Join(filepath.Dir(ff),"ffprobe"),
      "-v","error","-show_entries","stream=codec_name,codec_type,width,height","-of","json",file)
    data,e:=probe.Output();if e!=nil{t.Fatal(e)}
    var result struct{Streams []struct{
        Codec,Type string `json:"-"`
        Width int `json:"width"`
        Height int `json:"height"`
    } `json:"streams"`}
    _=json.Unmarshal(data,&result)
    if !strings.Contains(string(data),`"codec_name": "h264"`) ||
      !strings.Contains(string(data),`"codec_name": "aac"`) ||
      !strings.Contains(string(data),`"width": 1080`) ||
      !strings.Contains(string(data),`"height": 1350`) {
       t.Fatalf("wrong 1080x1350 H264 AAC output: %s",data)
    }
    fmt.Println("PASS: 1.2-second local source -> 1080x1350 H264/AAC video file")
}
