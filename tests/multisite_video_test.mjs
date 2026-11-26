import { test } from 'node:test';
import assert from 'node:assert/strict';
import { videoPage, supportedSite, buildCandidates } from '../src/browser_extension/core.mjs';

test('Naver Clip and Assembly public video links are recognized', () => {
  const cases = [
    ['https://m.naver.com/shorts/?mediaId=8A94DF43BEECAA92D927198B8C5F07A0D6FB&serviceType=CLIP', 'Naver Clip'],
    ['https://clip.naver.com/contents?recType=CLIP_PC&seedMediaId=42C87377808E4ABD5FE30AF7884589909A7F', 'Naver Clip'],
    ['https://tv.naver.com/v/60223053', 'Naver Video'],
    ['https://w3.assembly.go.kr/main/player.do?menu=30&mc=345&ct1=22&ct2=439&ct3=V2&wv=1&', 'Assembly Video'],
  ];
  for (const [url, platform] of cases) {
    assert.equal(videoPage(url)?.platform, platform);
    assert.equal(supportedSite(url), platform);
    assert.equal(buildCandidates({ page_url: url, records: [] }).length, 1);
  }
});

test('reject unrelated pages, missing media IDs and host lookalikes', () => {
  const bad = [
    'https://m.naver.com/shorts/',
    'https://m.naver.com/shorts/?mediaId=not-media',
    'https://clip.naver.com/@profile',
    'https://clip.naver.com/contents?seedMediaId=bad',
    'https://evil-clip.naver.com/contents?seedMediaId=42C87377808E4ABD5FE30AF7884589909A7F',
    'https://w3.assembly.go.kr/main/player.do?mc=345&ct1=22&ct2=439',
    'https://w3.assembly.go.kr/admin',
    'https://evilw3.assembly.go.kr/main/player.do?mc=345&ct1=22&ct2=439&ct3=V2',
    'http://127.0.0.1:8765/',
  ];
  for (const url of bad) assert.equal(videoPage(url), null, url);
});