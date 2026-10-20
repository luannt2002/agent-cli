# 10 — Kết quả cô lập provider (§32, §35, Daily C, D)

Luật cứng: lời gọi LLM chỉ đi tới provider người dùng chọn; fallback chéo provider không bao giờ lặng lẽ và mặc định
đóng (fail closed). Mọi số dưới đây chạy trên binary cuối (`bin/agent-cli-final4`, build từ `r8-final.patch`) trừ khi ghi
khác. Đường đi của quyết định: [04](04_PROVIDER_TRACE.md).

## Cổng bằng LLM giả (E3, thoát 1 khi hỏng)

Server giả (`scripts/eval/fakellm.py`) phục vụ hai provider `qwen` và `glm` trên cùng cổng, đếm request theo provider và
vai trò (đoán từ system prompt). Sổ ghi LLM của binary đếm độc lập.

**`scripts/eval/faults.sh`** (`scratchpad/evo/faults-final4.txt`): model chính ở `qwen`, model nhỏ cố ý ở `glm`.

| Ca | rc | request tới glm | request tới qwen |
|---|---|---|---|
| ok | 0 | 0 | title 1, main 1 |
| provider-down (không ai nghe ở cổng của qwen) | 1 | 0 | 0 |
| http-500 | 1 | 0 | title 1, main 4 |
| http-400 | 1 | 0 | title 1, main 1 |
| empty-reply | 1 | 0 | title 1, main 1 |
| malformed-stream | 1 | 0 | title 1, main 1 |
| truncated-stream | 1 | 0 | title 1, main 4 |
| hang (provider treo 60 s) | 124 | 0 | title 1, main 1 |
| summary-loop (model làm lại việc sau mỗi lần tóm tắt) | 1 | 0 | title 1, main 33, summary 10 |
| selected-no-key (model chính chọn `openai/gpt-4o`, không có key) | 1 | 0 | 0 |
| selected-no-key-server (như trên, qua server: `CRUSH_CLIENT_SERVER=1`, socket riêng) | 1 | 0 | 0 |

Hai ca `selected-no-key*` còn chạy `agent-cli login no-such-platform`: ra "Unknown platform", tức `login` qua được bước dựng
workspace; lỗi của `run` phải nêu `"openai"`. FAULTS: PASS.

Cùng bộ ca trên binary gốc (`faults-base.txt`, chưa có ca `selected-no-key`): 8/8 ca có 1 request sang glm (lời gọi tiêu
đề). Ca `selected-no-key` trên ứng viên trước vòng 6: rc=0 sau 2 request sang glm (`"provider":"glm"` trong `--json`).

**`scripts/eval/qwen-only.sh`** (`qwen-only-final4.txt`): một lượt `run` đi qua main → agent con → `agentic_fetch` → 3 output
lớn → tự tóm tắt → trả lời; rồi một lượt `loop`.

| | Binary gốc | Binary cuối |
|---|---|---|
| Request tới glm (server) | 799 (title 533, agentic_fetch 266) | 0 |
| Request tới qwen (server) | 1.860, hết giờ rc=124 (tóm tắt lặp) | 12: title 2, main 7, subagent 1, agentic_fetch 1, summary 1 |
| Sổ ghi LLM khớp server | (sổ ghi chưa có ở binary gốc) | khớp từng ô |
| Đường loop | — | 2 request, đều qwen; dừng `NO_CHANGE`, rc=1 |

## Ngoài LLM giả

| Kiểm | Kết quả | Bằng chứng |
|---|---|---|
| Model đang chọn thiếu key, qua server (chế độ mặc định của launcher) | `run` rc=1, 0 request, lỗi tới client nêu `provider "openai", model "gpt-4o"`; `login` qua được. Binary vòng 7: lỗi chỉ còn "status code 500" | `faults.sh` ca `selected-no-key-server` (`r8-faults-old.txt`, `r8-faults-new.txt`) |
| Cùng trạng thái, TUI | màn "To start, let's choose a provider and model.", 0 request (local: tmux riêng của lead; server: người soát vòng 7) | `scratchpad/evo/r7-srv/tui.txt`, `review7/p-tui-srv` |
| `run -m`, `loop -m` sang provider dùng được khi model đang chọn thiếu key | vẫn dừng, 0 request (người soát vòng 6) | `scratchpad/evo/review6/p-loc-run-m`, `p-loc-loop-m`, `p-srv-run-m` |
| Provider dựng sẵn bị tắt mà model vẫn biết | bản trước: 8 request tới provider đã tắt (người soát vòng 6); bản cuối: dựng agent lỗi | `TestBuildAgentModelsRefusesADisabledProvider` |
| Eval LLM thật, 13 kịch bản | chỉ s13 bước 2 (đổi có chủ ý `-m glm/…`) gọi provider khác `main` | [15](15_BEFORE_AFTER_METRICS.md) |

## Trả lời

- **Ranh giới có được ép thật không?** Có, cho mọi đường gọi LLM (đều qua `buildAgentModels`), ở các ca hỏng đã tiêm.
- **Chỉ-Qwen chứng minh được 0 lời gọi trả tiền ngoài ý muốn không?** Với các đường gọi LLM mà kịch bản đi qua: có, bằng hai
  bộ đếm độc lập. Chưa phủ: kết nối không phải LLM khi đã login ChatGPT/Grok; cấu hình không đặt allowlist mà repo lạ chọn
  provider khác (đó là lựa chọn "có chủ ý" của cấu hình, TUI có hiện); giao việc `claude_code`/`codex` (mặc định tắt).
- **Còn hở:** allowlist chỉ so tên (PRB-30); key đã export làm provider đó chọn được; nhãn provider của `run --json` không
  phải bằng chứng (dùng sổ ghi LLM).
