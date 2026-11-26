package main

// Preview V2 copies, never updates or deletes, already-present local tools
// from the previous independent Preview profile before attempting a network
// download. The source profile and its active jobs remain untouched.
import (
    "io"
    "os"
    "path/filepath"
    "runtime"
)

func copyExistingPreviewTools(targetDir string) {
    if !isPreviewBuild() || runtime.GOOS!="windows" {return}
    root:=os.Getenv("LOCALAPPDATA")
    if root=="" {return}
    sourceDir:=filepath.Join(root,"ClipCatch-Preview","tools")
    if filepath.Clean(sourceDir)==filepath.Clean(targetDir) {return}
    for _,filename:=range []string{"yt-dlp.exe","ffmpeg.exe","ffprobe.exe"} {
        destination:=filepath.Join(targetDir,filename)
        if nonempty(destination) {continue}
        source:=filepath.Join(sourceDir,filename)
        st,e:=os.Lstat(source)
        if e!=nil||!st.Mode().IsRegular()||st.Size()<1024||st.Size()>400<<20{continue}
        input,e:=os.Open(source);if e!=nil{continue}
        magic:=make([]byte,2)
        if _,e=io.ReadFull(input,magic);e!=nil||magic[0]!='M'||magic[1]!='Z' {_=input.Close();continue}
        if _,e=input.Seek(0,io.SeekStart);e!=nil{_ = input.Close();continue}
        tmp,e:=os.CreateTemp(targetDir,".local-tool-*.exe")
        if e!=nil{_ = input.Close();continue}
        written,writeErr:=io.Copy(tmp,input)
        closeErr:=tmp.Close()
        _=input.Close()
        if writeErr!=nil||closeErr!=nil||written!=st.Size(){
            _=os.Remove(tmp.Name());continue
        }
        if e=os.Rename(tmp.Name(),destination);e!=nil{
            _=os.Remove(tmp.Name())
        }
    }
}
