# AGENT CLI V3 — ISSUE/FORK ARCHAEOLOGY + INTEGRATION DECISION (overlay)

Nguồn: người dùng dán trong phiên 19/10/2026, đi kèm `V3_SPEC.md`. Chép giữ nguyên các schema, danh sách và câu hỏi;
phần văn xuôi rút gọn. Không viết lại spec V3; đây là chỉ thị thi hành BỔ SUNG.

## 0. Không làm gián đoạn V2
POC/code V3 chỉ sau khi V2 có mốc ổn định (đã có: nhánh `v2-final`, cây `128d744a`). Nghiên cứu chạy chỉ đọc.

## 1. V3 KHÔNG phải "cài ba dự án"
Context Mode → bài học về WORKING CONTEXT. claude-mem → ENGINEERING CONTINUITY. Hindsight → HISTORICAL INTELLIGENCE.
Agent CLI vẫn sở hữu: Workspace, Session, Run, Task, Attempt, Provider Policy, Skills, Agents, Routine, Git/Runtime,
Verification, Evidence, Information Authority. Dự án ngoài là NGUỒN NGHIÊN CỨU, không phải thẩm quyền kiến trúc.

## 2. Câu hỏi chính
Không hỏi "copy được tính năng gì". Hỏi "lỗi thật nào buộc các dự án này tiến hoá". Mỗi tính năng quan trọng:
ORIGINAL PROBLEM → INITIAL DESIGN → REAL USER FAILURE → ISSUE → ROOT CAUSE → FIX / PR / COMMIT → REGRESSION TEST →
LATER REGRESSION? → COMMUNITY/FORK ALTERNATIVE → CURRENT UPSTREAM DESIGN → GENERAL ENGINEERING LESSON → AGENT CLI ANALOG →
V3 DECISION.

## 3. IssueRecord
```
IssueRecord { project, issue_id, title, date, affected_version, symptom, user_impact, subsystem, root_cause,
  root_cause_confidence, state, fixing_pr, fixing_commit, regression_test, later_regression, current_code_status,
  reproducibility, architectural_lesson, agent_cli_analog, proposed_agent_cli_regression_test }
```
Trạng thái: OPEN, FIXED, PARTIALLY_FIXED, REGRESSED, DESIGN_LIMITATION, ENVIRONMENT_SPECIFIC, OBSOLETE, NOT_REPRODUCED,
INSUFFICIENT_EVIDENCE.

## 4. Cụm lỗi xuyên dự án (tối thiểu)
SCOPE_FAILURE, SESSION_CONTAMINATION, PROJECT_IDENTITY_FAILURE, BRANCH_CONTAMINATION, WORKTREE_CONTAMINATION,
ROUTING_FAILURE, CAPABILITY_MISMATCH, HOST_ADAPTER_MISMATCH, CONTEXT_OVERHEAD, CONTEXT_POLLUTION,
CONTEXT_INJECTION_FAILURE, CONTEXT_DELIVERY_FAILURE, STARTUP_RACE, WORKER_FAILURE, ZOMBIE_OPERATION, BACKPRESSURE,
RETRY_STORM, PARTIAL_SUCCESS, FALSE_SUCCESS, CURSOR_ADVANCE_BEFORE_DURABILITY, STALE_MEMORY, STALE_CONTEXT,
DERIVED_STATE_STALENESS, RETRIEVAL_MISS, RETRIEVAL_NOISE, WRONG_SCOPE_RETRIEVAL, PROVENANCE_LOSS, INVALIDATION_FAILURE,
DELETION_PROPAGATION_FAILURE, PROMPT_INTERFERENCE, MEMORY_INSTRUCTION_INJECTION, SECURITY_BOUNDARY_CONFUSION,
PROVIDER_POLICY_BYPASS, OBSERVABILITY_FAILURE.
Mỗi cụm: dự án nào mắc? vì sao? giải khác nhau thế nào? giải pháp nào sống? giải pháp nào sau đó hồi quy? Agent CLI nên
rút ra bất biến gì?

## 5. Context Mode — đào issue bắt buộc
Họ issue: chi phí routing ở subagent; lệch tool/capability của subagent; subagent không nhận được chỉ dẫn routing; prompt
tiêm vào làm nhiễu agent bên thứ ba; dọn dẹp/vòng đời session; độ bền fetch/index/parser; quyền; khác biệt host; context
containment ≠ cô lập an ninh OS; adapter nhiều client. Issue quan trọng: đọc thảo luận → tìm bản sửa → xem diff → xem test
→ xem mã hiện tại → thiết kế có đổi về sau không. Ví dụ đã biết: #233 (routing block tiêm thừa vào subagent), #946 (tiêm
làm đổi prompt, xung đột classifier của host), xử lý deferred tools của Claude Code sau #724.
Bất biến tiềm năng (CHƯA chấp nhận trước khi audit agent-cli): GLOBAL TOOL AVAILABLE ≠ CURRENT AGENT CAN USE TOOL →
ContextPlan phải biết capability.

## 6. claude-mem — đào issue bắt buộc
project scope, session scope, project identity, monorepo identity, worktree identity, derived/dream namespaces, context
injection, cold-start race, runtime/store mismatch, host output contracts, worker lifecycle, log storms, sync conflicts,
cross-project search, MCP scope, subagent observations. Đã biết: #2909 (cô lập session), #3608/#2882 (project identity lệch
giữa đường ghi/đọc, monorepo/worktree; đề xuất resolver chung), #2991 (runtime đọc nhầm store), #2795 (cold-start race),
#2547 (hợp đồng tiêm theo host), #3764 (truy xuất chéo dự án khi thiếu project).
Truy: CAPTURE PATH, SYNC PATH, RETRIEVAL PATH, INJECTION PATH, MIGRATION PATH — mỗi đường tính identity thế nào; ghi lại
từng resolver; upstream có (định) đưa về MỘT resolver chuẩn không. Nguyên tắc tiềm năng cho Agent CLI: ExecutionScope tạo
MỘT lần rồi truyền qua mọi hệ con, không để từng hệ con tự suy workspace/session/branch/worktree/provider — chứng minh trên
Agent CLI trước.

## 7. Hindsight — đào issue bắt buộc
Dùng lại audit Hindsight trước (SE 24–30, `opensource/luannt-hindsight-1610.md`), không làm lại mù. Đi: revision đã audit →
upstream hiện tại → Git delta → issue delta → kiến trúc delta → coding-agent delta → worker delta → retrieval delta.
Đào: async retain, accepted operation, extraction failure, cursor advancement, incremental append, full replacement fallback,
worker recovery, replacement worker identity, structured-output parse failures, zombie processing, queue health,
consolidation, derived observations, coding-agent injection, recall relevance, mô hình lợi ích của memory lịch sử.
Đã biết: #5251 (async retain được accept rồi extraction fail, cursor vẫn tiến, mất turn), #2413 (worker suy giảm mà health
vẫn xanh), #2347 (memory dài hạn cho coding có lợi đo được không).
Bất biến phải điều tra: ACCEPTED ≠ DURABLE ≠ EXTRACTED ≠ INDEXED ≠ SEARCHABLE ≠ CURRENT ≠ VERIFIED. Agent CLI không được
gộp các trạng thái này.

## 8–10. Đào fork
Mỗi dự án: tìm fork có ý nghĩa (không xếp theo sao). Mỗi fork: UPSTREAM BASE COMMIT → FORK POINT → FORK-ONLY COMMITS →
CHANGED FILES → CHANGED ARCHITECTURE → NEW TESTS → REMOVED UPSTREAM LOGIC → NEW FAILURE MODES → CURRENT UPSTREAM EQUIVALENT.
Phân loại: MIRROR, TRACKING_FORK, HOST_PORT, SECURITY_HARDENING, RETRIEVAL_EXPERIMENT, SCOPE_FIX, WORKER_FIX,
ARCHITECTURAL_FORK, DEPLOYMENT_VARIANT, OTHER (bỏ qua mirror trừ khi có ích về lịch sử).
Câu hỏi: nỗi đau nào sinh ra fork? upstream không muốn hay không thể giải? riêng của host? thí nghiệm? upstream có nhận ý
tưởng về sau không, theo cách khác không? fork chết chưa? fork thêm bug mới không? Agent CLI có cùng ràng buộc không?
```
ForkChange { project, fork, fork_base, commit_or_series, original_upstream_behavior, changed_behavior,
  problem_addressed, architecture_change, complexity_added, tests_added, runtime_evidence, new_failure_modes,
  current_upstream_equivalent, still_relevant, agent_cli_equivalent_problem, decision }
```
decision: PORT_IDEA, ADAPT_IDEA, REIMPLEMENT_NATIVE, DEFER, IGNORE, OBSOLETE.

## 11. Không chép bản sửa nguyên văn
FAILURE → INVARIANT → GENERAL DESIGN trước khi tính cài đặt. Ví dụ: "nếu là subagent Claude thì bootstrap deferred MCP
tool" → bài học chung: capability của agent phải được giải trước khi soạn context/chỉ dẫn tool.

## 12. Đầu ra xuyên dự án
`reports/v3-research/cross-project/ISSUE_TO_INVARIANT_MAP.md`, mỗi họ lỗi: FAILURE FAMILY, REAL ISSUES, ROOT CAUSES,
UPSTREAM FIXES, FORK ALTERNATIVES, REGRESSIONS, GENERAL INVARIANT, AGENT CLI EXPOSURE, V3 TEST, V3 DESIGN CONSEQUENCE.

## 13. Ứng viên phải CHỨNG MINH (không trả lời trước audit)
- A — ARTIFACT / CONTEXT CONTAINMENT (Context Mode): RAW TOOL OUTPUT → ARTIFACT → SEARCH/FILTER → SMALL CONTEXT. V2 đã đủ
  chưa? cắt bớt có đủ? tìm chính xác? FTS? semantic?
- B — CANONICAL SCOPE RESOLVER (claude-mem): `ExecutionScope { workspace, session, run, task, attempt, branch, worktree,
  revision, agent, provider_scope, host }` giải một lần. Agent CLI có tương đương chưa? Có thì mở rộng chủ cũ, không dựng
  bộ máy scope thứ hai.
- C — CONTEXT COMPOSITION AUTHORITY: thi V2 vs section thứ tự cố định vs bộ chọn theo ngân sách vs compiler đầy đủ.
- D — STRUCTURED SESSION CONTINUITY (claude-mem): EngineeringEvent, SessionDigest; thi transcript vs summary vs digest vs
  digest + truy xuất.
- E — PROVENANCE + LIFECYCLE (Hindsight): SOURCE, REVISION, CREATED, VERIFIED, SUPERSEDED, INVALIDATED.
- F — HISTORICAL KNOWLEDGE: chỉ giữ quyết định đã kiểm, cách đã thất bại, nguyên nhân gốc, quy ước, bài học dùng lại.
- G — ENGINEERING EVENT MODEL: sự kiện nhẹ vs mô hình chuẩn vs event sourcing (event sourcing KHÔNG mặc định).
- H — PROVIDER POLICY XUYÊN ĐƯỜNG THÔNG TIN: summarizer, compactor, context selector, memory extractor, memory reflector,
  reviewer, background worker. Không provider ẩn.

## 14. Chưa được duyệt cho V3
semantic everything, graph memory, mental models, Reflect, LLM observation cho mọi event, tự recall mỗi lượt, cloud memory
sync, full event sourcing, ba database, ba hệ worker, ba bộ tiêm.

## 15–18. Luật
- Mỗi lỗi ngoài đáng học → một ứng viên test cho Agent CLI (routing → test context subagent bị hạn chế; session
  contamination → test cô lập hai session cùng workspace; project identity → test identity monorepo/worktree; cold start →
  test memory chưa sẵn lúc SessionStart; accepted-retain → test persist accept-rồi-hỏng; worker suy giảm → test
  health-xanh-mà-worker-kẹt).
- ISSUE → ROOT CAUSE → INVARIANT → AGENT CLI EXPOSURE → TEST → SIMPLEST DESIGN. Không bao giờ ISSUE → COPY PATCH.
- Bug lặp (sửa → hồi quy → sửa) là tín hiệu thiếu chủ sở hữu/bất biến; hỏi vì sao lớp bug cứ quay lại.
- Phân loại bản sửa: LOCAL_PATCH, SHARED_ABSTRACTION, ARCHITECTURAL_FIX. Năm chỗ cùng phải vá → tìm chủ chung thiếu; không
  thêm abstraction khi chưa chứng minh lặp.

## 19. Fit/gap
Cột: Capability, Agent CLI V2, Context Mode, claude-mem, Hindsight, Issue Evidence, Fork Evidence, Agent CLI Gap, Simplest
Solution, Complexity, Decision. Decision: KEEP_V2, EXTEND_V2, ADAPT_EXTERNAL_IDEA, REIMPLEMENT_NATIVE, POC_REQUIRED, DEFER,
REJECT.

## 20–23. Thi kiến trúc, đo, red team, cắt
Mọi đề xuất lớn thi với V2 (working context, session continuity, historical memory, scope, context composition). Đo tối
thiểu: verified task success, false DONE, resume correctness, rediscovery, wrong/stale assumptions, scope contamination,
context tokens, tool-output tokens, LLM calls, latency, background jobs, storage, provider violations, recovery time, user
interventions. "Kiến trúc trông sạch hơn" không phải bằng chứng. Sau mỗi POC: agent độc lập tấn công scope, staleness,
provider leakage, partial persistence, cold start, worker crash, replacement worker, malicious memory, secret retention,
context explosion, prompt composition, reviewer bias, branch/worktree contamination. Không có lợi đo được → bỏ; bản
deterministic ngang thì chọn deterministic; docs/test trong repo giải được → chọn chúng; V2 đã giải → không nhân đôi.

## 24. Báo cáo cuối bắt buộc
`reports/v3-research/V3_INTEGRATION_DECISION.md` trả lời rõ: từ Context Mode / claude-mem / Hindsight: cái gì vào V3, vì
sao, issue nào chứng minh, fork nào cho ý tưởng tốt hơn, bác cái gì; Agent CLI đã làm tốt hơn ở đâu; cái gì phải do Agent
CLI sở hữu; MINIMUM V3; OPTIONAL V3; độ phức tạp bị bác.

## 25–26. Giả thuyết Minimum-V3 (KHÔNG phải sự thật định sẵn)
Current Truth, Canonical Execution Scope, Engineering Events, Artifact Store, Working Context, Structured Session
Continuity, Historical Knowledge (provenance + lifecycle), Context Composition, Provider Governance, Verification/Evidence.
Được phép phát hiện là vẫn quá lớn; ví dụ hệ thắng có thể chỉ là: Current Truth, Scope Resolver, Artifact Store + FTS,
Structured Session Digest, Small Verified Knowledge Store, Provider Policy, Verification — không graph, không reflect, không
mental model, không semantic, không event sourcing. Đó là kết quả thành công nếu số đo ủng hộ.

## 27–28. Hành vi tự chủ + luật cuối
Không chờ xác nhận sau mỗi phát hiện. Họ lỗi mới → ghi → điều tra → thêm ứng viên test → đi tiếp. Giả định sai → ghi
ASSUMPTION_INVALIDATED. POC hồi quy → ROLLBACK + ghi bằng chứng. Chỉ chặn khi cần người thật.
Chỉ xây V3 khi: REAL FAILURE → VERIFIED ROOT CAUSE → GENERAL INVARIANT → AGENT CLI HAS SAME EXPOSURE → TEST REPRODUCES
EXPOSURE → SIMPLEST SOLUTION → POC → MEASUREMENT → RED TEAM → BETTER THAN V2 → KEEP.
Issue cần đào thêm theo người dùng: claude-mem #3608 (đường ghi/đọc/migration tự tính project identity → memory phân mảnh
hoặc biến mất; đề xuất resolver chung) và Hindsight #5251 (semantic false success: API accept ≠ đã retain).
