"""Verify the integrated six-field time entry and the real segment API contract
in Chromium, without fetching external videos. Screenshots show the actual UI.
"""
from pathlib import Path
from urllib.parse import urlparse
import json
from playwright.sync_api import expect, sync_playwright

ROOT=Path(__file__).resolve().parents[1]
WEB=ROOT/'src'/'web'
OUT=ROOT/'qa-output';OUT.mkdir(exist_ok=True)
html=(WEB/'index.html').read_text(encoding='utf-8')
extra=['premiere.js','quality.js','workspace-tabs.js','startup-update.js','lifecycle.js',
       'inline-download-ui.js','segment-enhancement.js']
html=html.replace('</body>', ''.join('<script src="/'+x+'"></script>' for x in extra)+'</body>',1)
html=html.replace('__APP_VERSION__','2.0.1-test.11')
posted=[]
state={'version':'2.0.1-test.11','channel':'preview','phase':'ready','jobs':[],
       'save_dir':'C:\\ClipCatch\\Downloads','output_mode':'original',
       'app_update':{'status':'current'},'update_required':False,
       'engine':{'channel':'stable','message':'ready'}}
def serve(route):
    path=urlparse(route.request.url).path
    if path=='/':
        route.fulfill(status=200,content_type='text/html; charset=utf-8',body=html)
    elif path.endswith(('.js','.css')) and (WEB/path.lstrip('/')).is_file():
        route.fulfill(status=200,content_type='text/css' if path.endswith('.css') else 'text/javascript',
                      body=(WEB/path.lstrip('/')).read_text(encoding='utf-8'))
    elif path=='/api/segment/preset':
        route.fulfill(json={'preset':'dotbogi-a-body','installed_font':True,
            'width':1080,'height':1350,'font_size':76})
    elif path=='/api/segment' and route.request.method=='POST':
        data=route.request.post_data_json
        posted.append(data)
        state['jobs'].insert(0,{'id':'preview-test-'+str(len(posted)),'status':'done',
            'file':'C:\\ClipCatch\\Downloads\\test.mp4','percent':100,'detail':'downloaded'})
        route.fulfill(status=202,json={'ok':True,'job_id':'preview-test-'+str(len(posted)),'kind':data['kind']})
    elif path=='/api/state':route.fulfill(json=state)
    elif path=='/api/browser-link/status':route.fulfill(json={'paired':False})
    elif path=='/api/quality/status':route.fulfill(json={'engine_ready':False})
    elif route.request.method=='POST':route.fulfill(json={'ok':True})
    else:route.fulfill(status=404,body='Not found')

with sync_playwright() as p:
    browser=p.chromium.launch(headless=True)
    page=browser.new_page(viewport={'width':1366,'height':768},device_scale_factor=1)
    errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
    page.route('http://127.0.0.1:18768/**',serve)
    page.goto('http://127.0.0.1:18768/')
    expect(page.locator('#ccwWorkflow')).to_be_visible()
    assert page.locator('.workspace-tabs [role=tab]').count()==3,'Must use existing tab'
    page.locator('#ccwPart').click()
    expect(page.locator('#ccwEdit')).to_be_visible()
    expect(page.locator('#ccwStartHour')).to_be_visible()
    expect(page.locator('#ccwStartMinute')).to_be_visible()
    expect(page.locator('#ccwStartSecond')).to_be_visible()
    expect(page.locator('#ccwEndHour')).to_be_visible()
    page.locator('#urlInput').fill('https://w3.assembly.go.kr/main/player.do?menu=10&mc=10&ct1=22&ct2=439&ct3=11&wv=1')
    page.locator('#ccwStartHour').fill('0')
    page.locator('#ccwStartMinute').fill('1')
    page.locator('#ccwStartSecond').fill('20')
    page.locator('#ccwEndHour').fill('0')
    page.locator('#ccwEndMinute').fill('1')
    page.locator('#ccwEndSecond').fill('55')
    expect(page.locator('#ccwDuration')).to_contain_text('00:00:35')
    page.locator('#ccwBodyToggle').click()
    page.locator('#ccwCopy').fill('아래에서 나온 새로운 소식\n두 번째 줄의 본문')
    expect(page.locator('#ccwPreviewText')).to_contain_text('아래에서 나온')
    assert page.locator('#ccwInlineNumber').count()==0,'Yellow number must not be inserted'
    expect(page.locator('#ccwChooseLocal')).to_be_hidden()
    # Verify effective geometry, including the expanded editor selector.
    # A compact override previously reduced the preview to 130px.
    page.wait_for_function("() => document.querySelector('#ccwFrame') && parseFloat(getComputedStyle(document.querySelector('#ccwFrame')).width)>=210",timeout=10000)
    dimensions=page.locator('#ccwFrame').evaluate('(el)=>({width:el.getBoundingClientRect().width,css:getComputedStyle(el).width,parent:el.parentElement.getBoundingClientRect().width})')
    print('In-page preview dimensions:',dimensions,flush=True)
    page.screenshot(path=str(OUT/'preview-v2-six-fields-basic-1366.png'),full_page=False)
    assert dimensions['width']>=210,dimensions
    assert page.locator('#ccwFrame .ccw-preview-gradient').evaluate('(el)=>getComputedStyle(el).backgroundImage')=='none'
    assert page.locator('#ccwVideo').evaluate('(el)=>getComputedStyle(el).objectFit')=='contain'
    # The Bridge body template dynamically pushes two lines higher than one line.
    geometry=page.evaluate("ClipCatchSegmentV2.geometry('첫 줄\\n둘째 줄')")
    assert geometry['bodyY']>=800 and geometry['photoY']<1100,geometry
    assert page.evaluate("getComputedStyle(document.querySelector('#ccwPreviewText')).position")=='absolute'
    page.screenshot(path=str(OUT/'preview-v2-six-fields-basic-1366.png'),full_page=False)
    page.locator('#ccwTrimOnly').click()
    page.wait_for_function("() => document.querySelector('#ccwInlineStatus').textContent.includes('저장 완료')")
    assert len(posted)==1,posted
    assert posted[0]['start_seconds']==80 and posted[0]['end_seconds']==115 and posted[0]['kind']=='clip'
    page.locator('#ccwCreateCarousel').click()
    page.wait_for_function("() => document.querySelector('#ccwInlineStatus').textContent.includes('저장 완료')")
    assert len(posted)==2 and posted[1]['kind']=='carousel' and '본문' in posted[1]['caption'],posted
    page.locator('#ccwDetailOpen').click()
    expect(page.locator('#ccwDetailDialog')).to_be_visible()
    page.screenshot(path=str(OUT/'preview-v2-advanced-1366.png'),full_page=False)
    page.locator('#ccwFontSize').fill('90')
    page.locator('#ccwApplyStyle').click()
    assert page.evaluate('ClipCatchTrimUI.getDraft().advanced') is True
    page.locator('#ccwDetailOpen').click()
    page.locator('#ccwResetStyle').click()
    assert page.evaluate('ClipCatchTrimUI.getDraft().advanced') is False
    assert not errors,errors
    # Existing full download button must still be available in original mode.
    page.locator('#ccwAll').click()
    expect(page.locator('#downloadBtn')).to_be_visible()
    browser.close()
print('PASS: 6 numeric time fields, real segment POST, locked Bridge-style body, detail edit and original download')
