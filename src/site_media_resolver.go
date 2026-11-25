package main

// Public site playback metadata resolvers. They fetch only site-owned
// metadata and hand verified HTTPS media URLs to the EXISTING yt-dlp pipeline.
// No cookies, browser sessions, DRM keys or unrestricted URL fetching.
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "time"
)

type siteVideoFormat struct {
    FormatID string            `json:"format_id"`
    URL string                 `json:"url"`
    Ext string                 `json:"ext"`
    Protocol string            `json:"protocol,omitempty"`
    VCodec string              `json:"vcodec,omitempty"`
    ACodec string              `json:"acodec,omitempty"`
    Width int                  `json:"width,omitempty"`
    Height int                 `json:"height,omitempty"`
    TBR int                    `json:"tbr,omitempty"`
    HTTPHeaders map[string]string `json:"http_headers,omitempty"`
}

type siteVideoMetadata struct {
    ID string
    Title string
    WebpageURL string
    Referer string
    Formats []siteVideoFormat
}

const siteVideoUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/129.0 Safari/537.36"

func isSpecialVideoPage(raw string) bool {
    u, e := url.Parse(raw)
    if e != nil { return false }
    switch u.Hostname() {
    case "w3.assembly.go.kr":
        return u.Path == "/main/player.do"
    case "m.naver.com":
        return u.Path == "/shorts" || u.Path == "/shorts/"
    case "clip.naver.com":
        return u.Path == "/contents" || u.Path == "/contents/"
    default: return false
    }
}

func fetchSiteJSON(ctx context.Context, client *http.Client, method, endpoint, referer string, body io.Reader, out any) error {
    req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
    if err != nil { return errors.New("영상 정보 요청을 만들지 못했어") }
    req.Header.Set("User-Agent", siteVideoUA)
    req.Header.Set("Accept", "application/json, text/javascript, */*")
    req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9")
    if referer != "" { req.Header.Set("Referer", referer) }
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Origin", "https://clip.naver.com")
        req.Header.Set("x-creator-hub-sid", "clip")
    }
    resp, err := client.Do(req)
    if err != nil { return errors.New("영상 정보 서버에 연결하지 못했어") }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("영상 정보 서버 응답 %d", resp.StatusCode)
    }
    if len(resp.Header.Values("Content-Length")) > 0 && resp.ContentLength > 2<<20 {
        return errors.New("영상 정보가 지나치게 커")
    }
    decoder := json.NewDecoder(io.LimitReader(resp.Body, (2<<20)+1))
    decoder.UseNumber()
    if err := decoder.Decode(out); err != nil { return errors.New("영상 정보 응답을 읽지 못했어") }
    return nil
}

func verifiedMediaURL(raw, source string) (string, error) {
    if strings.HasPrefix(raw, "//") { raw = "https:" + raw }
    if len(raw) > 8192 { return "", errors.New("영상 주소가 너무 길어") }
    u, err := url.Parse(raw)
    if err != nil || u.User != nil || u.Scheme != "https" || u.Port() != "" || u.Hostname() == "" {
        return "", errors.New("허용되지 않는 영상 주소 형식이야")
    }
    host := strings.ToLower(u.Hostname())
    if netIP := regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`).MatchString(host); netIP {
        return "", errors.New("영상 주소가 공개 CDN이 아니야")
    }
    if source == "assembly" {
        if host != "m.webcast.go.kr" || !strings.HasPrefix(u.Path, "/vod/") || !strings.HasSuffix(strings.ToLower(u.Path), ".m3u8") {
            return "", errors.New("국회 공식 영상 서버 주소가 아니야")
        }
    } else {
        allowed := []string{"pstatic.net","naver.com","naver.net","nvideo.net","nvcdn.net","pstatics.net","akamaized.net"}
        ok := false
        for _, root := range allowed {
            if host == root || strings.HasSuffix(host, "."+root) { ok = true; break }
        }
        if !ok { return "", errors.New("네이버 공식 영상 서버 주소가 아니야") }
    }
    u.Fragment = ""
    return u.String(), nil
}

// Make public National Assembly player URL resolve as its official player does:
// movieInfo yields the recording number, fileInfo yields named HLS qualities.
func resolveAssemblyVideo(ctx context.Context, client *http.Client, page, root string) (siteVideoMetadata, error) {
    var out siteVideoMetadata
    u, err := url.Parse(page)
    if err != nil { return out, errors.New("국회 영상 주소를 읽지 못했어") }
    q := url.Values{}
    for _, k := range []string{"mc","ct1","ct2","ct3","no","wv"} {
        if value:=u.Query().Get(k);value!="" {q.Set(k,value)}
    }
    if q.Get("wv")=="" { q.Set("wv","1") }
    mcRE := regexp.MustCompile("^[A-Za-z0-9]{1,12}$")
    digits := regexp.MustCompile("^[0-9]+$")
    if !mcRE.MatchString(q.Get("mc")) || !digits.MatchString(q.Get("ct1")) ||
        !digits.MatchString(q.Get("ct2")) || !mcRE.MatchString(q.Get("ct3")) ||
        !digits.MatchString(q.Get("wv")) {
        return out, errors.New("국회 영상 회의 정보가 올바르지 않아")
    }
    if q.Get("no")!="" && !digits.MatchString(q.Get("no")) {
        return out, errors.New("국회 영상 녹화 번호가 올바르지 않아")
    }
    q.Set("cmd","movieInfo")
    q.Set("vv",strconv.FormatInt(time.Now().Unix(),10))
    var listing struct {
        ConfTitle string `json:"confTitle"`
        MovieList []struct {
            No json.Number `json:"no"`
            MovieTitle string `json:"movieTitle"`
        } `json:"movieList"`
    }
    endpoint := strings.TrimRight(root,"/") + "/main/service/movie.do?" + q.Encode()
    if err:=fetchSiteJSON(ctx,client,"GET",endpoint,page,nil,&listing);err!=nil {return out,err}
    if q.Get("no")=="" {
        if len(listing.MovieList)==0 {return out,errors.New("해당 회의의 공개 영상이 없어")}
        q.Set("no",listing.MovieList[0].No.String())
    }
    if !digits.MatchString(q.Get("no")) {return out,errors.New("국회 영상 녹화 번호가 없어")}
    q.Set("cmd","fileInfo")
    q.Set("xreferer","")
    var file struct {
        MovieTitle string `json:"movieTitle"`
        FilePath json.RawMessage `json:"filePath"`
    }
    endpoint = strings.TrimRight(root,"/") + "/main/service/movie.do?" + q.Encode()
    if err:=fetchSiteJSON(ctx,client,"GET",endpoint,page,nil,&file);err!=nil {return out,err}
    var path string
    var quality map[string]string
    if json.Unmarshal(file.FilePath,&quality)==nil && quality!=nil {
        for _, k := range []string{"1080p","720p","480p","240p"} {
            if quality[k]!="" {path=quality[k];break}
        }
        if path=="" && quality["default"]!="" {
            path=quality[quality["default"]]
            if path=="" {path=quality["default"]}
        }
    } else { _ = json.Unmarshal(file.FilePath,&path) }
    if path=="" {return out,errors.New("국회 영상 파일 주소가 없어")}
    hls,err:=verifiedMediaURL(path,"assembly")
    if err!=nil {return out,err}
    title:=strings.TrimSpace(listing.ConfTitle+" "+file.MovieTitle)
    if title=="" {title="국회 영상회의록"}
    out = siteVideoMetadata{
        ID: "assembly-"+q.Get("mc")+"-"+q.Get("ct1")+"-"+q.Get("ct2")+"-"+q.Get("ct3")+"-"+q.Get("no"),
        Title:title, WebpageURL:page, Referer:page,
        Formats:[]siteVideoFormat{{FormatID:"hls",URL:hls,Ext:"mp4",Protocol:"m3u8_native",VCodec:"avc1",ACodec:"mp4a.40.2"}},
    }
    return out,nil
}

func baseURLValue(raw json.RawMessage) string {
    var s string
    if json.Unmarshal(raw,&s)==nil{return s}
    var arr []json.RawMessage
    if json.Unmarshal(raw,&arr)==nil {
        for _, item:=range arr {
            if val:=baseURLValue(item);val!="" {return val}
        }
    }
    var obj map[string]json.RawMessage
    if json.Unmarshal(raw,&obj)==nil {
        for _, key:=range []string{"#text","_text","url","value","BaseURL"} {
            if val:=baseURLValue(obj[key]);val!="" {return val}
        }
    }
    return ""
}

func intValue(v json.RawMessage) int {
    var num json.Number
    if json.Unmarshal(v,&num)==nil {n,_:=strconv.Atoi(num.String());return n}
    var s string
    if json.Unmarshal(v,&s)==nil {n,_:=strconv.Atoi(s);return n}
    return 0
}

func resolveNaverVideo(ctx context.Context,client *http.Client,page,root string) (siteVideoMetadata,error) {
    var out siteVideoMetadata
    u,err:=url.Parse(page)
    if err!=nil{return out,errors.New("네이버 클립 주소를 읽지 못했어")}
    mediaID:=u.Query().Get("mediaId")
    if mediaID=="" {mediaID=u.Query().Get("seedMediaId")}
    if !regexp.MustCompile("^[a-fA-F0-9]{16,64}$").MatchString(mediaID) {
        return out,errors.New("네이버 클립 영상 ID가 없어")
    }
    serviceType:=strings.ToUpper(u.Query().Get("serviceType"))
    if serviceType=="" {serviceType="CLIP"}
    if serviceType!="CLIP" && serviceType!="CHZZK" {return out,errors.New("지원하지 않는 네이버 클립 서비스야")}
    payload,_:=json.Marshal(map[string]any{"media":[]map[string]string{{"videoId":mediaID,"serviceType":serviceType}}})
    var response struct {
        Body struct {
            Media []struct {
                MediaID string `json:"mediaId"`
                ServiceType string `json:"serviceType"`
                VOD struct {
                    Playable bool `json:"playable"`
                    Type string `json:"type"`
                    Playback struct {
                        MPD []struct {
                            Period []struct {
                                AdaptationSet []struct {
                                    MimeType string `json:"@mimeType"`
                                    ContentType string `json:"@contentType"`
                                    Representations []struct {
                                        BaseURL json.RawMessage `json:"BaseURL"`
                                        Width json.RawMessage `json:"@width"`
                                        Height json.RawMessage `json:"@height"`
                                        Bandwidth json.RawMessage `json:"@bandwidth"`
                                    } `json:"Representation"`
                                } `json:"AdaptationSet"`
                            } `json:"Period"`
                        } `json:"MPD"`
                    } `json:"playback"`
                } `json:"vod"`
            } `json:"media"`
        } `json:"body"`
    }
    endpoint:=strings.TrimRight(root,"/")+"/api/v3.0/clip/playback"
    if err:=fetchSiteJSON(ctx,client,"POST",endpoint,"https://clip.naver.com/contents",strings.NewReader(string(payload)),&response);err!=nil{return out,err}
    if len(response.Body.Media)==0 || !response.Body.Media[0].VOD.Playable {return out,errors.New("이 네이버 클립은 공개 재생이 허용되지 않아")}
    vod:=response.Body.Media[0].VOD
    var formats []siteVideoFormat
    for _,mpd:=range vod.Playback.MPD {
        for _,period:=range mpd.Period {
            for _,set:=range period.AdaptationSet {
                for _,rep:=range set.Representations {
                    raw:=baseURLValue(rep.BaseURL)
                    if raw=="" {continue}
                    safe,err:=verifiedMediaURL(raw,"naver")
                    if err!=nil {continue}
                    w,h:=intValue(rep.Width),intValue(rep.Height)
                    // The CreatorHub MPD includes an MP4 representation and
                    // a secondary MPEG-TS alternative. Prefer the MP4 file.
                    if set.MimeType=="video/mp2t" { continue }
                    audio:=strings.HasPrefix(set.MimeType,"audio/") || strings.EqualFold(set.ContentType,"audio")
                    if !audio && h==0 && w==0 {continue}
                    id:="v"; ext:="mp4";vcodec:="avc1";acodec:="none"
                    if audio {id="a";ext="m4a";vcodec="none";acodec="mp4a.40.2"}
                    formats=append(formats,siteVideoFormat{
                        FormatID:id+"-"+strconv.Itoa(len(formats)),URL:safe,Ext:ext,
                        VCodec:vcodec,ACodec:acodec,Width:w,Height:h,TBR:intValue(rep.Bandwidth)/1000,
                    })
                }
            }
        }
    }
    if len(formats)==0 {return out,errors.New("네이버 클립 재생 주소를 찾지 못했어")}
    hasSeparateAudio:=false
    for _,f:=range formats {if f.VCodec=="none" {hasSeparateAudio=true}}
    if !hasSeparateAudio {
        // Official CLIP MPD commonly has only a video/mp4 AdaptationSet:
        // its video file may carry audio as well. Do not incorrectly mark it
        // as silent and make yt-dlp's best-combined format selector reject it.
        for i:=range formats {
            if formats[i].VCodec!="none" {formats[i].ACodec="mp4a.40.2"}
        }
    }

    // The original yt-dlp format selector bv*+ba/b handles separate audio and
    // video tracks; do not invent a single combined file if audio is separate.
    sort.SliceStable(formats,func(i,j int)bool{
        if formats[i].VCodec=="none" {return false}
        if formats[j].VCodec=="none" {return true}
        return formats[i].Height>formats[j].Height
    })
    out = siteVideoMetadata{
        ID:"naver-"+mediaID, Title:"네이버 클립 "+mediaID[:8],
        WebpageURL:page, Referer:"https://clip.naver.com/contents",
        Formats:formats,
    }
    return out,nil
}

func resolveSiteVideo(ctx context.Context,page string, client *http.Client) (siteVideoMetadata,error) {
    u,err:=url.Parse(page)
    if err!=nil || !isSpecialVideoPage(page) {
        return siteVideoMetadata{},errors.New("지원하지 않는 사이트 주소야")
    }
    if _,_,err:=cleanURL(page);err!=nil {return siteVideoMetadata{},err}
    switch u.Hostname(){
    case "w3.assembly.go.kr":
        return resolveAssemblyVideo(ctx,client,page,"https://w3.assembly.go.kr")
    case "m.naver.com","clip.naver.com":
        return resolveNaverVideo(ctx,client,page,"https://creatorhub-api.naver.com")
    }
    return siteVideoMetadata{},errors.New("지원하지 않는 사이트 주소야")
}

// Temporary yt-dlp info JSON uses the same download queue, progress, output
// folder, media preparation and cancellation as other providers. Delete it
// after the process exits; signed CDN tokens never enter job history or args.
func (a *App) siteVideoDownloadArgs(ctx context.Context,j *Job,args []string) ([]string,func(),error){
    client:=&http.Client{Timeout:20*time.Second}
    media,err:=resolveSiteVideo(ctx,j.URL,client)
    if err!=nil{return nil,nil,err}
    if len(media.Formats)==0 {return nil,nil,errors.New("재생 가능한 영상 형식을 찾지 못했어")}
    headers:=map[string]string{"User-Agent":siteVideoUA,"Referer":media.Referer}
    for i:=range media.Formats {media.Formats[i].HTTPHeaders=headers}
    info:=map[string]any{
        "id":media.ID,"title":media.Title,"webpage_url":media.WebpageURL,
        "extractor":"ClipCatchSite","extractor_key":"ClipCatchSite",
        "ext":"mp4","_type":"video","formats":media.Formats,
        "http_headers":headers,
    }
    data,err:=json.Marshal(info)
    if err!=nil{return nil,nil,errors.New("영상 정보를 구성하지 못했어")}
    if err:=os.MkdirAll(a.dataDir,0700);err!=nil{return nil,nil,errors.New("영상 임시 폴더를 만들지 못했어")}
    f,err:=os.CreateTemp(a.dataDir,".clipcatch-site-*.info.json")
    if err!=nil{return nil,nil,errors.New("영상 임시 파일을 만들지 못했어")}
    cleanup:=func(){_ = os.Remove(f.Name())}
    if _,err=f.Write(data);err!=nil{_ = f.Close();cleanup();return nil,nil,errors.New("영상 임시 정보를 기록하지 못했어")}
    if err=f.Close();err!=nil{cleanup();return nil,nil,errors.New("영상 임시 정보를 보관하지 못했어")}
    // The original URL is the final argument. Replace only that input with
    // a locally generated info file; all download settings remain unchanged.
    if len(args)==0 || args[len(args)-1]!=j.URL {
        cleanup();return nil,nil,errors.New("다운로드 입력을 확인하지 못했어")
    }
    args=append(append([]string{},args[:len(args)-1]...), "--load-info-json",filepath.Clean(f.Name()))
    return args,cleanup,nil
}
