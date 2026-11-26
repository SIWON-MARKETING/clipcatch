package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestSegmentWorkspaceStoresOnlyTemporarySources(t *testing.T) {
    root:=t.TempDir()
    a:=newApp(filepath.Join(root,"data"),filepath.Join(root,"downloads"))
    if err:=os.MkdirAll(a.saveDir,0700);err!=nil{t.Fatal(err)}
    existing:=filepath.Join(a.saveDir,"user-original.mp4")
    if err:=os.WriteFile(existing,[]byte("PRESERVE"),0600);err!=nil{t.Fatal(err)}
    j:=&Job{ID:"segment-123",Task:"segment",Segment:&segmentSpec{Kind:"clip",Start:1,End:3}}
    a.state.Jobs=[]*Job{j}
    if err:=a.prepareSegmentWorkspace(j);err!=nil{t.Fatal(err)}
    if !a.isSegmentWorkspace(j){t.Fatal("new workspace is not job-owned")}
    if a.downloadRoot(j)!=j.DownloadDir{t.Fatal("segment download not isolated")}
    if j.DownloadDir==a.saveDir{t.Fatal("segment sources in user output folder")}
    input:=filepath.Join(j.DownloadDir,"temp-source.mp4")
    if err:=os.WriteFile(input,[]byte("temporary"),0600);err!=nil{t.Fatal(err)}
    forged:=&Job{ID:j.ID,Task:"segment",Segment:j.Segment,DownloadDir:a.saveDir}
    if a.isSegmentWorkspace(forged){t.Fatal("user output folder should never count as temp")}
    if err:=a.cleanupSegmentWorkspace(forged);err==nil{t.Fatal("must reject deletion of unowned directory")}
    if err:=a.cleanupSegmentWorkspace(j);err!=nil{t.Fatal(err)}
    if _,err:=os.Stat(j.DownloadDir);!os.IsNotExist(err){t.Fatal("temporary source directory not removed")}
    data,err:=os.ReadFile(existing)
    if err!=nil||string(data)!="PRESERVE"{t.Fatal("pre-existing user original was modified")}
    if a.state.Jobs[0].DownloadDir!=""{t.Fatal("download state still exposes temp folder")}
}
