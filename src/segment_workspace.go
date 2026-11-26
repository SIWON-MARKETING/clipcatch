package main

import (
    "errors"
    "os"
    "path/filepath"
)

// Segment-only source files are downloaded under the app data directory.
// They are not user output and may be deleted only with a matching job receipt.
func (a *App) isSegmentWorkspace(j *Job) bool {
    if j == nil || j.Task != "segment" || j.Segment == nil || j.DownloadDir == "" {
        return false
    }
    root, err := filepath.Abs(filepath.Join(a.dataDir, "segment-work"))
    if err != nil { return false }
    dir, err := filepath.Abs(j.DownloadDir)
    if err != nil || filepath.Dir(dir) != root { return false }
    st, err := os.Lstat(dir)
    if err != nil || !st.IsDir() || st.Mode() & os.ModeSymlink != 0 { return false }
    actualRoot, err := filepath.EvalSymlinks(root)
    if err != nil { return false }
    actualDir, err := filepath.EvalSymlinks(dir)
    if err != nil || filepath.Dir(actualDir) != actualRoot { return false }
    receipt, err := os.ReadFile(filepath.Join(dir, ".clipcatch-segment-job"))
    return err == nil && string(receipt) == j.ID
}

func (a *App) prepareSegmentWorkspace(j *Job) error {
    if j == nil || j.Segment == nil || j.Task != "segment" {
        return nil
    }
    if j.DownloadDir != "" {
        if a.isSegmentWorkspace(j) { return nil }
        return errors.New("invalid segment temporary directory")
    }
    root := filepath.Join(a.dataDir, "segment-work")
    if err := os.MkdirAll(root, 0700); err != nil { return err }
    dir, err := os.MkdirTemp(root, "job-")
    if err != nil { return err }
    if err := os.WriteFile(filepath.Join(dir, ".clipcatch-segment-job"), []byte(j.ID), 0600); err != nil {
        _ = os.RemoveAll(dir)
        return err
    }
    j.DownloadDir = dir
    a.mutateJob(j.ID, func(st *Job) { st.DownloadDir = dir })
    return nil
}

func (a *App) cleanupSegmentWorkspace(j *Job) error {
    if !a.isSegmentWorkspace(j) { return errors.New("segment directory ownership not verified") }
    if err := os.RemoveAll(j.DownloadDir); err != nil { return err }
    a.mutateJob(j.ID, func(st *Job) {
        st.DownloadDir = ""
        st.OriginalFile = ""
        st.Media = nil
    })
    return nil
}
