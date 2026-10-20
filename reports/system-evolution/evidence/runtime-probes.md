# Probe runtime (sổ bằng chứng chạy thật, cập nhật dần)

Hạng E3: binary build từ HEAD `4a0b25b6`, provider là `scripts/eval/fakellm.py`. Server giả ghi mọi request thật mà agent-cli gửi. Các cơ chế đo ở đây (chọn provider, đếm token, ngưỡng tóm tắt) không phụ thuộc chất lượng LLM.

Môi trường cô lập: `env -i`, HOME/XDG/`CRUSH_GLOBAL_*` trỏ vào thư mục tạm, `CRUSH_CLIENT_SERVER=0` (local), dự án tạm là `git init` với 1 README. Script: `scratchpad/evo/probe-provider-roles.sh <dir> <prompt>` (biến `RULES`, `CW`, `PROBE_ENV`). Cấu hình: hai provider giả trên cùng server, `large = qwen/qwen3-coder`, `small = glm/glm-5.3-flash`.

## P-01 · Vai trò → provider/model (2026-10-16)

| Vai trò | Provider nhận request | Nguồn mã | Probe |
|---|---|---|---|
| main (coder) | qwen (large) | `coordinator.go:809` | roles-1, roles-4 |
| title của session chính | **glm (small)**, gửi kèm nguyên văn prompt đầu tiên | `agent.go:2235` | roles-1 |
| title của **mỗi** session sub-agent | **glm (small)** | — | roles-4 (seq 3) |
| sub-agent (`agent` tool) | qwen (large) | `coordinator.go:809` | roles-4 (seq 4) |
| summary (auto-compact) | qwen (large) | `agent.go:1493`, `:1735` | roles-5 |
| agentic_fetch | small (đọc mã, chưa chạy) | `agentic_fetch_tool.go:150` | — |

Lệnh: `bash probe-provider-roles.sh evo/probe-roles-1 "hello probe"` → `requests.jsonl` có 2 dòng: `1 glm glm-5.3-flash title`, `2 qwen qwen3-coder main`. `RULES=rules-roles.json … "SUBAGENT please"` → 5 dòng: title(glm), main(qwen), title(glm, cho task session), subagent(qwen), main(qwen).

**Kết luận:** phiên chọn "Qwen" vẫn gửi prompt đầu của người dùng sang provider của model small. Mỗi lần gọi sub-agent lại gửi thêm một lần nữa. Không có cảnh báo.

## P-02 · Allowlist chặn ra sao

- `AGENT_CLI_ALLOWED_MODELS=qwen:qwen3-coder` → rc=1, `MODEL_NOT_ALLOWED: glm:glm-5.3-flash is not listed …`, **0 request** gửi đi. Đây là fail-closed, nhưng chặn cả lượt chứ không chỉ vai trò small.
- Cú pháp sai `qwen/qwen3-coder` → rc=1, báo rõ `is not provider:model`.
- JSON của `run` không có trường provider/model: `{"session_id","text","prompt_tokens","completion_tokens","cost"}`.

## P-03 · Thành phần context của một phiên mới (dự án rỗng, HOME cô lập)

| Phần | Byte | Ghi chú |
|---|---|---|
| Request main đầu tiên | 48.119 | |
| System prompt (`coder.md.tpl`) | 22.040 | 19 khối; `<available_skills>` 1.197 B (3 skill builtin: crush-config, crush-hooks, jq), `<skills_usage>` 1.557 B, `<env>` 337 B kèm snapshot `git status` |
| Schema tool | 25.785 | 26 tool: agent, agentic_fetch, bash, crush_info, crush_logs, download, edit, fetch, glob, grep, job_kill, job_output, ls, 7× lsp_*, multiedit, sourcegraph, todos, view, write |
| System prompt của sub-agent (`task.md.tpl`) | 1.031 | **không có** `<available_skills>` → sub-agent không thấy skill nào; 8 tool chỉ-đọc, 5.426 B |
| Title | 492 | |
| Summary | 1.752 | |

## P-04 · Vòng tự tóm tắt vô hạn (LỖI, mức HIGH)

`CW=14000 RULES=rules-summary.json bash probe-provider-roles.sh evo/probe-roles-5 "LONGTASK go"`:
- 1.511 request trong 89,9 s: **755 main + 755 summary** (+1 title). Chỉ dừng khi `timeout 90` giết tiến trình; text trả về là hàng trăm bản summary nối nhau, lỗi cuối `Context canceled`.
- Chuỗi: main (48.119 B) → summary → main "The previous session was interrupted because it got too long…" (48.278 B) → summary → … lặp.
- Cơ chế (`agent.go:1292-1308`): ngưỡng = 20% context window với window ≤ 200k; còn lại ≤ ngưỡng thì tóm tắt. Phần cố định của request (system + tool ≈ 12k token theo cách đếm của server giả) đã chiếm hơn 80% window 14k, nên sau khi tóm tắt vẫn vượt ngưỡng và tóm tắt lại. `summarizeFailed` chỉ chặn khi tóm tắt **lỗi**, không chặn khi tóm tắt xong mà không giải phóng được chỗ.
- Vùng nguy hiểm: phần cố định nằm trong [0,8·W, W). Context file lớn, tool MCP hoặc danh sách skill dài đều đẩy phần cố định lên, nên window 16k–32k của model local cũng có thể rơi vào vùng này.
- Hạng E3. Server giả đếm token = byte/4; tokenizer thật đếm khác nhưng lỗ logic (không kiểm tiến triển sau khi tóm tắt, không có trần số lần) không đổi.

## P-05 · Thành phần prompt với skill THẬT của máy (2026-10-16)

Script `scratchpad/evo/probe-context.sh <dir> <binary>`: HOME tạm, symlink chỉ-đọc `~/.agents/skills` và `~/.claude/skills` thật, dự án `git init` rỗng, provider giả. Đếm bằng regex trên system prompt đã bắt được (khối `<available_skills>` thật, bắt đầu ở đầu dòng).

| | Binary gốc (`4a0b25b6`) | Binary sau sửa "bỏ thư mục ẩn" |
|---|---|---|
| Request main | 78.972 B | 74.240 B |
| System prompt | 52.685 B | 47.993 B |
| Khối skill | 31.860 B, 48 skill (60% system prompt) | 40 skill |
| Skill lấy từ `~/.claude/skills/.trash` (đã xoá) | 8: canvas-design, algorithmic-art, internal-comms, mcp-builder, theme-factory, slack-gif-creator, web-artifacts-builder, brand-guidelines | 0 |
| Skill lấy từ `~/.claude/skills/synced` (app Claude) | 14 (docs, docx, pdf, pptx, xlsx, google-workspace, computer-use, chrome-browser, built-in-browser, deep-research, import-memory, morning, skill-creator, ui-ux-pro-max) | 14 (chờ quyết định sản phẩm) |
| Byte của trash + synced | 19.136 B (60% khối skill) | — |
| Schema tool | 25.775 B, 26 tool | như cũ |

Lớn nhất: docs 1.332 B, google-workspace 1.309, docx 1.308, computer-use 1.294. Một request "hello" ≈ 79 KB ≈ 19,7k token (byte/4) trước khi có context file nào.

## P-06 · Sổ ghi lời gọi LLM (`AGENT_CLI_LLM_LEDGER`) khớp với server giả

Binary build từ worktree sửa (`scratchpad/evo/wt`), cùng kịch bản P-01 "SUBAGENT please". Hai nguồn độc lập cho cùng một chuỗi 5 lời gọi:

| # | Server giả thấy | Sổ ghi agent-cli ghi |
|---|---|---|
| 1 | glm / glm-5.3-flash / title | `title glm glm-5.3-flash in=191 out=7 stop` |
| 2 | qwen / qwen3-coder / main | `main qwen qwen3-coder sys=22041B tools=26 in=12031 tool-calls` |
| 3 | glm / title (session agent con) | `title glm glm-5.3-flash in=194` |
| 4 | qwen / subagent | `subagent qwen qwen3-coder sys=1032B tools=8 in=1617` |
| 5 | qwen / main | `main qwen qwen3-coder in=12092 stop` |

Khớp 5/5 về thứ tự, vai trò, provider và model. Sổ không chứa nội dung prompt (unit test `TestLedgerRecordsEachCallWithRoleProviderAndUsage` kiểm chuỗi bí mật), tệp 0600. Chênh 1 byte system prompt so với P-01 là do đường dẫn thư mục probe dài hơn 1 ký tự.

## P-07 · Cô lập provider sau sửa `sameProviderSmall` (binary worktree)

| Kịch bản | Trước (P-01/P-02, binary gốc) | Sau |
|---|---|---|
| large=qwen, small=glm, "SUBAGENT please" | title ×2 đi **glm**, main ×2 + subagent đi qwen | cả 5 lời gọi đi **qwen**; glm = 0 lời gọi |
| như trên + `AGENT_CLI_ALLOWED_MODELS=qwen:qwen3-coder` | rc=1 `MODEL_NOT_ALLOWED: glm:glm-5.3-flash`, 0 lời gọi | chạy được; title + main đều qwen |

Lệnh: `AGENT_CLI_BIN=evo/bin/agent-cli-wt RULES=rules-roles.json bash probe-provider-roles.sh evo/probe-iso-1 "SUBAGENT please"` và `PROBE_ENV="AGENT_CLI_ALLOWED_MODELS=qwen:qwen3-coder" … evo/probe-iso-2 "hello probe"`. Test: `TestBuildAgentModelsKeepsSmallOnTheMainProvider` (đột biến bỏ dòng sửa → FAIL).
