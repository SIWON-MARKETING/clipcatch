(() => {
'use strict';
const $ = id => document.getElementById(id);
const entry = document.querySelector('.main-card .entry');
const info = entry?.querySelector('.info-line');
const originalAction = entry?.querySelector('.action-row');
const originalButton = $('downloadBtn');
if (!entry || !info || !originalAction || !originalButton || $('ccwWorkflow')) return;
const cssLink = document.createElement('link');
cssLink.rel='stylesheet';cssLink.href='/inline-download-ui.css';
document.head.append(cssLink);
info.insertAdjacentHTML('afterend',"\n<section class=\"ccw-workflow\" id=\"ccwWorkflow\" aria-label=\"영상 구간 및 본문 제작\">\n  <div class=\"ccw-mode\">\n    <span class=\"ccw-mode-label\">다운로드 범위</span>\n    <div class=\"ccw-segment\" role=\"group\" aria-label=\"다운로드 범위 선택\">\n      <button id=\"ccwAll\" type=\"button\" aria-pressed=\"true\">전체 영상</button>\n      <button id=\"ccwPart\" type=\"button\" aria-pressed=\"false\">구간 지정</button>\n    </div>\n    <span class=\"ccw-summary\" id=\"ccwModeHint\">전체 영상은 기존 방식대로 다운로드해.</span>\n  </div>\n  <div id=\"ccwEdit\" class=\"ccw-edit\" hidden>\n    <div class=\"ccw-quickgrid\">\n      <div class=\"ccw-controls\">\n        <div class=\"ccw-rowtitle\">사용할 영상 구간 <small>시 : 분 : 초</small></div>\n        <div class=\"ccw-times\">\n          <label for=\"ccwStart\"><span>시작</span><input id=\"ccwStart\" type=\"text\" inputmode=\"numeric\" value=\"00:00:00\" maxlength=\"9\" spellcheck=\"false\" aria-label=\"구간 시작 시간 시 분 초\"></label>\n          <span class=\"ccw-tilde\" aria-hidden=\"true\">~</span>\n          <label for=\"ccwEnd\"><span>종료</span><input id=\"ccwEnd\" type=\"text\" inputmode=\"numeric\" value=\"00:00:30\" maxlength=\"9\" spellcheck=\"false\" aria-label=\"구간 종료 시간 시 분 초\"></label>\n        </div>\n        <div class=\"ccw-duration\" id=\"ccwDurationRow\"><span id=\"ccwTimeHint\">원본 길이는 실제 다운로드 기능 연결 후 확인돼.</span><strong id=\"ccwDuration\">선택 30초</strong></div>\n        <div class=\"ccw-config-row\">\n          <button id=\"ccwBodyToggle\" type=\"button\" class=\"ccw-subbtn ccw-green\" aria-expanded=\"false\">+ 본문 넣기</button>\n          <button id=\"ccwDetailOpen\" type=\"button\" class=\"ccw-subbtn\">세부 편집 <span aria-hidden=\"true\">↗</span></button>\n          <span class=\"ccw-small\">본문 기본형 · POP 76px</span>\n        </div>\n        <div class=\"ccw-copy-section\" id=\"ccwCopySection\" hidden>\n          <label for=\"ccwCopy\">본문 문구 <span id=\"ccwCopyStyle\">기본 크기·위치 자동 적용</span></label>\n          <textarea id=\"ccwCopy\" maxlength=\"220\" placeholder=\"영상에 들어갈 본문만 입력해줘.\" rows=\"3\"></textarea>\n        </div>\n      </div>\n      <div class=\"ccw-preview-area\">\n        <div class=\"ccw-preview-header\"><span>캐러셀 영상 미리보기</span><span>4:5</span></div>\n        <div class=\"ccw-preview-frame\" id=\"ccwFrame\">\n          <div id=\"ccwPreviewEmpty\" class=\"ccw-preview-placeholder\">링크 영상 미리보기<br>엔진 연결 예정</div>\n          <video id=\"ccwVideo\" muted playsinline preload=\"metadata\" hidden></video>\n          <div class=\"ccw-preview-gradient\"></div>\n          <div id=\"ccwPreviewText\" class=\"ccw-preview-text\" hidden></div>\n        </div>\n        <button type=\"button\" class=\"ccw-subbtn\" id=\"ccwChooseLocal\" style=\"font-size:10px;padding:6px 10px\">로컬 영상으로 미리보기</button>\n        <input id=\"ccwLocalFile\" type=\"file\" accept=\"video/*,.mp4,.mov,.webm,.mkv\" hidden>\n        <span class=\"ccw-preview-note\">미리보기 참고 화면 · 출력 미연결</span>\n      </div>\n    </div>\n    <p class=\"ccw-edit-notice\">UI 우선 구현 · 구간 다운로드·MP4 출력은 아직 연결되지 않았어. 기존 전체 영상 다운로드는 그대로 사용할 수 있어.</p>\n    <p id=\"ccwInlineStatus\" class=\"ccw-status\" role=\"status\" aria-live=\"polite\" hidden></p>\n  </div>\n</section>\n");
originalAction.insertAdjacentHTML('afterend',"\n<div class=\"ccw-trim-actions\" id=\"ccwTrimActions\" hidden>\n  <button id=\"ccwTrimOnly\" class=\"ccw-download\" type=\"button\">이 구간만 다운로드</button>\n  <button id=\"ccwCreateCarousel\" class=\"ccw-carousel\" type=\"button\">영상 캐러셀 MP4 만들기</button>\n  <small>UI 시제품 · 파일 저장 기능 미연결</small>\n</div>");
document.body.insertAdjacentHTML('beforeend',"\n<dialog id=\"ccwDetailDialog\" aria-labelledby=\"ccwDetailTitle\">\n  <div class=\"ccw-detail-head\">\n    <h2 id=\"ccwDetailTitle\">세부 편집 <span style=\"font-size:12px;font-weight:500;color:#b0bfcd\">/ 선택한 구간 · 영상 캐러셀</span></h2>\n    <button id=\"ccwDetailClose\" type=\"button\" aria-label=\"세부 편집 닫기\">×</button>\n  </div>\n  <div class=\"ccw-detail-grid\">\n    <div class=\"ccw-detail-stage\">\n      <div class=\"ccw-detail-preview\" id=\"ccwDetailPreview\">\n        <div class=\"ccw-preview-placeholder\" id=\"ccwDetailEmpty\">선택한 로컬 영상만<br>임시로 미리볼 수 있어</div>\n        <video id=\"ccwDetailVideo\" muted playsinline preload=\"metadata\" hidden></video>\n        <div class=\"ccw-preview-gradient\"></div>\n        <div class=\"ccw-detail-overlay\" id=\"ccwDetailOverlay\" tabindex=\"0\" aria-label=\"문구 위치 드래그\">영상에 들어갈 문구</div>\n      </div>\n    </div>\n    <div class=\"ccw-detail-controls\">\n      <div><label for=\"ccwDetailCopy\">본문 문구 <span class=\"ccw-small\">기본형과 내용 공유</span></label>\n        <textarea id=\"ccwDetailCopy\" maxlength=\"220\" rows=\"3\" placeholder=\"본문을 입력해줘\"></textarea>\n      </div>\n      <div>\n        <label for=\"ccwFontSize\">글씨 크기 <output id=\"ccwSizeLabel\" for=\"ccwFontSize\">76px</output></label>\n        <input id=\"ccwFontSize\" type=\"range\" min=\"40\" max=\"110\" step=\"1\" value=\"76\">\n      </div>\n      <div class=\"ccw-half\">\n        <label for=\"ccwColor\">글씨 색상 <input id=\"ccwColor\" type=\"color\" value=\"#ffffff\"></label>\n        <label for=\"ccwFit\">영상 맞춤 <select id=\"ccwFit\" style=\"background:#20303a;color:#fff;padding:8px;border-radius:8px;border:1px solid #455562\"><option value=\"contain\">전체 보기</option><option value=\"cover\">꽉 채우기</option></select></label>\n      </div>\n      <div><label for=\"ccwY\">글씨 위치 <span class=\"ccw-small\">왼쪽에서 직접 드래그 가능</span></label>\n        <input id=\"ccwY\" type=\"range\" min=\"15\" max=\"87\" value=\"72\"></div>\n      <div class=\"ccw-small\">기본형은 돋보기 POP 76px, 가운데 정렬·흰색 기준이야. 세부 편집을 적용한 경우에만 크기·색·위치 설정을 사용해.</div>\n    </div>\n  </div>\n  <div class=\"ccw-detail-footer\">\n    <span class=\"ccw-small\">1080 × 1350 · H.264/AAC MP4 예정 · 실제 출력 미연결</span>\n    <button class=\"ccw-secondary\" id=\"ccwResetStyle\" type=\"button\">기본형으로 되돌리기</button>\n    <button id=\"ccwApplyStyle\" type=\"button\">설정 적용</button>\n  </div>\n</dialog>\n");
const state={mode:'full',body:'',advanced:false,size:76,color:'#ffffff',x:50,y:72,fit:'contain',previewUrl:null,videoDuration:null};
const trimActions=$('ccwTrimActions'),dialog=$('ccwDetailDialog');
const draft={size:76,color:'#ffffff',x:50,y:72,fit:'contain'};
function timeParse(s){
  if(!/^\d{1,3}:\d{2}:\d{2}$/.test(s))return null;
  const [h,m,sec]=s.split(':').map(Number);
  if(m>59||sec>59)return null;
  return h*3600+m*60+sec;
}
function timeFormat(t) { t=Math.max(0,Math.floor(t));return [Math.floor(t/3600),Math.floor(t/60)%60,t%60].map(n=>String(n).padStart(2,'0')).join(':'); }
function range(){
  const start=timeParse($('ccwStart').value.trim()),end=timeParse($('ccwEnd').value.trim());
  const valid=start!==null&&end!==null&&end>start&&(state.videoDuration===null||end<=state.videoDuration+1);
  return {start,end,valid};
}
function setStatus(message,error=false){
  const box=$('ccwInlineStatus');box.hidden=false;box.textContent=message;box.dataset.error=error?'true':'false';
}
function refreshRange(){
  const r=range();
  $('ccwStart').setAttribute('aria-invalid',r.start===null?'true':'false');
  $('ccwEnd').setAttribute('aria-invalid',r.end===null||r.start!==null&&r.end<=r.start?'true':'false');
  $('ccwDurationRow').dataset.error=r.valid?'false':'true';
  $('ccwDuration').textContent=r.valid?'선택 '+timeFormat(r.end-r.start):'시간을 확인해줘';
  $('ccwTimeHint').textContent=state.videoDuration===null?'원본 길이는 영상 연결 후 확인돼.':'미리보기 파일 '+timeFormat(state.videoDuration);
}
function updatePreview(){
  const txt=state.body.trim(),style=state.advanced?state:{size:76,color:'#ffffff',x:50,y:72,fit:'contain'};
  const inline=$('ccwPreviewText'),detail=$('ccwDetailOverlay');
  inline.hidden=!txt;
  inline.textContent=txt;
  detail.textContent=txt||'영상에 들어갈 문구';
  const scale=239/1080;
  detail.style.fontSize=(draft.size*scale).toFixed(2)+'px';
  detail.style.color=draft.color;
  detail.style.left=draft.x+'%';
  detail.style.top=draft.y+'%';
  inline.style.color=style.color;
  inline.style.fontSize=(style.size*250/1080).toFixed(2)+'px';
  inline.style.left=(style.x-45)+'%';
  inline.style.right=(100-style.x-45)+'%';
  inline.style.bottom=state.advanced?(100-style.y-8)+'%':'7%';
  inline.style.textAlign='center';
  $('ccwCopyStyle').textContent=state.advanced?'세부 편집 스타일 적용':'기본 크기·위치 자동 적용';
  $('ccwVideo').style.objectFit='contain';
  $('ccwDetailVideo').style.objectFit='contain';
}
function modeChange(next){
  state.mode=next;
  const part=next==='trim';
  $('ccwAll').setAttribute('aria-pressed',part?'false':'true');
  $('ccwPart').setAttribute('aria-pressed',part?'true':'false');
  $('ccwEdit').hidden=!part;
  originalAction.hidden=part;trimActions.hidden=!part;
  $('ccwModeHint').textContent=part?'구간 선택 후 본문을 넣거나 세부 편집으로 이동할 수 있어.':'전체 영상은 기존 방식대로 다운로드해.';
  if(part)refreshRange();
}
function bodyChange(s){
  state.body=String(s).slice(0,220);
  $('ccwCopy').value=state.body;
  $('ccwDetailCopy').value=state.body;
  updatePreview();
}
$('ccwAll').addEventListener('click',()=>modeChange('full'));
$('ccwPart').addEventListener('click',()=>modeChange('trim'));
$('ccwStart').addEventListener('input',refreshRange);
$('ccwEnd').addEventListener('input',refreshRange);
$('ccwStart').addEventListener('blur',()=>{
 const t=timeParse($('ccwStart').value.trim());if(t!==null)$('ccwStart').value=timeFormat(t);
});
$('ccwEnd').addEventListener('blur',()=>{
 const t=timeParse($('ccwEnd').value.trim());if(t!==null)$('ccwEnd').value=timeFormat(t);
});
$('ccwBodyToggle').addEventListener('click',()=>{
 const section=$('ccwCopySection'),show=section.hidden;
 section.hidden=!show;
 $('ccwBodyToggle').setAttribute('aria-expanded',String(show));
 $('ccwBodyToggle').textContent=show?'본문 입력 닫기':'+ 본문 넣기';
 if(show)$('ccwCopy').focus();
});
$('ccwCopy').addEventListener('input',e=>bodyChange(e.target.value));
$('ccwDetailCopy').addEventListener('input',e=>bodyChange(e.target.value));
function openDetail(){
 $('ccwDetailCopy').value=state.body;
 Object.assign(draft,{size:state.size,color:state.color,x:state.x,y:state.y,fit:state.fit});
 $('ccwFontSize').value=String(draft.size);
 $('ccwSizeLabel').textContent=state.size+'px';
 $('ccwColor').value=state.color;
 $('ccwY').value=String(state.y);
 $('ccwFit').value=state.fit;
 updatePreview();
 dialog.showModal();
}
$('ccwDetailOpen').addEventListener('click',openDetail);
$('ccwDetailClose').addEventListener('click',()=>dialog.close());
function syncControls(){
 draft.size=Number($('ccwFontSize').value);
 draft.color=$('ccwColor').value;
 draft.y=Number($('ccwY').value);
 draft.fit='contain';
 $('ccwSizeLabel').textContent=draft.size+'px';
 updatePreview();
}
['ccwFontSize','ccwColor','ccwY','ccwFit'].forEach(id=>$(id).addEventListener('input',syncControls));
let drag=null;
$('ccwDetailOverlay').addEventListener('pointerdown',e=>{
  const el=e.currentTarget;drag={x:e.clientX,y:e.clientY,left:draft.x,top:draft.y};el.setPointerCapture(e.pointerId);
});
$('ccwDetailOverlay').addEventListener('pointermove',e=>{
  if(!drag)return;
  const rect=$('ccwDetailPreview').getBoundingClientRect();
  draft.x=Math.max(10,Math.min(90,drag.left+(e.clientX-drag.x)/rect.width*100));
  draft.y=Math.max(15,Math.min(87,drag.top+(e.clientY-drag.y)/rect.height*100));
  $('ccwY').value=String(Math.round(draft.y));
  updatePreview();
});
['pointerup','pointercancel','lostpointercapture'].forEach(ev=>$('ccwDetailOverlay').addEventListener(ev,()=>{drag=null;}));
function resetStyle(){
 state.advanced=false;state.size=76;state.color='#ffffff';state.x=50;state.y=72;state.fit='contain';
 Object.assign(draft,{size:76,color:'#ffffff',x:50,y:72,fit:'contain'});
 $('ccwFontSize').value='76';$('ccwColor').value='#ffffff';$('ccwY').value='72';$('ccwFit').value='contain';
 $('ccwSizeLabel').textContent='76px';
 updatePreview();
}
$('ccwResetStyle').addEventListener('click',()=>{resetStyle();dialog.close();setStatus('기본형을 적용했어. 본문은 그대로 유지돼.');});
$('ccwApplyStyle').addEventListener('click',()=>{
 Object.assign(state,draft);
 state.advanced=true;updatePreview();dialog.close();
 $('ccwCopySection').hidden=false;
 $('ccwBodyToggle').setAttribute('aria-expanded','true');
 $('ccwBodyToggle').textContent='본문 입력 닫기';
 setStatus('세부 편집 설정을 반영했어. MP4 출력은 아직 연결되지 않았어.');
});
function uiOnly(which){
 const r=range();
 if(!r.valid){setStatus('시간을 HH:MM:SS 형식으로 입력하고 시작보다 종료를 늦게 지정해줘.',true);$('ccwStart').focus();return;}
 if(!$('urlInput').value.trim()&&!state.previewUrl){setStatus('영상 링크를 먼저 넣어줘.',true);$('urlInput').focus();return;}
 if(which==='carousel'&&!state.body.trim()){setStatus('영상 캐러셀을 만들려면 본문을 입력해줘.',true);if($('ccwCopySection').hidden)$('ccwBodyToggle').click();$('ccwCopy').focus();return;}
 setStatus('입력한 구간 '+timeFormat(r.start)+' ~ '+timeFormat(r.end)+' 확인 완료 · 지금은 UI 시제품이라 '+(which==='carousel'?'1080×1350 MP4 제작':'구간 다운로드')+'은 아직 실행되지 않아.');
}
$('ccwTrimOnly').addEventListener('click',()=>uiOnly('trim'));
$('ccwCreateCarousel').addEventListener('click',()=>uiOnly('carousel'));
$('ccwChooseLocal').addEventListener('click',()=>$('ccwLocalFile').click());
$('ccwLocalFile').addEventListener('change',e=>{
 const file=e.target.files?.[0];
 if(!file)return;
 if(!/^video\//.test(file.type)&&!/\.(mp4|mov|mkv|webm)$/i.test(file.name)){
  setStatus('동영상 파일을 선택해줘.',true);return;
 }
 if(state.previewUrl)URL.revokeObjectURL(state.previewUrl);
 state.previewUrl=URL.createObjectURL(file);
 $('ccwVideo').src=state.previewUrl;$('ccwVideo').hidden=false;
 $('ccwDetailVideo').src=state.previewUrl;$('ccwDetailVideo').hidden=false;
 $('ccwPreviewEmpty').hidden=true;$('ccwDetailEmpty').hidden=true;
 state.videoDuration=null;
 $('ccwVideo').onloadedmetadata=()=>{
  state.videoDuration=Number.isFinite($('ccwVideo').duration)?$('ccwVideo').duration:null;
  refreshRange();
 };
 $('ccwDetailVideo').onloadedmetadata=()=>{updatePreview();};
 setStatus('로컬 파일을 미리보기로 불러왔어. 실제 다운로드 기능과는 연결되지 않아.');
});
dialog.addEventListener('click',e=>{if(e.target===dialog)dialog.close();});
window.addEventListener('beforeunload',()=>{if(state.previewUrl)URL.revokeObjectURL(state.previewUrl);});
window.ClipCatchTrimUI={
 parseTime:timeParse,
 formatTime:timeFormat,
 getDraft:()=>({mode:state.mode,start:$('ccwStart').value,end:$('ccwEnd').value,body:state.body,advanced:state.advanced,size:state.size,color:state.color,x:state.x,y:state.y,fit:state.fit})
};
modeChange('full');bodyChange('');
})();