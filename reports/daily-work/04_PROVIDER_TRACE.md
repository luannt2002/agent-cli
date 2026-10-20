# 04 — Dấu vết provider (§58, Daily C, D)

Câu hỏi: mỗi lời gọi LLM đi tới provider nào, ai quyết định, và có đường nào đổi provider mà người dùng không biết?
Đọc mã ở ứng viên cuối (`r6-final.patch`), đối chiếu bằng request thật phía server giả (`scripts/eval/fakellm.py` ghi
provider theo đoạn đường dẫn trước `/v1`) và sổ ghi LLM của binary (`AGENT_CLI_LLM_LEDGER`).

## Đường quyết định

```
cấu hình (env → crushrc → crush.json các tầng → <dự án>/.crush/crush.json)
  └─ config.resolveSelectedModels        (Load và nạp lại dùng chung)
       ├─ model ID còn            → giữ nguyên
       ├─ model ID mất, provider dùng được → model mặc định của CÙNG provider (log cảnh báo, không ghi đĩa)
       └─ provider không dùng được → giữ nguyên lựa chọn, cờ Unresolved (không mượn provider khác)
  └─ coordinator.buildAgentModels        (điểm chặn DUY NHẤT; mọi agent, agent con, tiêu đề, tóm tắt, agentic_fetch)
       ├─ sameProviderSmall: model nhỏ ở provider khác → dùng model chính
       ├─ checkModelPolicy: allowlist `execpolicy` cho cả large lẫn small (sau bước thay)
       ├─ provider không có trong cấu hình / model không có → lỗi nêu provider/model (fail closed)
       └─ bọc sổ ghi LLM nếu bật
  └─ mỗi lượt ghim model lúc bắt đầu; prompt xếp hàng giữ model lúc gửi
```

## Từng đường gọi LLM

| Đường | Model dùng | Provider | Bằng chứng |
|---|---|---|---|
| Lượt chính (TUI, `run`, `loop`, `routine`) | large | provider chính | qwen-only: `main` 7 request ở qwen |
| Đặt tiêu đề | small (cùng provider) hoặc large | provider chính | qwen-only: `title` 2 ở qwen, 0 ở glm (binary gốc: 533 ở glm) |
| `agentic_fetch` | small (cùng provider) hoặc large | provider chính | qwen-only: 1 ở qwen (binary gốc: 266 ở glm) |
| Agent con (`agent`) | large | provider chính | qwen-only: `subagent` 1 ở qwen |
| Tự tóm tắt | large | provider chính | qwen-only: `summary` 1 ở qwen |
| Loop/routine | như lượt chính, agent chạy in-process | provider chính | qwen-only: 2 request đường loop, đều qwen |
| Giao việc `claude_code`/`codex` | CLI ngoài | chỉ khi `AGENT_CLI_DELEGATION=on` + allowlist + có CLI; hỏi quyền mỗi lần | `internal/agent/tools/provider/` (đọc mã); mặc định tắt |

## Đường có thể đổi provider, và trạng thái

| Đường | Trạng thái ở ứng viên | Bằng chứng |
|---|---|---|
| Model nhỏ cấu hình ở provider khác | thay bằng model chính, log một lần | faults 10/10 ca glm = 0 |
| `--small-model <provider khác>/…` chỉ định rõ | cũng bị thay (giới hạn L4, có chủ ý) | `TestSameProviderSmall` kiểm hàm thay; đường `--small-model` đi qua cùng hàm trong `buildAgentModels` (đọc mã) |
| Model đang chọn thiếu key / provider không còn lúc khởi động hoặc nạp lại | **dừng có lỗi**, không đổi provider | faults ca `selected-no-key`: rc=1, 0 request (trước vòng 6: rc=0, 2 request sang glm) |
| Model ID không còn trong provider | model mặc định của cùng provider | `TestSelectedModelGoneFallsBackWithinProvider`, `TestResolveSelectedModels_SameProviderCatalogDefault` |
| Provider lỗi lúc chạy (500, mất kết nối, treo) | lượt lỗi, không thử provider khác | faults: `http-500`, `provider-down`, `hang` đều glm = 0 |
| Người dùng đổi có chủ ý (`alt+p`, `-m`) | đổi cho cả dự án (ghi `<dự án>/.crush/crush.json`) | s13-provider-switch (LLM thật) |
| Cấu hình repo (`crush.json`) chọn provider khác | được, nếu không có allowlist; allowlist chặn bằng `MODEL_NOT_ALLOWED` | `docs/agent-cli-local-setup.md` (ranh giới provider/model); giới hạn ghi trong README |
| Kết nối không phải LLM: làm mới token / danh sách model ChatGPT, Grok | chỉ khi đã login mà chưa có danh sách model (giới hạn L8) | `internal/config/store.go` `RefetchOpenAIChatGPTModels`, `RefetchGrokModels` (đọc mã) |

## Quan sát được

- Sổ ghi LLM ghi `provider`/`model` THẬT của từng lời gọi, kể cả thử lại; khớp đếm phía server ở mọi lần chạy qwen-only (server và sổ ghi cùng `{'qwen': {'title': 2, 'main': 7, 'subagent': 1, 'agentic_fetch': 1, 'summary': 1}}`).
- `run --json` có `provider`/`model` của tin trả lời cuối đã lưu: là nhãn, không phải bằng chứng (ca `provider-down`:
  0 request vẫn có `"provider":"qwen"`).
- Báo cáo routine chưa có provider/model.
