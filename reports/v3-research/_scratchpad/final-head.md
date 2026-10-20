# FINAL V3 AUDIT — agent-cli

**Trạng thái cuối (spec §77, chọn đúng một): `V3 PARTIALLY ADOPTED — VERIFIED`**

- Ngày 20/10/2026. Ứng viên: nhánh local `v3-candidate` @ `__HEAD__` (19 commit trên `v2-final` `7cba42ff`). Chưa
  merge vào `main`, chưa push.
- "Adopted": các bất biến rút từ lỗi của ba dự án ngoài, mỗi bất biến ép ở chủ sẵn có của agent-cli. Mỗi cái có test
  tái hiện đỏ trên V2 và xanh trên ứng viên, cộng một lượt soát độc lập DAT.
- "Partially": mọi lớp trí nhớ/kiến trúc mới của giả thuyết Minimum-V3 bị DEFER hoặc REJECT (§B Q22–Q28). Một phần T6
  bị chặn vì cần ghi lại cassette.
- Quyết định tích hợp chi tiết: `V3_INTEGRATION_DECISION.md`. Nhật ký: `WORKFLOW_PROGRESS.md` (sự kiện 1–9x).

Viết tắt: CM = Context Mode, MEM = claude-mem, HS = Hindsight, MAP = `cross-project/ISSUE_TO_INVARIANT_MAP.md`,
FIT = `cross-project/FIT_GAP.md`, WP = `WORKFLOW_PROGRESS.md`.

---

## A. Số đo cuối (đều từ lệnh đã chạy; tái lập bằng `scratchpad/verify-final.sh <commit> <thư-mục-mới>`)

### A.1 Kiểm cơ học trên ứng viên cuối `__HEAD__`

__R3TABLE__

### A.2 Eval LLM thật (provider `main/qwen3.8`, local)

| | v2-final `7cba42ff` | ứng viên `73ce25eb` (trước P2-d, P3-a, INTEG3, P2-c) |
|---|---|---|
| passed | 12/12 | 12/12 |
| false_done | 0 | 0 |
| lọt provider | 0 | 0 |
| tokens vào | 1.525.887 | 1.581.972 |
| giây | 343 | 376 |

- Xen kẽ A/B theo kịch bản; mỗi kịch bản 1 lượt; bỏ s13 (gọi GLM).
- Kết luận được: không hồi quy pass/false_done trên mẫu này.
- Không kết luận được: chi phí (n=1, dao động hai chiều: s3 ×2,9, s16 ×0,44).
- Ứng viên cuối CHƯA chạy lại eval LLM thật. Bốn thay đổi sau `73ce25eb` đã qua cổng LLM giả (A.1).
- Nguồn: `experiments/eval/ab1-COMPARE.md`, `ab1.log`.

### A.3 Bộ phơi nhiễm: V2 → ứng viên cuối

__R3EXPOSURE__

---

## B. Trả lời 55 câu (spec §78)

### Q1. V2 sau khi xong là gì?
- Nhánh local `v2-final` `7cba42ff`: 8 commit trên `main` `4a0b25b6`; chưa push.
- Trạng thái V2: SE "SYSTEM IMPROVED — WITH KNOWN LIMITATIONS"; Daily "DAILY WORKSPACE READY — WITH KNOWN LIMITATIONS".
- Kiến trúc dựng lại từ mã: `01_POST_V2_ARCHITECTURE.md`.
  - Mỗi dự án một server (socket theo hash của gốc git); TUI/`run`/`send`/`loop` là client.
  - Một điểm dựng model với allowlist mỗi lượt, fail closed.
  - Session trong SQLite của dự án; tóm tắt bằng LLM khi đầy context.
  - Spill output bash ra `<data>/shell-output`.
  - Loop có journal fsync; routine có flock.
- Không có kho trí nhớ dài hạn, worker nền hay observer LLM.

### Q2. Giả định nào của V2 sai?
Đều đã tái hiện trên binary/unit của V2 (WP 20, 24, 30–35, 78, 83):
- "Chờ agent dựng xong" an toàn khi đồng thời: sai. 10 `run` song song làm server panic 27/200 lượt (T20).
- Cảnh báo first-wins là đủ cho chính sách client sau: sai. Allowlist chặt hơn vẫn chạy provider ngoài danh sách (C5).
- `.env` của repo chỉ ảnh hưởng tiến trình đã nạp nó: sai. Nó tới CLI con giao việc, `!cmd`, server do Home dựng.
- `send` rc 0 nghĩa là prompt đã được ghi: sai. 7/8 prompt mất khi server bị giết ngay sau (T11); prompt xếp hàng mất khi
  SIGTERM (T13).
- Mở/tiếp session cũ không đổi gì của dự án: sai. `run -s` ghi model của session thành cấu hình workspace (T17).
- Kết quả tool nào cũng có trần: sai. MCP 5 MiB đi nguyên vào request kế và DB; `isError` bị bỏ (T9, T19).
- `-c` tương đối an toàn: sai. DB rơi ra ngoài repo; `loop -c ..` ra GREEN cho thư mục khác (T2).
- Bộ lọc tool theo kiểu thấy mọi tool: sai. Có hook PreToolUse thì MCP đã tắt vẫn gọi được.
- Văn bản prompt khớp tool set: sai. Agent con nhận chỉ dẫn cho tool nó không có (T5); một phần T6 còn mở.
- Tóm tắt là dữ liệu của assistant: sai. Nó tới model dưới vai user không nhãn (T12).
- Sổ ghi LLM là bằng chứng: sai khi ghi lỗi, vì lỗi bị nuốt (T15).
- `agent-cli-tmux rm` chỉ xoá worktree rỗi: sai khi session bận ở thư mục con (T4).
- Data dir/workspace có một danh tính cho mỗi repo: sai. Có 2 DB theo thứ tự mở (T3), DB ngoài repo qua symlink (T1),
  hai workspace chung một DB chạy hai lượt cùng session (T1M).

__Q3Q13__

### Q14. Trí nhớ dài hạn có cải thiện việc code của agent-cli không?
Chưa có bằng chứng nào cho thấy có, và V3 không làm nó.
- Trên 12 kịch bản eval thật, gồm `s10-resume` và `s14-wrong-context`, M0 (không trí nhớ dài hạn) đạt 12/12 ở cả V2 lẫn
  ứng viên.
- Không lỗi nào trong 22 họ MAP của agent-cli truy về việc thiếu trí nhớ dài hạn (FIT F, SE28).
- Bằng chứng ngoài đều E0 hoặc âm: HS #2347 có 7/1.360 lượt có tìm; MEM không có eval lợi ích trên `main`.

### Q15. So với baseline nào?
- `v2-final` `7cba42ff`, chạy cùng ngày, cùng provider `main/qwen3.8`, xen kẽ A/B theo kịch bản.
- Cùng bộ cổng LLM giả và cùng bộ phơi nhiễm cho cả hai binary.

### Q16. Current truth là gì?
Tệp trong repo, git, và cấu hình của dự án. Agent đọc qua tool ở mỗi lượt. Không có bản sao dẫn xuất nào được coi là
thẩm quyền:
- model của session không còn ghi thành cấu hình (P1-c);
- tóm tắt mang nhãn là dữ liệu do assistant viết (P3-a).

### Q17. Working context là gì?
Prompt của một lượt, dựng trong `sessionAgent.Run` / `preparePrompt`:
- system prompt (template + context file ở gốc dự án + `<mcp-instructions>` chỉ của server có tool trong lượt, P2-a);
- tin của session từ điểm tóm tắt (tóm tắt có nhãn nguồn);
- tool set của lượt.

### Q18. Session continuity là gì?
Bảng session/messages trong `<gốc>/.crush/crush.db`. Prompt được ghi bền trước khi ack (P1-a). Tool call bỏ dở được ghi
"Interrupted" khi mở lại. Tóm tắt thay lịch sử khi đầy context; tóm tắt rỗng hoặc lỗi không thay.

### Q19. Historical knowledge là gì?
Không có lớp riêng. Hiện chỉ có session cũ, context file (`AGENTS.md`…), git history, và journal của loop. Store tri thức
đã kiểm bị DEFER (FIT F).

### Q20. Ai sở hữu từng phần?
| Phần | Chủ trong mã |
|---|---|
| Current truth | repo/git; cấu hình qua `internal/config` (một `ProjectRoot`) |
| Working context | `internal/agent` (`sessionAgent.Run`, `prompt.Build`) |
| Session continuity | `internal/message`, `internal/session`, `internal/backend` (ghi bền, settle) |
| Chính sách tin cậy / provider | `internal/execpolicy`, `coordinator.buildAgentModels` |
| Bằng chứng "xong" | `internal/loop` (journal + verify), `scripts/eval/*` |

### Q21. Mô hình scope cuối là gì?
- Gốc dự án là gốc git vật lý (`config.ProjectRoot`), ngoài git là thư mục vật lý.
- Mỗi gốc dự án ứng với một server, một workspace, một data dir `<gốc>/.crush`. Áp cho mọi chế độ, kể cả khi mở từ thư mục
  con (quyết định Q5 của người dùng, P2-c).
- Session nằm trong DB của dự án.
- Worktree liên kết là gốc riêng.
- Branch không phải chiều của scope; revision cho tóm tắt DEFER vì chưa viết T8.
- Không dùng `ExecutionScope` 11 chiều (FIT B2 DEFER).

### Q22. Engineering Event Model có đáng không?
Không, DEFER. Không lỗi nào truy về việc thiếu event log. Lỗi quan sát được (T15) sửa bằng tín hiệu ở chủ sẵn có (P3-b).

### Q23. Artifact Store có đáng không?
Không tạo store mới. Đường spill sẵn có (`<data>/shell-output`, trần + marker có đường dẫn) được mở cho MCP (P1-b).

### Q24. Context Compiler có đáng không?
Không. Chỉ cần một điểm quyết: văn bản prompt đọc tool set của lượt (P2-a). Bộ chọn theo ngân sách DEFER (FIT C2).

### Q25. Mức truy hồi nào thắng?
Không cần truy hồi mới: spill + `grep`/`view` sẵn có. FTS và semantic DEFER (FIT A2, A3).

### Q26. Observation có đáng không?
Không, REJECT: MEM tốn 1 lời gọi LLM mỗi sự kiện (#2201, #3653); HS #5582.

### Q27. Graph có đáng không?
Không, DEFER. Không lỗi nào đòi; bằng chứng hỏng riêng của graph chỉ có HS #5417.

### Q28. Reflect có đáng không?
Không, REJECT: HS #5728 ("nothing found" đè trang tốt), #5228, H-26.

### Q29. Phát hiện trí nhớ cũ thế nào?
Không có kho trí nhớ. Dẫn xuất duy nhất là bản tóm tắt; nó mang nhãn nguồn và không thay lịch sử khi rỗng/lỗi. Trường
revision cho tóm tắt DEFER (chờ T8).

### Q30. Sửa trí nhớ sai thế nào?
Tóm tắt được tạo lại ở lần nén sau; người dùng mở session mới. Không có tri thức bền nào cần đính chính.

### Q31. Vô hiệu hoá item dẫn xuất thế nào?
Dẫn xuất không bao giờ được nâng thành thẩm quyền:
- model suy từ session không ghi vào cấu hình (P1-c, T17);
- tóm tắt không thành lời người dùng (P3-a, T12);
- model của lượt cuối trong ghi chú bỏ qua placeholder (`message.LastReply`, INTEG2).

### Q32. Ngăn secret lưu bền thế nào?
**Chưa ngăn.** T21 (MEASURED, ứng viên): canary trong output tool nằm ở 3 tệp khi chạy qua server (`crush.db*`,
`shell-output`, journal) và 4 tệp khi chạy local `-d` (thêm log).
- Đã có: quyền 0600/0700 cho spill; sổ ghi LLM chỉ lưu metadata.
- Là giới hạn đã biết; không POC nào nhắm tới.

### Q33. Trí nhớ có ảnh hưởng chính sách được không?
Không qua đường nào đã đo:
- chính sách lấy từ env gốc chụp đầu `main` cộng tệp env (`internal/execpolicy`); `.env` của repo không đổi được nó (P0-2b);
- client có chính sách khác bị từ chối (P0-2);
- session không ghi model (P1-c);
- tóm tắt có nhãn (P3-a).

### Q34. Cô lập provider qua "trí tuệ nền" ép thế nào?
Không có worker LLM nền. Mọi lời gọi LLM nội bộ (tiêu đề, tóm tắt, agent con, `agentic_fetch`) đi qua một điểm dựng model
có allowlist. Cổng `qwen-only` (LLM giả đếm theo provider và vai trò): 5 vai trò đều đi qwen, glm = 0 (A.1).

### Q35. Scope chỉ Qwen local/miễn phí có sinh lời gọi trả phí trái phép nào không?
Không, trên các đường đã đo:
- cổng `qwen-only` và `faults`: 0 request sang provider khác (A.1);
- eval LLM thật: 0 lọt provider;
- giao việc Claude Code/Codex mặc định TẮT.

Chưa đo: giao việc khi bật (`AGENT_CLI_DELEGATION=on`) với CLI thật.

### Q36. Cold start ra sao?
- Spawn single-flight, phân loại socket cũ.
- Đua "agent dựng xong" khi nhiều `run` đồng thời đã sửa (P0-1, T20: 0/10 lỗi).
- Server khác bản mà đang bận thì lệnh chạy model bị từ chối, không chạy nhầm bản cũ.

### Q37. Worker crash ra sao?
Không có worker nền.
- Server bị giết thì tool call dở được ghi "Interrupted" khi mở lại.
- Prompt đã ack còn trong DB (T11 8/8) và hiện là đã huỷ, không tự chạy lại (D1).
- Routine bị SIGKILL ra `ABANDONED` và nhả khoá (E2E routine).

### Q38. Worker thay thế đổi danh tính ra sao?
Không áp dụng. flock nhả khi tiến trình chết, tương đương lease (FIT X5 KEEP_V2).

### Q39. Thành công một phần (async) ra sao?
- `send` ghi bền rồi mới ack (P1-a).
- Lỗi ghi sổ LLM thành `ledger_error` cộng WARN (P3-b).
- MCP `isError` được giữ (P1-b).
- Tóm tắt lỗi không thành điểm cắt.

### Q40. Backlog ra sao?
Prompt xếp hàng là hàng trong DB. Prompt bị huỷ khi còn chờ không gửi model nhưng để lại dấu trong UI (D5). Không có retry
vô hạn, không có hàng đợi nền.

### Q41. Trí nhớ không có thì sao?
Không áp dụng (không có kho). Phần phụ hỏng (sổ ghi LLM) không làm hỏng lượt chính (D2).

### Q42. Xuyên workspace?
- Mỗi dự án một server và workspace; client có chính sách hay `--yolo` khác bị từ chối kèm lời nói cách khớp (INTEG3).
- Cổng `workspaces` (Daily E: A→B→A→B) rc 0: context file và lịch sử không lọt.

### Q43. Xuyên session?
Cổng `workspaces` (Daily F) rc 0: lịch sử session khác không lọt; `-s` nối đúng. Xoá session đang bận không để tin mồ côi
(T23).

### Q44. Xuyên branch/worktree?
- Worktree liên kết cho đúng danh tính (T1 hàng `wt`).
- `agent-cli-tmux rm` không xoá worktree có session bận (T4).
- Branch không thuộc scope; độ tươi của branch trong system prompt DEFER (T8).

### Q45. Context của người soát độc lập tới đâu?
- Mỗi lượt soát là một agent riêng (loại Plan, không có Edit), context mới, chỉ nhận diff, báo cáo người viết và quyết
  định đã chốt.
- WP có 16 sự kiện REGRESSION, đa số là lượt soát KHONG-DAT bắt được lỗi.
- Giới hạn: cùng họ model với người viết. Người soát P2-c tự nhận đã sót một lỗi CHẶN ở vòng 1. Lỗi tích hợp INTEG3 do lượt
  soát docs bắt được, không phải lượt soát mã.

### Q46. V3 làm tăng hay giảm số agent?
Sản phẩm: không thêm agent, worker hay tiến trình nền nào. Quy trình làm V3: tối đa 3 agent cùng lúc (trần RAM của máy).

### Q47. Tăng hay giảm token?
- Chưa kết luận được: 1,53M → 1,58M với n=1, dao động hai chiều.
- Thay đổi có hướng biết trước:
  - agent con không còn nhận `<mcp-instructions>` thừa (P2-a);
  - session đã nén thêm câu nhãn khoảng 45 token mỗi lượt (P3-a; ước lượng đếm chữ, chưa đo bằng tokenizer).

### Q48. Có giảm khám phá lại không?
Chưa đo.

### Q49. Có giảm DONE giả không?
- Eval thật: false_done 0 → 0.
- Đã sửa hai đường DONE/STALLED giả có tái hiện:
  - `loop -c ..` ra GREEN cho thư mục khác (P2-b);
  - `loop` từ thư mục con STALLED oan (P2-c).

### Q50. Ý tưởng nào bị bác?
Reflect, mental models, auto-recall mỗi lượt, observation LLM mỗi sự kiện, nhiều DB/worker/injector, worker hay hàng đợi bền
cho trí nhớ, fallback chéo provider mặc định, cloud sync, dùng benchmark LoCoMo làm bằng chứng (FIT §3).

### Q51. Độ phức tạp nào bị cắt?
- Toàn bộ lớp trí nhớ của giả thuyết Minimum-V3 trừ phần bất biến.
- Trong POC:
  - P2-c xoá cơ chế Home gộp nhiều DB (`Locations`/`DataDirs`), thay bằng một DB;
  - INTEG2 gom ba chỗ tự tính "trả lời thật cuối" về `message.LastReply`;
  - P2-d gom việc bóc lớp hook về `unwrapTool`.
- Không POC nào thêm bảng, worker hay store.

### Q52. Cái gì còn thử nghiệm?
- Nhãn nguồn của tóm tắt (P3-a): hiệu ứng lên model thật chưa đo.
- Một workspace cho mỗi repo khi mở từ thư mục con (P2-c): mới chạy với LLM giả và unit; chưa có ngày dùng thật.

### Q53. Cái gì còn chưa biết?
- Hiệu ứng P3-a lên LLM thật.
- Chi phí token V3 so với V2 (cần n > 1).
- Giao việc thật khi bật.
- Phần T6 cần cassette mới (`CRUSH_HYPER_API_KEY`).
- `task lint` chưa chạy.
- Filetracker của server ghép đường theo cwd của tiến trình server (có từ trước, P2-c báo 11.4).
- Phần env của C5: biến thường như `$VAR` trong `base_url` theo client đã dựng server (có tài liệu).
- T21b: `.env` của cwd vào tiến trình (thiết kế, D4).
- C9: tool ra ngoài thư mục làm việc; không làm POC.

### Q54. README/docs có đúng không?
Soát độc lập đối chiếu mã, 3 vòng, DAT ở `ef2b8050`. P2-c tự cập nhật CHANGELOG và docs theo quyết định Q5. Lượt soát
docs cuối sau P2-c: __DOCS_FINAL__.

### Q55. V3 có tốt hơn V2 một cách đo được không?
- **Có, ở phạm vi đã đo:**
  - mọi ca phơi nhiễm mà POC nhắm tới từ EXPOSED thành NOT_EXPOSED (A.3);
  - bộ test toàn kho không mất test ngoài các xoá có chủ đích;
  - race 0;
  - 5 E2E và 5 cổng xanh;
  - eval thật không hồi quy (12/12).
- **Chưa đo được tốt hơn** ở chi phí, khám phá lại và hiệu ứng trên model thật.

---

## C. Vì sao chọn trạng thái này, và việc cần người dùng

- Không chọn ADOPTED vì:
  - các lớp kiến trúc của giả thuyết V3 không được làm, đúng theo số đo;
  - ứng viên chưa merge;
  - một phần T6 bị chặn.
- Không chọn DEFERRED/REJECTED vì có cải thiện đo được và đã kiểm.
- Không chọn REGRESSED vì không có hồi quy còn mở; hồi quy INTEG3 đã sửa.
- Cần người dùng:
  1. Cho phép merge `v3-candidate` vào `main`, hoặc không.
  2. `CRUSH_HYPER_API_KEY` để ghi lại cassette, nếu muốn đóng phần còn lại của T6.
  3. Bật đầu pha DEW (`spec/DEW_RECONSTRUCTION_WORKFLOW.md`).
