"""Real Chromium UI-only verification for the embedded TEST2 trim workflow.

No remote video, login, GitHub release, download execution, or real export is used.
"""
from pathlib import Path
from urllib.parse import urlparse
import os

from playwright.sync_api import expect, sync_playwright

ROOT = Path(__file__).resolve().parents[1]
WEB = ROOT / "src" / "web"
OUT = ROOT / "qa-output"
OUT.mkdir(exist_ok=True)

scripts = ["premiere.js", "quality.js", "workspace-tabs.js",
           "startup-update.js", "lifecycle.js", "inline-download-ui.js"]
html = (WEB / "index.html").read_text(encoding="utf-8")
html = html.replace("</body>", "".join('<script src="/'+n+'"></script>' for n in scripts) + "</body>", 1)
html = html.replace("__APP_VERSION__", "2.0.1-test.7")
state = {
    "version": "2.0.1-test.7",
    "channel": "test",
    "phase": "ready",
    "jobs": [],
    "save_dir": "C:\\Users\\Designer\\Downloads\\ClipCatch-Test2",
    "output_mode": "original",
    "app_update": {"status": "current"},
    "update_required": False,
    "update_checking": False,
    "engine": {"channel": "stable", "message": "사용 가능"},
    "diagnostics": {"checks": []},
}
download_posts = []
def serve(route):
    path = urlparse(route.request.url).path
    if path == "/":
        route.fulfill(status=200, content_type="text/html; charset=utf-8", body=html)
    elif path.endswith((".js", ".css")) and (WEB / path.lstrip("/")).is_file():
        ctype = "text/css" if path.endswith(".css") else "text/javascript"
        route.fulfill(status=200, content_type=ctype,
                      body=(WEB / path.lstrip("/")).read_text(encoding="utf-8"))
    elif path == "/api/state":
        route.fulfill(json=state)
    elif path == "/api/browser-link/status":
        route.fulfill(json={"paired": False})
    elif path == "/api/quality/status":
        route.fulfill(json={"engine_ready": False, "engine_bundled": False})
    elif route.request.method == "POST":
        if path == "/api/download":
            download_posts.append(route.request.post_data_json)
        route.fulfill(json={"ok": True, "count": 1})
    else:
        route.fulfill(status=404, body="Not found")

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 1366, "height": 768}, device_scale_factor=1)
    errors=[]
    page.on("pageerror",lambda e:errors.append(str(e)))
    page.route("http://127.0.0.1:18766/**",serve)
    page.goto("http://127.0.0.1:18766/")
    expect(page.locator("#navDownload")).to_be_visible()
    expect(page.locator("#ccwWorkflow")).to_be_visible()
    assert page.locator(".workspace-tabs [role=tab]").count()==3, "UI must not add a fourth tab"
    expect(page.locator("#downloadBtn")).to_be_visible()
    expect(page.locator("#ccwEdit")).to_be_hidden()
    page.screenshot(path=str(OUT/"inline_ui_full_1366.png"),full_page=False)

    page.locator("#urlInput").fill("https://w3.assembly.go.kr/main/player.do?menu=10&mc=10&ct1=22&ct2=439&ct3=11&wv=1")
    page.locator("#ccwPart").click()
    expect(page.locator("#ccwEdit")).to_be_visible()
    expect(page.locator("#downloadBtn")).to_be_hidden()
    expect(page.locator("#ccwTrimOnly")).to_be_visible()
    page.locator("#ccwStart").fill("00:12:30")
    page.locator("#ccwEnd").fill("00:13:00")
    expect(page.locator("#ccwDuration")).to_contain_text("00:00:30")
    page.locator("#ccwBodyToggle").click()
    page.locator("#ccwCopy").fill("국회에서 포착된\n이 장면")
    expect(page.locator("#ccwPreviewText")).to_contain_text("국회에서 포착된")
    page.locator("#ccwTrimOnly").click()
    expect(page.locator("#ccwInlineStatus")).to_contain_text("아직 실행되지 않아")
    assert not download_posts, "UI-only trim action must not call full download endpoint"
    page.screenshot(path=str(OUT/"inline_ui_trim_1366.png"),full_page=False)

    page.locator("#ccwDetailOpen").click()
    expect(page.locator("#ccwDetailDialog")).to_be_visible()
    page.locator("#ccwFontSize").fill("95")
    page.locator("#ccwDetailClose").click()
    assert page.evaluate("ClipCatchTrimUI.getDraft().size") == 76
    page.locator("#ccwDetailOpen").click()
    page.locator("#ccwFontSize").fill("92")
    page.locator("#ccwApplyStyle").click()
    assert page.evaluate("ClipCatchTrimUI.getDraft().size") == 92
    assert page.evaluate("ClipCatchTrimUI.getDraft().advanced") is True
    page.locator("#ccwDetailOpen").click()
    page.screenshot(path=str(OUT/"inline_ui_detail_1366.png"),full_page=False)
    page.locator("#ccwResetStyle").click()
    assert page.evaluate("ClipCatchTrimUI.getDraft().advanced") is False
    expect(page.locator("#ccwCopy")).to_have_value("국회에서 포착된\n이 장면")

    page.set_viewport_size({"width": 1536, "height": 900})
    page.screenshot(path=str(OUT/"inline_ui_trim_1536.png"),full_page=False)
    # Existing full download path must still be callable from the original tab.
    page.locator("#ccwAll").click()
    expect(page.locator("#downloadBtn")).to_be_visible()
    page.locator("#downloadBtn").click()
    assert len(download_posts)==1, "full download must retain original API path"
    assert not errors, "Chromium JS errors: "+repr(errors)
    browser.close()
print("PASS: 3 existing tabs; full download preserved; times, caption, modal Apply/Cancel, and UI-only trim actions verified")
