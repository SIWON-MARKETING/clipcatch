package main

import (
    "os"
    "path/filepath"
)

// The preview build is a portable copy: separate identity, port and data.
// It never migrates or shuts down stable or the older TEST2 installation.
var appChannel = "stable"

func isPreviewBuild() bool { return appChannel == "preview" }
func isTrialBuild() bool { return appChannel == "test" || isPreviewBuild() }

func runtimeAppID() string {
    switch appChannel {
    case "preview":
        return "clipcatch-preview-v2"
    case "test":
        return "clipcatch-test2"
    default:
        return "clipcatch"
    }
}
func runtimeDataName() string {
    switch appChannel {
    case "preview":
        return "ClipCatch-Preview-V2"
    case "test":
        return "ClipCatch-Test2"
    default:
        return appName
    }
}
func runtimePort() string {
    switch appChannel {
    case "preview":
        return "18768"
    case "test":
        return "18766"
    default:
        return "8765"
    }
}
func runtimeSaveDir(home string) string {
    name, key := "ClipCatch", "CLIPCATCH_SAVEDIR"
    switch appChannel {
    case "preview":
        name, key = "ClipCatch-Preview-V2", "CLIPCATCH_PREVIEW_SAVEDIR"
    case "test":
        name, key = "ClipCatch-Test2", "CLIPCATCH_TEST2_SAVEDIR"
    }
    if override := os.Getenv(key); override != "" {
        return override
    }
    return filepath.Join(home, "Downloads", name)
}
func selectedOutputMode(requested, fallback string) (string, bool) {
    if requested == "" {
        requested = fallback
    }
    if requested != originalMode && requested != premiereMode {
        return "", false
    }
    return requested, true
}
