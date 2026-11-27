package main

import (
    "context"
    "io"
    "net/http"
    "os"
    "strings"
    "testing"
    "time"
)

func TestLiveOfficialSitePlaybackMetadata(t *testing.T) {
    if os.Getenv("CLIPCATCH_SITE_LIVE")!="1" {t.Skip("public-site network test opt-in only")}
    ctx,cancel:=context.WithTimeout(context.Background(),90*time.Second)
    defer cancel()
    client:=&http.Client{Timeout:20*time.Second}
    page:="https://w3.assembly.go.kr/main/player.do?menu=10&mc=10&ct1=22&ct2=439&ct3=11&wv=1"
    parliament,err:=resolveSiteVideo(ctx,page,client)
    if err!=nil {t.Fatalf("Assembly public playback metadata unavailable: %v",err)}
    if len(parliament.Formats)==0||!strings.HasSuffix(parliament.Formats[0].URL,".m3u8"){
        t.Fatalf("Assembly HLS format missing: format count %d",len(parliament.Formats))
    }
    req,e:=http.NewRequestWithContext(ctx,"GET",parliament.Formats[0].URL,nil)
    if e!=nil {t.Fatal(e)}
    req.Header.Set("Referer",page)
    resp,e:=client.Do(req)
    if e!=nil{t.Fatalf("Assembly public manifest unavailable: %v",e)}
    body,_:=io.ReadAll(io.LimitReader(resp.Body,256))
    resp.Body.Close()
    if resp.StatusCode!=200||!strings.HasPrefix(string(body),"#EXTM3U"){
        t.Fatalf("Assembly manifest not playable: HTTP %d",resp.StatusCode)
    }
    t.Logf("Assembly public 720p HLS manifest reachable; formats %d",len(parliament.Formats))
    clipURL:="https://m.naver.com/shorts/?mediaId=3C8DF44C96A26217490A1AB575D276BEF723&serviceType=CLIP"
    clip,e:=resolveSiteVideo(ctx,clipURL,client)
    if e!=nil {t.Fatalf("CreatorHub public playback metadata unavailable: %v",e)}
    videoCount,audioCount:=0,0
    for _,f:=range clip.Formats{
        if f.VCodec=="none" {audioCount++} else {videoCount++}
    }
    if videoCount==0 {t.Fatal("Naver playback returned no downloadable video representations")}
    t.Logf("Naver public CLIP playback verified; video representations %d, audio representations %d",videoCount,audioCount)
}
