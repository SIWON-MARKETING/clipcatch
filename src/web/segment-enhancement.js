// Progressive enhancement: keep the existing video download panel intact.
// Split six time fields, apply the GitHub dotbogi-a body layout in Basic mode,
// and connect only the Preview segment buttons to the real local job API.
(() => {
  'use strict';
  const $=id=>document.getElementById(id);
  if (!$('ccwWorkflow')||!$('ccwStart')||!$('ccwEnd')||!window.ClipCatchTrimUI) return;

  const start=$('ccwStart'),end=$('ccwEnd'),container=document.querySelector('.ccw-times');
  const clock=(prefix,label,value)=> {
    const group=document.createElement('div');
    group.className='ccw-clock';group.setAttribute('role','group');group.setAttribute('aria-label',label+' 시간');
    const caption=document.createElement('div');caption.className='ccw-clock-title';caption.textContent=label;group.append(caption);
    const cells=document.createElement('div');cells.className='ccw-clock-cells';
    const vals=value.split(':');
    for(const [i,part] of ['시','분','초'].entries()){
      const wrap=document.createElement('label');wrap.className='ccw-clock-part';
      const field=document.createElement('input');
      field.id='ccw'+prefix+['Hour','Minute','Second'][i];
      field.type='number';field.inputMode='numeric';field.min='0';field.max=i===0?'23':'59';
      field.step='1';field.value=String(Number(vals[i]));field.setAttribute('aria-label',label+' '+part);
      const unit=document.createElement('span');unit.textContent=part;
      wrap.append(field,unit);cells.append(wrap);
      if(i<2){const dot=document.createElement('span');dot.className='ccw-clock-colon';dot.textContent=':';cells.append(dot);}
      field.addEventListener('input',()=>syncClock());
      field.addEventListener('blur',()=>{
        const n=Number(field.value);
        if(Number.isFinite(n)&&n>=0&&n<=Number(field.max))field.value=String(Math.floor(n));
        syncClock();
      });
      field.addEventListener('keydown',ev=>{
        if(ev.key==='ArrowRight'&&i<2){ev.preventDefault();$('ccw'+prefix+['Minute','Second'][i]).focus();}
      });
    }
    group.append(cells);return group;
  };
  const sh=clock('Start','시작',start.value),eh=clock('End','종료',end.value);
  const tilde=document.createElement('span');tilde.className='ccw-between';tilde.textContent='→';tilde.setAttribute('aria-hidden','true');
  start.type='hidden';end.type='hidden';
  container.replaceChildren(start,end,sh,tilde,eh);
  const parts=prefix=>[...document.querySelectorAll('#ccw'+prefix+'Hour, #ccw'+prefix+'Minute, #ccw'+prefix+'Second')];
  function seconds(prefix){
    const fields=parts(prefix);const arr=fields.map(f=>f.value.trim());
    if(arr.some(x=>x===''||!/^\\d{1,2}$/.test(x)))return null;
    const [h,m,s]=arr.map(Number);
    if(h>23||m>59||s>59)return null;
    return 3600*h+60*m+s;
  }
  function syncClock(){
    const a=seconds('Start'),b=seconds('End');
    start.value=a===null?'':window.ClipCatchTrimUI.formatTime(a);
    end.value=b===null?'':window.ClipCatchTrimUI.formatTime(b);
    start.dispatchEvent(new Event('input',{bubbles:true}));
    end.dispatchEvent(new Event('input',{bubbles:true}));
    const valid=a!==null&&b!==null&&a<b&&b-a<=3600;
    for(const f of [...parts('Start'),...parts('End')])f.setAttribute('aria-invalid',String(!valid));
    if(a!==null&&b!==null&&b-a>3600)$('ccwTimeHint').textContent='한 번에 최대 60분까지 지정할 수 있어.';
    return {start:a,end:b,valid};
  }
  syncClock();

  const css=document.createElement('link');css.rel='stylesheet';css.href='/segment-enhancement.css';document.head.append(css);
  const note=$('ccwModeHint');note.textContent='한 구간을 지정한 뒤 원본 저장하거나 영상 캐러셀을 만들 수 있어.';
  const captionNote=$('ccwCopyStyle');if(captionNote)captionNote.textContent='돋보기 A 내지 기본형 자동 배치';
  const statusHelp=$('ccwEdit').querySelector('.ccw-edit-notice');
  if(statusHelp)statusHelp.textContent='시간을 선택하고 다운로드해줘. 영상 캐러셀은 ONE Mobile POP 글꼴이 PC에 설치돼 있어야 해.';
  const small=$('ccwTrimActions')?.querySelector('small');
  if(small)small.textContent='저장 진행 상황은 아래 작업 목록에서 확인해.';
  const previewInfo=$('ccwPreviewEmpty');
  if(previewInfo)previewInfo.textContent='원본 링크 미리보기는 추후 지원 · 로컬 영상은 아래에서 선택 가능';
  const detailInfo=$('ccwDetailEmpty');
  if(detailInfo)detailInfo.textContent='로컬 영상으로 미리보기 가능';

  // Layout is based on the GitHub-owned dotbogi-a body template:
  // 1080x1350; 1008 px body width; 76 px POP; lineHeight 1.2;
  // 18px top, 58px bottom, automatic yellow 01.
  // This is a template-based Preview, not a claim of pixel-exact Figma Bridge parity.
  function approxWidth(line,size){
    return [...line].reduce((n,ch)=>{
      const cp=ch.codePointAt(0);
      return n+(ch===' '?0.34:cp<128?0.55:1)*size;
    },0);
  }
  function bodyGeometry(text){
    let size=76;const lines=text.replace(/\r/g,'').split('\n');
    while(size>40 && lines.some(l=>approxWidth(l,size)>1008))size-=2;
    const height=Math.ceil(Math.max(1,lines.length)*size*1.2);
    const numberHeight=Math.ceil(32*1.2);
    const footer=18+numberHeight+8+height+58;
    return {size,numberY:1350-footer+18,
      bodyY:1350-footer+18+numberHeight+8,photoY:1350-footer};
  }
  function addNumber(frame,id){
    const el=document.createElement('div');el.id=id;el.className='ccw-bridge-number';el.textContent='01';
    frame.append(el);return el;
  }
  const inlineNum=addNumber($('ccwFrame'),'ccwInlineNumber');
  const detailNum=addNumber($('ccwDetailPreview'),'ccwDetailNumber');
  function bridgeStyle(){
    const draft=window.ClipCatchTrimUI.getDraft();
    const body=(draft.body||'').trim();
    const layout=bodyGeometry(body||' ');
    const basic=!draft.advanced;
    for(const [frame,label,number,frameWidth] of [
      [$('ccwFrame'),$('ccwPreviewText'),inlineNum,Number($('ccwFrame').getBoundingClientRect().width)||130],
      [$('ccwDetailPreview'),$('ccwDetailOverlay'),detailNum,Number($('ccwDetailPreview').getBoundingClientRect().width)||239]
    ]){
      if(!frame||!label||!number)continue;
      frame.classList.toggle('ccw-bridge-basic',basic);
      number.hidden=!body||!basic;
      const posPct=100*layout.numberY/1350;
      number.style.top=posPct+'%';
      number.style.fontSize=(32*frameWidth/1080)+'px';
      number.style.left='50%';
      const shade=frame.querySelector('.ccw-preview-gradient');
      if(shade&&basic) {
        shade.style.top=(100*(layout.photoY-90)/1350)+'%';
        shade.style.height='auto';shade.style.bottom='0';
        shade.style.background='linear-gradient(to bottom,transparent,#0e0e10 17%,#0e0e10 100%)';
      } else if(shade) {
        shade.style.top='auto';shade.style.height='42%';shade.style.background='linear-gradient(transparent,rgba(0,0,0,.9))';
      }
      if(basic){
        label.style.setProperty('top',(100*layout.bodyY/1350)+'%','important');
        label.style.setProperty('bottom','auto','important');
        label.style.setProperty('font-size',(layout.size*frameWidth/1080)+'px','important');
        label.style.setProperty('line-height','1.2','important');
        label.style.setProperty('color','#ffffff','important');
        label.style.setProperty('left','50%','important');
        label.style.setProperty('right','auto','important');
        label.style.setProperty('width',(1008/1080*100)+'%','important');
        label.style.setProperty('transform','translateX(-50%)','important');
      }else{
        for(const key of ['top','bottom','font-size','line-height','color','left','right','width','transform'])label.style.removeProperty(key);
      }
    }
    const captionLabel=$('ccwCopyStyle');
    if(captionLabel)captionLabel.textContent=basic?'돋보기 A 내지 기본형 · 위치/크기 자동':'세부 편집값 적용';
  }
  function afterExisting(){queueMicrotask(bridgeStyle);}
  const watched=['ccwCopy','ccwDetailCopy','ccwResetStyle','ccwApplyStyle','ccwDetailOpen',
    'ccwFontSize','ccwColor','ccwY','ccwFit','ccwBodyToggle','ccwPart','ccwAll'];
  for(const id of watched){
    const el=$(id);if(!el)continue;
    for(const type of ['input','click'])el.addEventListener(type,afterExisting);
  }
  // Basic form remains visually locked even inside the Advanced dialog,
  // until the designer changes a control; then the draft advanced preview takes over.
  let editTouched=false;
  $('ccwDetailOpen').addEventListener('click',()=>{editTouched=false;afterExisting();});
  for(const id of ['ccwFontSize','ccwColor','ccwY','ccwFit']){
    $(id).addEventListener('input',()=>{editTouched=true;const el=$('ccwDetailOverlay');
      for(const key of ['top','bottom','font-size','line-height','color','left','right','width','transform'])el.style.removeProperty(key);
      detailNum.hidden=true;
    });
  }
  function bridgeModalMode(){
    if($('ccwDetailDialog').open&&editTouched)return;
    bridgeStyle();
  }
  $('ccwDetailOpen').addEventListener('click',()=>queueMicrotask(bridgeModalMode));

  // The previous UI-only buttons have placeholder handlers. Clone only
  // the two action buttons to remove those callbacks, not the real full-download button.
  function cleanButton(id){
    const el=$(id),copy=el.cloneNode(true);
    el.replaceWith(copy);
    return copy;
  }
  const trimButton=cleanButton('ccwTrimOnly');
  const carouselButton=cleanButton('ccwCreateCarousel');
  let busy=false,currentJob='',poller=null;
  function feedback(message,error=false){
    const el=$('ccwInlineStatus');el.hidden=false;el.textContent=message;el.dataset.error=String(error);
  }
  async function submit(kind){
    if(busy)return;
    const r=syncClock();
    if(!r.valid){feedback('시·분·초를 확인해줘. 종료 시간이 시작보다 늦어야 해.',true);return}
    const links=$('urlInput').value.match(/https?:\/\/[^\s]+/g)||[];
    if(links.length!==1){feedback('영상 링크는 한 개만 입력해줘.',true);return}
    const st=window.ClipCatchTrimUI.getDraft();
    if(kind==='carousel'&&!String(st.body).trim()){
      feedback('영상 캐러셀에는 본문 문구가 필요해.',true);
      if($('ccwCopySection').hidden)$('ccwBodyToggle').click();
      $('ccwCopy').focus();return;
    }
    const payload={
      url:links[0],start_seconds:r.start,end_seconds:r.end,kind,
      caption:kind==='carousel'?st.body:'',
      advanced:!!st.advanced,size:st.size,color:st.color,x:st.x,y:st.y,
      fit:st.fit
    };
    busy=true;trimButton.disabled=carouselButton.disabled=true;
    feedback('구간 작업을 등록하는 중...');
    try{
      const response=await fetch('/api/segment',{
        method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},
        body:JSON.stringify(payload)});
      let data;try{data=await response.json()}catch{data={error:'서버 응답을 읽지 못했어'}}
      if(!response.ok)throw Error(data.error||'작업 등록 실패: HTTP '+response.status);
      currentJob=data.job_id;
      feedback(kind==='clip'?'구간 다운로드가 시작됐어.':'구간 다운로드 후 영상 캐러셀을 제작할게.');
      clearInterval(poller);
      poller=setInterval(pollJob,1700);await pollJob();
    }catch(e){
      feedback(e.message||'다운로드 요청에 실패했어.',true);
    }finally{
      busy=false;trimButton.disabled=carouselButton.disabled=false;
    }
  }
  async function pollJob(){
    if(!currentJob)return;
    try{
      const resp=await fetch('/api/state',{credentials:'same-origin',cache:'no-store'});
      if(!resp.ok)return;
      const data=await resp.json();const job=(data.jobs||[]).find(j=>j.id===currentJob);
      if(!job)return;
      if(job.status==='done'){
        feedback('저장 완료 · '+(job.file||'파일을 저장했어.')+' · 아래 [저장 폴더]에서 확인해줘.');
        clearInterval(poller);poller=null;currentJob='';
      }else if(job.status==='error'||job.status==='cancelled'){
        feedback((job.error_title||job.detail||'영상 작업이 실패했어.'),true);
        clearInterval(poller);poller=null;currentJob='';
      }else{
        feedback((job.detail||'영상 다운로드 중...')+
          (Number.isFinite(job.percent)?' '+Math.round(job.percent)+'%':''));
      }
    }catch{}
  }
  trimButton.addEventListener('click',()=>submit('clip'));
  carouselButton.addEventListener('click',()=>submit('carousel'));
  // Font presence is checked by Preview backend; do not silently replace the
  // brand font in output with a different Windows font.
  fetch('/api/segment/preset',{credentials:'same-origin'}).then(r=>r.json()).then(d=>{
    const note=$('ccwPreviewNote')||document.querySelector('.ccw-preview-note');
    if(note)note.textContent=d.installed_font
      ?'돋보기 A 내지 배치 · 로컬 POP 사용'
      :'ONE Mobile POP 글꼴 미설치 · 캐러셀 출력 전 설치 필요';
  }).catch(()=>{});
  window.ClipCatchSegmentV2={
    clock:()=>({start:seconds('Start'),end:seconds('End')}),
    geometry:bodyGeometry,
    submit
  };
  bridgeStyle();
})();
