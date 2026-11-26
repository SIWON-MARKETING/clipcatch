# GitHub 조직 이전 후 자동 업데이트 복구

상태: 2026-11-26 v2.0.1 정식 배포 완료. PR과 main의 Windows·미디어 검사, 공개 실행파일 일치 확인, 구버전 4종의 수동 교체 검증을 통과했다.

## 배포 및 공개 검증 기록

- [정식 v2.0.1](https://github.com/SIWON-MARKETING/clipcatch/releases/tag/v2.0.1)
- [수정 PR #26](https://github.com/SIWON-MARKETING/clipcatch/pull/26): 정식판 병합 완료
- [main 빌드·설치·교체·배포 검사](https://github.com/SIWON-MARKETING/clipcatch/actions/runs/45803665818): 통과
- [게시 후 실제 공개 교체 경로](https://github.com/SIWON-MARKETING/clipcatch/actions/runs/45804657669): 통과
- 배포 소스: `811b545676f58999e0caed91031517b87a55cd7c`; 검증한 소스 트리: `f1a3e766d20a174e04f6f5dfdb2cd41585115531`
- 공개 `ClipCatch.exe` SHA-256: `58d2bc37a1faab688b0bb05e8a03ea20dd2052c7af48e9a6e58c783438ae0098`
- 공식 공개 주소에서 EXE·build-info·SHA256SUMS를 직접 내려받아 GitHub 자산 digest 및 체크섬 일치를 확인했다. EXE의 Go 빌드 정보도 위 소스 커밋과 `vcs.modified=false`로 일치한다.
- 공개 build-info는 정식 앱 `clipcatch`, 버전 `2.0.1`, 새 조직의 최신 EXE 공유 주소를 갖는다.
- Windows 실제 실행 환경에서 공개 v2.0.0·1.9.9·1.9.8·1.9.6을 실행한 뒤 새 EXE를 실행하는 경로로 v2.0.1 교체, 기록·설정·원본 보존, Chrome 버튼을 통한 정확한 폴더 열기를 확인했다. 이 공개 교체 검사에서는 업데이트 헬퍼를 직접 호출하지 않았다.
- 기존 브랜치 보존 및 Preview 초안 [PR #25](https://github.com/SIWON-MARKETING/clipcatch/pull/25)의 초안·미병합 상태를 확인했다.
- 위 검사는 GitHub의 Windows 환경에서 수행했다. 사용자 PC나 미래의 다음 정식 버전으로의 공개 업데이트를 실측한 것으로 표시하지 않는다.

- 대상 저장소: `SIWON-MARKETING/clipcatch`
- 기준 브랜치: `main`, 커밋 `0100b1a928009f7c7e9a186216a05284be41f5c2`
- 수정 브랜치: `fix/github-organization-updater`
- 정식 버전: `2.0.1`. PR 검증 후 main에 병합했으며, main의 독립적인 빌드·검증을 통과한 실행파일을 정식 릴리스로 게시했다.

## 원인과 수정

기존 실행파일은 `shindongyoon-siwon/clipcatch`의 Release URL만 정확히 허용한다. 저장소 이전 후 GitHub API는 `SIWON-MARKETING/clipcatch` 주소로 자산을 반환한다. 이전 API 주소가 리디렉션되어도 다운로드 주소 검증에서 거부된다.

| 항목 | 수정 후 동작 |
| --- | --- |
| 정식 업데이트 조회·다운로드·공유 | `SIWON-MARKETING/clipcatch`를 기준으로 구성 |
| 실제 EXE·SHA256SUMS 주소 | 새 조직의 정확한 저장소·태그·파일명만 허용. 예전 소유자 주소를 설치 경로로 허용하지 않음 |
| 배포된 v2.0.0의 build-info.json | 옛 공유 주소를 버전 변경 힌트로만 읽음. 힌트만으로 설치하거나 업데이트를 강제하지 않음 |
| 과거 업데이트 캐시 | 옛 EXE/체크섬 주소가 있으면 재사용하지 않고 공식 정보를 다시 조회 |
| 업데이트 창·공유 버튼 | 새 조직 링크 사용. 임의 저장소·스크립트·변형 URL 차단 유지 |
| Windows 검증 | 옛 배포본을 새 조직 URL로 다운로드하고 기존 고정 SHA-256 확인. 실행 중인 실제 v2.0.0에서 교체 트랜잭션 검사 추가 |

SHA-256 검증, Windows 실행파일 형식 확인, 설치 전 `--build-info` 앱/채널/버전 검사, 교체 잠금·재시작 확인·실패 시 복구 절차를 유지한다. 정식 버전의 활동 시 확인·1시간 제한과 TEST2의 15분 간격도 변경하지 않는다.

## 운영판·TEST2·Preview 분리

| 구분 | 피드·자산 | 설치/데이터 폴더 | 포트 | 이번 처리 |
| --- | --- | --- | --- | --- |
| 운영판 | 정식 `vX.Y.Z`, `ClipCatch.exe` | `ClipCatch` | 8765 | 새 조직으로 갱신 |
| TEST2 | prerelease `test2-vX.Y.Z-test.N`, `ClipCatch-Test2.exe` | `ClipCatch-Test2` | 18766 | 공통 저장소 상수를 통해 새 조직 사용. 운영판·Preview 자산 거부 |
| Preview V2 | 독립 실행, 자동 업데이트 비활성화 | `ClipCatch-Preview-V2` | 18768 | 기존 Preview 브랜치를 읽어 분리 규칙 확인. 이 PR로 해당 브랜치나 기능을 합치지 않음 |

Preview 기준은 `test/clipcatch-preview-2.0.1-test.10` 커밋 `57fd6ae2030c522e625ddee93e5f8605584ab492`이다. 해당 소스는 앱 ID `clipcatch-preview-v2`, 독립 경로/포트, 업데이트 조회·설치 비활성화 조건을 갖는다. 시험판 브랜치에 이 수정을 나중에 옮길 경우 이 조건을 그대로 유지해야 한다. 이번 수정은 그 브랜치의 실행 검증을 뜻하지 않는다.

## 기존 v2.0.0 사용자 안내

**v2.0.1 정식 버전이 게시된 뒤 적용한다.** 예전 v2.0.0 실행파일을 다시 받아도 이 문제가 해결되지 않는다.

1. 진행 중인 영상 작업을 마치고 필요한 결과를 저장한다.
2. [공식 릴리스 페이지](https://github.com/SIWON-MARKETING/clipcatch/releases/latest)에서 v2.0.1 이상인지 확인하고 `ClipCatch.exe`를 받는다.
3. 받은 EXE를 실행한다. 새 실행파일이 공식 배포 정보·SHA-256을 확인한 뒤 기존 설치를 교체한다. 저장 폴더나 앱 데이터 폴더를 삭제할 필요는 없다.
4. 새 버전 실행과 기존 다운로드 기록·설정·저장 파일을 확인한다. 이후 업데이트는 새 조직 피드로 조회한다.

이전 EXE의 엄격한 검증 코드는 서버 응답이나 안내 문구로 바꿀 수 없다. SHA-256이나 앱 식별 검사를 꺼서 우회하지 않는다.

## 자동 검증과 한계

- 회귀 재현: 수정 전 코드에서 새 조직 자산 거부, 새 공유 주소 힌트 거부, 옛 설치 캐시 재사용을 확인했다.
- 로컬 결과는 아래와 같다. Go는 1.27.4/Linux에서 실행했다.

| 검증 | 결과 |
| --- | --- |
| `CLIPCATCH_BUILD_ID_TEST=1 go test -race -count=1 -timeout=5m -json ./...` | 통과 201건(하위 케이스 포함), 실패 0건. 미디어/외부 엔진 등 조건이 필요한 38건 건너뜀 |
| 새 조직 주소·옛 주소 거부·캐시 폐기·채널 분리 회귀 | 위 테스트에 포함해 통과 |
| 실제로 빌드한 실행파일의 `--build-info` | 올바른 운영판 허용, 다른 버전·TEST2 교차 설치 거부 통과 |
| Node 확장프로그램/피드/전송 상태 테스트 4개 파일 | 44건 통과, 실패/건너뜀 0건 |
| `go vet ./...` | 통과 |
| Windows amd64 테스트 바이너리·앱 교차 컴파일 | 통과. 실행하지 않음. 결과물은 검토용이며 배포하지 않음 |
| 변경 워크플로 YAML·Python 테스트 문법·JS 문법·공백 검사 | 통과 |
| `tests/update_dialog.py`, `tests/auto_update_ui.py`, `tests/workspace_ui.py` | 로컬 Linux는 소켓 제한으로 실행 불가. PR 및 main Windows Actions와 브라우저 전용 검사에서 통과 |
| Windows 실제 실행·설치·롤백·수동 교체 | PR 및 main 검사 통과. 게시 후 공개 구버전 4종의 새 EXE 실행 경로도 통과. 위 Actions 기록 참조 |

- PR용 Windows Actions는 운영판/TEST2 교체·롤백 및 Preview 경로·포트 보존, 실제 v2.0.0에서 직접 헬퍼를 통한 트랜잭션을 검사한다. 직접 헬퍼 검사는 사용자의 다운로드 EXE 실행 경로와 구분해 기록한다.
- 게시 후 별도 워크플로는 공개 자산이 검증한 파일과 같은지 확인한 뒤 실제 v2.0.0·1.9.9·1.9.8·1.9.6에서 새 EXE 실행 경로와 기록·설정·원본 보존을 검사한다. 조직 이전으로 동작할 수 없는 예전 앱의 업데이트 버튼을 성공 조건으로 요구하지 않는다.
- 수정판을 거친 이후 버전에서 다음 정식 버전으로 자동 업데이트하는 실제 공개 경로는 이후 릴리스가 있을 때 별도 확인해야 한다.
- 로컬 Linux 테스트나 Windows용 교차 컴파일을 Windows 실제 실행·사용자 PC 검증 완료로 표시하지 않는다.

GitHub Actions의 PR 이벤트는 검증만 수행한다. 기존 publish 조건인 `push` + `refs/heads/main`을 유지한다. 이번 정식 배포 승인 범위에 따라 검증된 수정 PR만 main에 병합하며, 기존 초안 PR이나 Preview/TEST2 기능은 함께 병합하지 않는다. 게시 후 공개 자산과 실제 구버전 수동 교체 경로를 추가 검사한다.
