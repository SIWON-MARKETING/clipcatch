import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';
const source=readFileSync(new URL('../src/web/inline-download-ui.js',import.meta.url),'utf8');
function fakeUI(){
  const map=new Map(), events=new Map();
  const create=(id)=> {
    if(map.has(id))return map.get(id);
    const item={
      id,hidden:false,value:'',dataset:{},style:{},textContent:'',
      attributes:{},listeners:{},aria:{},onloadedmetadata:null,
      setAttribute(k,v){this.attributes[k]=String(v);},
      getAttribute(k){return this.attributes[k]||null;},
      addEventListener(k,fn){(this.listeners[k]??=[]).push(fn);},
      dispatch(k,data={}){for(const fn of this.listeners[k]||[])fn({target:this,currentTarget:this,...data});},
      click(){this.dispatch('click');},
      focus(){this.focused=true;},
      insertAdjacentHTML(where,html){for(const match of html.matchAll(/id="([^"]+)"/g)){create(match[1]);}},
      getBoundingClientRect(){return {left:0,top:0,width:239,height:299};},
      showModal(){this.open=true;},
      close(){this.open=false;},
      setPointerCapture(){},
      classList:{add(){},remove(){},toggle(){}}
    };
    map.set(id,item);return item;
  };
  const entry=create('entry'),info=create('info'),row=create('originalAction');
  entry.querySelector=select=>select==='.info-line'?info:select==='.action-row'?row:null;
  create('downloadBtn');create('urlInput');
  const doc={
    head:{append(){}},
    body:{insertAdjacentHTML:entry.insertAdjacentHTML},
    createElement:tag=>create(tag+'-created'),
    querySelector:selector=>selector==='.main-card .entry'?entry:null,
    getElementById:id=>map.get(id)||null
  };
  const sandbox={document:doc,window:{addEventListener(){}},URL:class FakeURL{},console};
  runInNewContext(source,sandbox);
  // Minimal mock DOM must honor the two input default values from HTML.
  create('ccwStart').value='00:00:00';
  create('ccwEnd').value='00:00:30';
  return {map,create,win:sandbox.window};
}
test('existing full-video download remains visible and trim is inside the video-download form',()=>{
 const {create,win}=fakeUI();
 assert.equal(create('originalAction').hidden,false);
 assert.equal(create('ccwEdit').hidden,true);
 assert.equal(create('ccwTrimActions').hidden,true);
 assert.equal(create('ccwAll').attributes['aria-pressed'],'true');
 create('ccwPart').click();
 assert.equal(create('ccwEdit').hidden,false);
 assert.equal(create('originalAction').hidden,true);
 assert.equal(create('ccwTrimActions').hidden,false);
 assert.equal(win.ClipCatchTrimUI.getDraft().mode,'trim');
 create('ccwAll').click();
 assert.equal(create('originalAction').hidden,false);
 assert.equal(create('ccwTrimActions').hidden,true);
});
test('start/end validation and duration operate without downloading',()=>{
 const {create,win}=fakeUI();
 create('ccwPart').click();
 assert.equal(win.ClipCatchTrimUI.parseTime('01:02:03'),3723);
 assert.equal(win.ClipCatchTrimUI.parseTime('00:61:00'),null);
 assert.equal(win.ClipCatchTrimUI.parseTime('x'),null);
 assert.equal(win.ClipCatchTrimUI.formatTime(3723),'01:02:03');
 create('ccwStart').value='00:01:10';
 create('ccwEnd').value='00:01:00';
 create('ccwStart').dispatch('input');
 assert.equal(create('ccwDurationRow').dataset.error,'true');
 create('ccwEnd').value='00:01:40';
 create('ccwEnd').dispatch('input');
 assert.equal(create('ccwDuration').textContent,'선택 00:00:30');
});
test('basic caption and detailed editor share text; Cancel leaves applied styling intact',()=>{
 const {create,win}=fakeUI();
 create('ccwPart').click();
 create('ccwBodyToggle').click();
 create('ccwCopy').value='본문 테스트';
 create('ccwCopy').dispatch('input');
 assert.equal(create('ccwDetailCopy').value,'본문 테스트');
 create('ccwDetailOpen').click();
 assert.equal(create('ccwDetailDialog').open,true);
 create('ccwFontSize').value='95';
 create('ccwFontSize').dispatch('input');
 create('ccwDetailClose').click();
 assert.equal(win.ClipCatchTrimUI.getDraft().size,76);
 assert.equal(win.ClipCatchTrimUI.getDraft().advanced,false);
 create('ccwDetailOpen').click();
 create('ccwFontSize').value='92';
 create('ccwFontSize').dispatch('input');
 create('ccwApplyStyle').click();
 assert.equal(win.ClipCatchTrimUI.getDraft().size,92);
 assert.equal(win.ClipCatchTrimUI.getDraft().advanced,true);
 create('ccwResetStyle').click();
 assert.equal(win.ClipCatchTrimUI.getDraft().size,76);
 assert.equal(win.ClipCatchTrimUI.getDraft().advanced,false);
 assert.equal(create('ccwCopy').value,'본문 테스트');
});
test('trim and carousel buttons never invoke the existing download handler in UI-only phase',()=>{
 const {create}=fakeUI();
 let downloads=0;
 create('downloadBtn').addEventListener('click',()=>downloads++);
 create('ccwPart').click();
 create('urlInput').value='https://example.com/video';
 create('ccwTrimOnly').click();
 assert.match(create('ccwInlineStatus').textContent,/아직 실행되지 않아/);
 assert.equal(downloads,0);
 create('ccwCreateCarousel').click();
 assert.match(create('ccwInlineStatus').textContent,/본문을 입력해줘/);
 assert.equal(downloads,0);
});
