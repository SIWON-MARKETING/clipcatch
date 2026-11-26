package main

// Preview-only section downloads. Uses the existing validated ClipCatch
// yt-dlp job queue, cancellation, output directory, and FFmpeg installation.
// This endpoint is never registered in stable or legacy TEST2 builds.
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "image"
    "image/color"
    "image/png"
    "math"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "runtime"
    "strconv"
    "strings"
    "time"
    "unicode"
    "unicode/utf8"
)

type segmentSpec struct {
    Start int `json:"start_seconds"`
    End int `json:"end_seconds"`
    Kind string `json:"kind"`
    Caption string `json:"caption"`
    Advanced bool `json:"advanced"`
    Size int `json:"size"`
    Color string `json:"color"`
    X float64 `json:"x"`
    Y float64 `json:"y"`
    Fit string `json:"fit"`
}

func (s *segmentSpec) validate() error {
    if s == nil { return errors.New("구간 설정이 없어") }
    if s.Start < 0 || s.End <= s.Start || s.End > 24*3600 || s.End-s.Start > 3600 {
        return errors.New("시작·종료 시간을 확인해 줘. 한 번에 최대 60분까지 지정할 수 있어.")
    }
    if s.Kind != "clip" && s.Kind != "carousel" { return errors.New("다운로드 종류를 확인해 줘") }
    if s.Kind == "carousel" {
        s.Caption = strings.TrimSpace(strings.ReplaceAll(s.Caption,"\r",""))
        if s.Caption == "" || utf8.RuneCountInString(s.Caption) > 220 {
            return errors.New("캐러셀에 넣을 본문은 1~220자로 입력해 줘")
        }
        for _,r := range s.Caption {
            if r == '\n' || r == '\t' { continue }
            if unicode.IsControl(r) { return errors.New("본문에 지원하지 않는 제어 문자가 있어") }
        }
        if !s.Advanced {
            s.Size,s.Color,s.X,s.Y,s.Fit = 76,"#ffffff",50,72,"contain"
        } else {
            if s.Size < 40 || s.Size > 110 || !regexp.MustCompile("^#[0-9a-fA-F]{6}$").MatchString(s.Color) ||
                math.IsNaN(s.X) || math.IsNaN(s.Y) || s.X < 10 || s.X > 90 || s.Y < 15 || s.Y > 90 ||
                (s.Fit != "contain" && s.Fit != "cover") {
                return errors.New("세부 편집 값이 허용 범위를 벗어났어")
            }
        }
    }
    return nil
}

func segmentStamp(s int) string {
    return fmt.Sprintf("%02d-%02d-%02d",s/3600,s/60%60,s%60)
}
func segmentArgs(j *Job, args []string) []string {
    if j == nil || j.Segment == nil || len(args)==0 || args[len(args)-1] != j.URL { return args }
    s:=j.Segment
    start:=strings.ReplaceAll(segmentStamp(s.Start),"-",":")
    end:=strings.ReplaceAll(segmentStamp(s.End),"-",":")
    output:="%(title).100B [%(id)s] ["+segmentStamp(s.Start)+"-"+segmentStamp(s.End)+"-"+j.ID+"].%(ext)s"
    out:=append([]string{},args[:len(args)-1]...)
    for i:=0;i+1<len(out);i++ { if out[i]=="-o" {out[i+1]=output;break} }
    out=append(out,"--download-sections","*"+start+"-"+end,
        // FFmpeg seeks accurately. A later compatibility pass creates
        // a verified H.264/AAC-LC MP4 without changing the original file.
        "--force-keyframes-at-cuts")
    return append(out,j.URL)
}

func (a *App) segmentRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /api/segment/preset",func(w http.ResponseWriter,r *http.Request){
        if !isPreviewBuild(){http.NotFound(w,r);return}
        _,ok:=installedPopFont()
        writeJSON(w,200,map[string]any{
            "preset":"dotbogi-a-body","width":1080,"height":1350,
            "font":"ONE Mobile POP","font_size":76,"min_font_size":40,
            "line_height":1.2,"text_width":1008,"padding_top":18,"padding_bottom":58,
            "number_size":0,"number_gap":0,"transition":0,
            "installed_font":ok,
        })
    })
    mux.HandleFunc("POST /api/segment",a.segmentHandler)
}
func (a *App) segmentHandler(w http.ResponseWriter,r *http.Request) {
    if !isPreviewBuild(){http.NotFound(w,r);return}
    r.Body=http.MaxBytesReader(w,r.Body,16000)
    var in struct {
        URL string `json:"url"`
        segmentSpec
    }
    dec:=json.NewDecoder(r.Body);dec.DisallowUnknownFields()
    if e:=dec.Decode(&in);e!=nil {writeError(w,400,"구간 요청을 읽지 못했어");return}
    if e:=in.segmentSpec.validate();e!=nil {writeError(w,400,e.Error());return}
    clean,host,e:=cleanURL(in.URL)
    if e!=nil {writeError(w,400,e.Error());return}
    if isRedditHost(host) {writeError(w,400,"Reddit 영상은 브라우저 확장프로그램으로 다운로드해 줘");return}
    // Copy job-specific data so later UI edits cannot mutate a running render.
    seg:=in.segmentSpec
    if seg.Kind=="carousel" {
        if _,ok:=installedPopFont();!ok {
            writeError(w,409,"ONE Mobile POP 글꼴을 찾지 못했어. Windows에 설치한 다음 Preview를 다시 실행해 줘.")
            return
        }
    }
    a.mu.Lock()
    if a.newWorkBlockedLocked(){a.mu.Unlock();writeError(w,409,"현재 다른 작업을 처리 중이야. 준비가 끝난 뒤 다시 시도해 줘.");return}
    a.trimHistoryLocked()
    a.created++
    id:=fmt.Sprintf("%d-%d",time.Now().UnixNano(),a.created)
    j:=&Job{ID:id,URL:clean,Host:host,Task:"segment",OutputMode:originalMode,
      Segment:&seg,Title:"구간 다운로드",Status:"queued",Percent:0,
      Time:time.Now().Format("2006-01-02 15:04"),Detail:"선택 구간 다운로드 대기 중"}
    a.state.Jobs=append([]*Job{j},a.state.Jobs...)
    a.mu.Unlock()
    a.queue<-id
    writeJSON(w,202,map[string]any{"ok":true,"job_id":id,"kind":seg.Kind})
}

// Pop fonts are user-installed. Never copy the font file into app resources,
// GitHub, or distributed ZIPs. Render uses a temporary local copy and deletes it.
func installedPopFont()(string,bool) {
    if override:=os.Getenv("CLIPCATCH_PREVIEW_FONT");override!=""{
        if validLocalFont(override){return override,true}
    }
    roots:=[]string{
        filepath.Join(os.Getenv("WINDIR"),"Fonts"),
        filepath.Join(os.Getenv("LOCALAPPDATA"),"Microsoft","Windows","Fonts"),
    }
    for _,root:=range roots{
        entries,e:=os.ReadDir(root);if e!=nil{continue}
        for _,entry:=range entries{
            if entry.IsDir(){continue}
            name:=strings.ToLower(entry.Name())
            pop:=strings.Contains(name,"pop")
            mobile:=strings.Contains(name,"mobile")||strings.Contains(name,"모바일")||strings.Contains(name,"one")
            if pop&&mobile{
                full:=filepath.Join(root,entry.Name())
                if validLocalFont(full){return full,true}
            }
        }
    }
    return "",false
}
func validLocalFont(path string) bool {
    ext:=strings.ToLower(filepath.Ext(path))
    if ext!=".ttf" && ext!=".otf" {return false}
    f,e:=os.Open(path);if e!=nil{return false}
    defer f.Close()
    st,e:=f.Stat();if e!=nil||!st.Mode().IsRegular()||st.Size()<1000||st.Size()>24<<20{return false}
    b:=make([]byte,4);_,e=f.Read(b);if e!=nil{return false}
    tag:=string(b)
    return tag=="OTTO"||tag=="true"||tag=="typ1"||(b[0]==0&&b[1]==1&&b[2]==0&&b[3]==0)
}

func segmentWrapWidth(line string,size int) float64 {
    // Conservative glyph estimate for preview and FFmpeg: Hangul occupies
    // roughly 1 em; ASCII roughly half. Shrink a long whole line, don't wrap
    // a user-authored line silently. Fits 1008px as the dotbogi-a template.
    used:=0.0
    for _,r:=range line{
        switch {
        case r=='\t':used+=1
        case r==' ':used+=0.34
        case r<=127:used+=0.55
        default:used+=1
        }
    }
    return used*float64(size)
}
func segmentBodyLayout(s *segmentSpec) (fontSize,numberTop,mainTop,photoEnd int,err error){
    if s==nil || s.Caption=="" {return 0,0,0,0,errors.New("caption missing")}
    fontSize=s.Size
    lines:=strings.Split(s.Caption,"\n")
    if len(lines)>8 {return 0,0,0,0,errors.New("본문 줄 수는 최대 8줄까지 지원해")}
    if !s.Advanced {
        // Equivalent to shrink-line in dotbogi-a, with fixed lower bound.
        for fontSize>40 {
            wide:=false
            for _,line:=range lines {if segmentWrapWidth(line,fontSize)>1008 {wide=true}}
            if !wide{break};fontSize-=2
        }
    }
    for _,line:=range lines {if segmentWrapWidth(line,fontSize)>1008{
        return 0,0,0,0,errors.New("본문이 너무 길어. 줄을 나누거나 문구를 줄여 줘")
    }}
    mainH:=int(math.Ceil(float64(len(lines))*float64(fontSize)*1.2))
    total:=18+mainH+58
    if total > 1350-90 {return 0,0,0,0,errors.New("본문이 카드에 들어가지 않아. 줄 수를 줄여 줘")}
    photoEnd=1350-total
    numberTop=0 // numbering removed in Preview test.10
    mainTop=photoEnd+18
    return
}

func makeSectionShade(path string,photoEnd int) error {
    if photoEnd<=90||photoEnd>=1350{return errors.New("invalid photo region")}
    // Fixed solid caption block with no translucent transition / gradient.
    img:=image.NewNRGBA(image.Rect(0,0,1080,1350))
    for y:=photoEnd;y<1350;y++{
        for x:=0;x<1080;x++{img.SetNRGBA(x,y,color.NRGBA{R:0x0e,G:0x0e,B:0x10,A:255})}
    }
    f,e:=os.Create(path);if e!=nil{return e}
    defer f.Close()
    return png.Encode(f,img)
}

func probeSegmentVideo(parent context.Context, ffmpeg, path string)(duration float64,hasAudio bool,e error){
    if ffmpeg=="" {return 0,false,errors.New("FFmpeg not installed")}
    probe:=filepath.Join(filepath.Dir(ffmpeg),"ffprobe")
    if runtime.GOOS=="windows"{probe+=".exe"}
    ctx,cancel:=context.WithTimeout(parent,30*time.Second);defer cancel()
    out,e:=hiddenCommandContext(ctx,probe,"-v","error","-show_entries",
        "format=duration:stream=codec_type,width,height,codec_name",
        "-of","json",path).Output()
    if e!=nil||len(out)>65536 {return 0,false,errors.New("FFprobe did not recognize the video")}
    var meta struct{
        Format struct {Duration string `json:"duration"`} `json:"format"`
        Streams []struct {
            Type string `json:"codec_type"`
            Width int `json:"width"`
            Height int `json:"height"`
            Codec string `json:"codec_name"`
        } `json:"streams"`
    }
    if json.Unmarshal(out,&meta)!=nil{return 0,false,errors.New("FFprobe response unreadable")}
    duration,e=strconv.ParseFloat(meta.Format.Duration,64)
    if e!=nil||!isFinite(duration)||duration<=0{return 0,false,errors.New("video duration missing")}
    video:=false
    for _,st:=range meta.Streams{if st.Type=="video"{video=true};if st.Type=="audio"{hasAudio=true}}
    if !video{return 0,false,errors.New("video track missing")}
    return duration,hasAudio,nil
}
func isFinite(n float64)bool{return !math.IsNaN(n)&&!math.IsInf(n,0)}

// Always publish a compatibility-encoded section. yt-dlp's extracted audio
// stream can be a codec variant that Windows/Premiere refuses to play.
func (a *App) finishSegment(id string,ctx context.Context){
    j:=a.getJob(id)
    if j==nil||j.Segment==nil{return}
    s:=j.Segment
    source:=j.File
    if !a.isSegmentWorkspace(j) || source=="" || !nonempty(source) || !validPostSource(j.DownloadDir,source) {
        a.segmentFailure(id,"CC-421","다운로드된 구간 파일을 찾지 못했어");return
    }
    a.mu.Lock();ff:=a.ffmpeg;a.mu.Unlock()
    duration,audio,e:=probeSegmentVideo(ctx,ff,source)
    if e!=nil {a.segmentFailure(id,"CC-422","구간 영상 형식을 확인하지 못했어.");return}
    if duration>float64(s.End-s.Start)+4 {
        a.segmentFailure(id,"CC-423","선택 구간보다 긴 영상이 저장됐어. 전체 영상으로 잘못 저장되지 않도록 중단했어.");return
    }
    if s.Kind=="clip"&&!audio {
        a.segmentFailure(id,"CC-427","구간 영상에 오디오 트랙이 없어. 원본 영상의 소리를 확인해 줘.");return
    }
    var rendered string
    if s.Kind=="clip" {
        a.mutateJob(id,func(st *Job){st.Stage="rendering";st.Detail="호환되는 H.264/AAC 구간 영상 제작 중"})
        rendered,e=a.renderSegmentClip(ctx,id,ff,source,s,duration,audio)
    } else if s.Kind=="carousel" {
        a.mutateJob(id,func(st *Job){st.Stage="rendering";st.Detail="1080×1350 영상 캐러셀 제작 중"})
        rendered,e=a.renderSegmentCarousel(ctx,id,ff,source,s,duration,audio)
    } else {
        a.segmentFailure(id,"CC-424","다운로드 형식 오류");return
    }
    if ctx.Err()!=nil||a.isJobCancelled(id){return}
    if e!=nil{
        a.segmentFailure(id,"CC-425","MP4 제작에 실패했어. 작업 파일은 임시 폴더에 보관했어.")
        return
    }
    // The user receives only the completed output, never the temporary source.
    a.mutateJob(id,func(st *Job){
        st.OriginalFile="";st.File=rendered;st.Status="done";st.Percent=100;st.Stage="";
        if s.Kind=="clip"{st.Detail="선택 구간 MP4 완료 · H.264/AAC"}
        if s.Kind=="carousel"{st.Detail="1080×1350 캐러셀 MP4 완료"}
    })
    if err:=a.cleanupSegmentWorkspace(j);err!=nil{
        a.mutateJob(id,func(st *Job){st.Detail+=" · 임시 파일 정리는 다음 실행에서 확인 필요"})
    }
}

func (a *App) segmentFailure(id,code,title string){
    a.mutateJob(id,func(st *Job){
        if st.Status=="cancelled"{return}
        st.Status="error";st.Stage="";st.ErrorCode=code;st.ErrorTitle=title;
        st.ErrorAdvice="영상 도구와 원본 링크 상태를 확인해 줘. 실패한 중간 파일은 앱 내부 임시 폴더에 있어."
        st.Detail=title
    })
}

func copyFileSimple(src,dst string)error {
    b,e:=os.ReadFile(src);if e!=nil{return e}
    return os.WriteFile(dst,b,0600)
}
// Validate that the exported stream has real AAC-LC metadata, not an
// unsupported source-specific mp4a codec that was simply remuxed into MP4.
func segmentAudioCodec(parent context.Context,ff,path string)(string,error){
    probe:=filepath.Join(filepath.Dir(ff),"ffprobe")
    if runtime.GOOS=="windows"{probe+=".exe"}
    ctx,cancel:=context.WithTimeout(parent,20*time.Second);defer cancel()
    data,err:=hiddenCommandContext(ctx,probe,"-v","error","-select_streams","a:0",
        "-show_entries","stream=codec_name","-of","default=noprint_wrappers=1:nokey=1",path).Output()
    if err!=nil{return "",err}
    return strings.TrimSpace(string(data)),nil
}

func verifySegmentMP4(parent context.Context,ff,path string,wantDuration float64,requireAudio bool) error {
    duration,hasAudio,err:=probeSegmentVideo(parent,ff,path)
    if err!=nil || math.Abs(duration-wantDuration)>2.5{return errors.New("rendered MP4 duration/video check failed")}
    if requireAudio && !hasAudio{return errors.New("rendered MP4 audio track missing")}
    if hasAudio{
        codec,err:=segmentAudioCodec(parent,ff,path)
        if err!=nil||codec!="aac"{return errors.New("rendered MP4 must contain AAC audio")}
    }
    return nil
}

func (a *App) renderSegmentClip(parent context.Context,id,ff,source string,s *segmentSpec,duration float64,hasAudio bool)(string,error) {
    work,err:=os.MkdirTemp(a.dataDir,"preview-trim-")
    if err!=nil{return "",err}
    defer os.RemoveAll(work)
    temp:=filepath.Join(work,"section.mp4")
    args:=[]string{"-hide_banner","-loglevel","error","-y","-i",source,
        "-map","0:v:0","-map","0:a:0?","-t",fmt.Sprintf("%.3f",duration),
        "-c:v","libx264","-preset","veryfast","-crf","18","-pix_fmt","yuv420p","-r","30",
        "-c:a","aac","-profile:a","aac_low","-b:a","128k","-ar","48000","-ac","2",
        "-movflags","+faststart","-shortest",temp}
    ctx,cancel:=context.WithTimeout(parent,2*time.Hour);defer cancel()
    cmd:=hiddenCommandContext(ctx,ff,args...)
    a.setJobProcess(id,cmd)
    output,err:=cmd.CombinedOutput()
    a.clearJobProcess(id,cmd)
    if err!=nil{return "",fmt.Errorf("MP4 audio/video conversion failed (%d diagnostic bytes): %w",len(output),err)}
    if err=verifySegmentMP4(parent,ff,temp,duration,hasAudio);err!=nil{return "",err}
    if err=os.MkdirAll(a.saveDir,0755);err!=nil{return "",err}
    target:=filepath.Join(a.saveDir,"ClipCatch_"+segmentStamp(s.Start)+"_"+segmentStamp(s.End)+"_"+id+"_section.mp4")
    if err=os.Rename(temp,target);err!=nil{return "",err}
    return target,nil
}

func (a *App) renderSegmentCarousel(parent context.Context,id,ff,source string,s *segmentSpec,duration float64,hasAudio bool)(string,error){
    if s==nil{return "",errors.New("no segment options")}
    font,ok:=installedPopFont();if !ok{return "",errors.New("POP font not installed")}
    size,_,textY,photoY,err:=segmentBodyLayout(s)
    if err!=nil{return "",err}
    work,err:=os.MkdirTemp(a.dataDir,"preview-segment-")
    if err!=nil{return "",err}
    defer os.RemoveAll(work)
    if err=copyFileSimple(font,filepath.Join(work,"localfont"+filepath.Ext(font)));err!=nil{return "",err}
    fontName:="localfont"+filepath.Ext(font)
    if err=os.WriteFile(filepath.Join(work,"caption.txt"),[]byte(s.Caption),0600);err!=nil{return "",err}
    // Always show the complete frame, centered with letterboxes if needed.
    // The lower caption panel is solid black: no crop and no gradient.
    photoH:=strconv.Itoa(photoY)
    geometry:="scale=1080:"+photoH+":force_original_aspect_ratio=decrease,"+
        "pad=1080:"+photoH+":(ow-iw)/2:(oh-ih)/2:color=0x0e0e10,"+
        "pad=1080:1350:0:0:color=0x0e0e10"
    base:="[0:v]"+geometry+",setsar=1,fps=30,format=yuv420p"
    textX:="(w-text_w)/2"
    if s.Advanced{textX=fmt.Sprintf("(w*%.5f)-(text_w/2)",s.X/100)}
    if s.Advanced{textY=int(math.Round(s.Y/100*1350))}
    lineSpacing:=int(math.Round(float64(size)*0.2))
    colorHex:=strings.TrimPrefix(s.Color,"#")
    if !regexp.MustCompile("^[0-9a-fA-F]{6}$").MatchString(colorHex) {return "",errors.New("bad caption color")}
    base+=",drawtext=fontfile="+fontName+":textfile=caption.txt:expansion=none"+
        ":fontcolor=0x"+colorHex+":fontsize="+strconv.Itoa(size)+
        ":line_spacing="+strconv.Itoa(lineSpacing)+":x="+textX+":y="+strconv.Itoa(textY)+"[outv]"
    temp:=filepath.Join(work,"carousel.mp4")
    args:=[]string{"-hide_banner","-loglevel","error","-y","-i",source}
    mapAudio:="0:a:0?"
    if !hasAudio{
        args=append(args,"-f","lavfi","-i","anullsrc=channel_layout=stereo:sample_rate=48000")
        mapAudio="1:a:0"
    }
    args=append(args,"-filter_complex",base,"-map","[outv]","-map",mapAudio,
        "-t",fmt.Sprintf("%.3f",duration),"-c:v","libx264","-preset","veryfast",
        "-crf","21","-pix_fmt","yuv420p","-r","30",
        "-c:a","aac","-profile:a","aac_low","-b:a","128k","-ar","48000","-ac","2",
        "-movflags","+faststart","-shortest",temp)
    ctx,cancel:=context.WithTimeout(parent,2*time.Hour);defer cancel()
    cmd:=hiddenCommandContext(ctx,ff,args...);cmd.Dir=work
    a.setJobProcess(id,cmd)
    output,err:=cmd.CombinedOutput()
    a.clearJobProcess(id,cmd)
    if err!=nil{return "",fmt.Errorf("render failed (%d diagnostic bytes): %w",len(output),err)}
    if err=verifySegmentMP4(parent,ff,temp,duration,true);err!=nil{return "",err}
    target:=filepath.Join(a.saveDir,"ClipCatch_"+segmentStamp(s.Start)+"_"+segmentStamp(s.End)+"_"+id+"_1080x1350.mp4")
    if err=os.MkdirAll(a.saveDir,0755);err!=nil{return "",err}
    if err=os.Rename(temp,target);err!=nil{return "",err}
    return target,nil
}
