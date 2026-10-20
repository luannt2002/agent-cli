# V3 — Quyết định tích hợp (overlay §24)

Trả lời: từ Context Mode, claude-mem và Hindsight, cái gì vào agent-cli V3, vì sao, issue nào chứng minh, fork nào cho ý
tốt hơn, cái gì bị bác; agent-cli đã tốt hơn ở đâu; phần nào agent-cli phải tự sở hữu; MINIMUM V3; OPTIONAL V3; độ phức
tạp bị bác.

**Viết tắt:** CM = Context Mode, MEM = claude-mem, HS = Hindsight; MAP = `cross-project/ISSUE_TO_INVARIANT_MAP.md` (họ
F01–F22); FIT = `cross-project/FIT_GAP.md`; WP = `WORKFLOW_PROGRESS.md` (số sự kiện).

**Phạm vi bằng chứng:**
- **Đã chứng minh**: có lệnh tái lập và kết quả trong `experiments/` hoặc `/home/luannt/.cache/v3ev/`.
- **Mới nghi**: kèm cách kiểm.
- **Không có mục nào ADOPT chỉ vì kiến trúc trông sạch hơn** (spec §77).

---

## 1. Kết luận một đoạn

Ba dự án ngoài không đóng góp được **cơ chế** nào vào V3. Chúng đóng góp **bất biến**: cùng một lớp lỗi tái diễn ở cả
ba, và agent-cli V2 có ít nhất một ca phơi nhiễm đo được ở 11/16 họ lỗi của spec §12
(`cross-project/01_FAILURE_TAXONOMY.md`; bộ test phơi nhiễm, WP 20, 87). V3 = V2 + mỗi bất biến được ép ở
**chủ sẵn có** của agent-cli (không thêm store, worker, DB hay event log). Mọi ứng viên "lớp trí nhớ" (artifact store
mới, digest, knowledge store, graph, reflect, observation LLM, event sourcing) bị **DEFER** hoặc **REJECT** vì không lỗi
nào của agent-cli đòi chúng và chưa có số đo cho thấy V2 hỏng ở chỗ chúng giải.

## 2. Cái gì vào V3 (đã gộp vào `main` local 20/10; chưa push)

Mỗi dòng: lỗi đo được trên V2 → sửa ở chủ cũ → bằng chứng đỏ→xanh → soát độc lập DAT. Commit là commit trong lịch sử `main` (trước là nhánh `v3-candidate`).

| POC | Lỗi trên V2 (đã tái hiện) | Bất biến (MAP) | Issue/fork ngoài chứng minh lớp lỗi | Sửa ở chủ cũ | Commit |
|---|---|---|---|---|---|
| P0-1 | 10 `run` đồng thời làm server panic `WaitGroup is reused` (T20, 27/200 lượt) | F10 STARTUP_RACE | CM sẵn sàng MCP sửa 4 vòng; MEM cold start sửa 5 lần | nhóm chờ "agent dựng xong" của coordinator | `fdd759fa` |
| P0-2 | client sau có allowlist chặt hơn vẫn chạy theo chính sách client đầu (C5) | F20 PROVIDER_POLICY_BYPASS | MEM #2201 (worker kế thừa cấu hình), #4727 | so chính sách tin cậy lúc tạo workspace, TỪ CHỐI khi khác (quyết định C5 của người dùng) | `3899c5b1` |
| P0-2b | `.env` của repo tới CLI con giao việc, `!cmd`, server do Home dựng | F19 SECURITY_BOUNDARY, F03 | MEM #3425 (observer tự bắt chính nó); CM denylist env | env gốc cho tiến trình con | `213411fb` |
| P0-3 | `agent-cli-tmux rm` xoá worktree khi session bận ở thư mục con (T4) | F11, F17 | CM dọn xoá dữ liệu sống | so tiền tố thay so bằng | `54ab374b` |
| P1-a | `send` trả rc 0 rồi mất prompt (T11: 7/8; T13 mất B) | F13 ACCEPTED ≠ DURABLE | CM `4164d3de`; MEM `markHandedOff`; HS #5251; ý watermark của fork tt12b | ghi tin user (placeholder liên kết ID) trong transaction trước khi ack; prompt chờ bị huỷ không gửi model (D5) | `b5a487f1` + `92182451`, `cf955131` |
| P1-b | MCP trả 5 MiB đi nguyên vào request kế và DB; `isError` bị bỏ (T9, T19) | F08, F14 | CM #197, MEM #3802; HS FC3 (envelope) | dùng lại đường trần + spill của bash | `33daa5bf` |
| P1-c | `run -s` ghi model của session thành cấu hình workspace (T17: 2/5 lượt sai provider) | F20 (dẫn xuất → thẩm quyền) | MEM `sdk_sessions.project` first-write-wins | model thuộc workspace, mở/tiếp session không ghi (Q4) | `4741c543` |
| P2-a | agent con nhận `<mcp-instructions>` cho tool nó không có (T5) | F05 CAPABILITY_MISMATCH | CM #233, #724, #794, #1037; CM v2 `subagent.ts` | chỉ dẫn theo đúng tool set của lượt; thứ tự ổn định | `73ce25eb` |
| P2-b | `-c ..` đặt workspace/DB sai chỗ; `loop -c ..` ra GREEN giả (T2) | F01 | MEM #3608 (resolver chung) | `ResolveCwd` tuyệt đối một lần | `1cda7f1b` |
| P2-c | 2 DB cho một repo theo thứ tự mở (T3); workspace lồng nhau chạy 2 lượt cùng session (T1M); symlink (T1) | F01 | MEM #3608, #2882; CM v2 resolver theo git-common-dir | một workspace + một DB cho mỗi repo ở MỌI chế độ (quyết định Q5 của người dùng); một hàm `config.ProjectRoot`; giữ dấu `.crush` cũ + cảnh báo; tiến trình chuyển về gốc; `loop` nêu thư mục verify. 4 vòng soát (3 KHONG-DAT, mỗi vòng một lỗi mới do chính quyết định kéo theo) | `300c8dd1` (nhánh POC `64bd6a45`) |
| P2-d | có hook PreToolUse thì MCP đã tắt vẫn gọi được (3/3 request) | F05/F09 (quyết ở một chỗ, chỗ dùng không đọc) | CM: bản sửa chỉ lan tới 1/9 hook | `unwrapTool` cho mọi bộ lọc nhận diện theo kiểu | `dd5fe93f` + `642fe075` |
| P3-a | tóm tắt LLM tới model dưới vai user không nhãn (T12) | F16, F18 | MEM #4141; CM #1259, #1263; nhánh `bugfix/security-hardening` của MEM (khung "unverified") | nhãn nguồn trong bộ nhớ, giữ vai user | `474a51f5` |
| P3-b | sổ ghi LLM ghi lỗi mà không tín hiệu (T15) | F14, F21 | HS #5582; MEM log không retention | WARN mỗi prompt + `ledger_error` (D2) | `01296899` |
| INTEG3 | hồi quy do P0-2: dự án mở `--yolo` thì mọi `run/send/loop` bị từ chối, không cờ nào khớp | — (lỗi tích hợp) | — | `--yolo` cho run/send/loop; lỗi nói chiều | `20212e24` |
| docs | 9 câu docs/help sai sau khi gộp | "docs match reality" (spec §76) | — | | `ef2b8050` |

**Bài học tích hợp đo được trong chính đợt này** (cùng hình dạng với lỗi của ba dự án):
- Gom QUYẾT ĐỊNH về một chỗ chưa đủ; mọi chỗ tiêu thụ phải đọc nó. Đã gặp 3 lần:
  - P2-d: hook bọc tool, bộ lọc type-assert trượt.
  - INTEG1/2: ghi chú model đọc placeholder của P1-a.
  - INTEG3: backend so `--yolo`, ba lệnh không khai cờ.
- Lỗi tích hợp chỉ lộ ở **binary + kịch bản ghép**. Cả 3 lần không unit test hay cổng E2E nào bắt được. Người bắt được:
  - INTEG1: soát độc lập.
  - INTEG3: soát docs đối chiếu mã.

## 3. Agent-cli đã làm tốt hơn ở đâu (giữ nguyên, không nhân đôi)

Lấy theo FIT §1 (11 dòng, đa số E1 đọc mã). Đo lại trong V3:
- **Dòng 1:** một điểm dựng model, allowlist mỗi lượt, fail closed. Đo lại bằng cổng `qwen-only`, `faults` (rc 0) và eval
  LLM thật (12/12, 0 lọt provider).
- **Dòng 3:** tóm tắt rỗng không thay lịch sử. Thuộc unit có sẵn trong bộ toàn kho.
- **Dòng 5:** mọi việc có trạng thái cuối. E2E routine có ca SIGKILL → `ABANDONED`.
- **Dòng 9:** spawn single-flight. T20 đo được panic trên V2 và đã sửa ở P0-1, nên dòng này **chỉ đúng một phần** trên V2.

Các dòng còn lại (journal fsync, log retention, sổ ghi chỉ metadata, không observer LLM, stop hook có trần, prompt giao
việc nguyên văn, worktree liên kết đúng danh tính) giữ như FIT §1, chưa đo lại.

## 4. Phần agent-cli phải tự sở hữu (không mượn cơ chế ngoài)

| Phần | Chủ trong mã | Vì sao không mượn |
|---|---|---|
| Danh tính dự án / data dir | `projects.NormalizeWorkspace`, `config/load.go`, `fsext/lookup.go` | MEM chưa có resolver chung (#3608 OPEN); CM v2 resolver chưa phát hành. Luật phải khớp launcher của agent-cli |
| Chính sách tin cậy + provider | `internal/execpolicy`, `coordinator.buildAgentModels` | MEM fallback chéo provider mặc định (#4727) là đúng cái V2 đã gỡ |
| Độ bền prompt/run | `internal/backend/agent.go`, `settle.go` | cả ba dự án đều hỏng ở "accepted ≠ durable"; cách giải phụ thuộc DB và vòng đời run của agent-cli |
| Văn bản prompt theo capability | `sessionAgent.Run` (tool set của lượt) | CM v2 quyết ở một module nhưng chỗ viết không đọc (V2-F2) |
| Bằng chứng "xong" | `loop` (journal + verify), các cổng `scripts/eval/*` | không dự án nào có cổng xác minh tương đương; HS/MEM báo thành công theo "accepted" |

## 5. MINIMUM V3

= V2 + 14 sửa ở §2 (13 POC + INTEG3; P2-c khi đạt soát) + docs khớp mã. Không thêm bảng, store, worker, tiến trình nền hay
lời gọi LLM nào. Kiểm cơ học trên ứng viên ghi ở `FINAL_V3_AUDIT.md`.

## 6. OPTIONAL V3 (chỉ mở khi có điều kiện đo)

| Mục | Điều kiện mở lại | Cách đo |
|---|---|---|
| Phần còn lại của T6 (todo reminder cho `agentic_fetch`/plan; template nêu `find_references`, `web_fetch`, `question` khi vắng) | có `CRUSH_HYPER_API_KEY` để ghi lại cassette `TestCoderAgent` | T6 xanh + cassette mới |
| Trần + spill cho kết quả giao việc Claude Code/Codex (FIT A1, phần chưa làm ở P1-b) | một lượt giao việc trả output lớn đo được | như T9 cho `provider/delegate.go` |
| Sổ ghi cho giao việc và hook (H3) | viết T16 và đỏ | T16 |
| Trường revision cho tóm tắt (E), làm mới env/git trong system prompt (X4) | T8 đỏ | T8 |
| Digest có cấu trúc, tất định (D2, theo hướng CM v2 không LLM) | baseline Daily B (resume) hỏng ở M0 | eval s10-resume nhiều lượt |
| Tri thức lịch sử đã kiểm (F, M1 bằng quy ước tệp) | Daily B/I/J hỏng ở M0 | eval tương ứng |
| Khung nhãn mở/đóng cho tóm tắt (sau P3-a) | eval LLM thật cho thấy model vẫn làm theo câu chèn | ca chèn ở cổng eval thật |

## 7. Bác và hoãn (độ phức tạp bị cắt)

Giữ nguyên FIT §3. Không có bằng chứng mới nào trong đợt POC làm đổi quyết định.

| Đề xuất | Quyết định | Lý do (bằng chứng chính) |
|---|---|---|
| Reflect, mental models | REJECT | HS #5728 "nothing found" đè trang tốt, #5228; không số đo lợi ích |
| Auto-recall mỗi lượt | REJECT | đổi tiền tố prompt, mất cache (MEM #4061); mức dùng 7/1.360 lượt (HS #2347) |
| Observation LLM cho mỗi sự kiện | REJECT | MEM #2201 (345M token/33 giờ), #3653 (13 obs/44k sự kiện) |
| Nhiều DB / worker / injector | REJECT | CM v1→v2 quay về một DB; chính agent-cli T3 (2 DB một repo) là lỗi |
| Worker nền / hàng đợi bền cho memory | REJECT | MEM gỡ vì retry storm rồi mất batch; HS worker kẹt |
| Fallback chéo provider mặc định | REJECT | MEM #4727; V2 đã gỡ |
| Cloud memory sync, benchmark LoCoMo làm bằng chứng | REJECT | ngoài phạm vi sản phẩm; bằng chứng E0 |
| Graph, semantic, event sourcing, ExecutionScope 11 chiều, context compiler đầy đủ, knowledge store | DEFER | không lỗi nào của agent-cli đòi; điều kiện mở lại ở FIT §0 |

## 8. Giấy phép

Không POC nào chép mã ngoài (FIT §5). CM là ELv2: chỉ học ý tưởng. Ý tưởng đã dùng (watermark, envelope, "xoá bản sao
luật, mọi caller qua một hàm") đều viết lại trong chủ cũ.

## 9. Giới hạn của quyết định này

- **Eval LLM thật**:
  - Mỗi kịch bản chỉ 1 lượt; A=v2-final, B=ứng viên `73ce25eb` (trước P2-d, P3-a, INTEG3); bỏ s13 (gọi GLM).
  - Kết quả: passed 12→12, false_done 0→0, lọt provider 0.
  - Với n=1, không kết luận được về chi phí: tokens vào 1,53M→1,58M, dao động hai chiều theo kịch bản
    (`experiments/eval/ab1-COMPARE.md`).
- **Hiệu ứng của P3-a lên model thật** chưa đo.
- **Delegation thật** (`scripts/test-agent-cli-delegation.sh`) không chạy trong V3: nó dùng Claude Code/Codex thật.
