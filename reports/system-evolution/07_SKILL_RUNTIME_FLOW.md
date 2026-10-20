# 07 — Luồng runtime của skill (Phase 2)

Vòng đời DISCOVERED → VALID → ELIGIBLE → SELECTED → ACTIVATED → USED → EFFECTIVE, đo ở HEAD `4a0b25b6` (g5 + verifier: 30 xác nhận, 1 sửa ở bước ACTIVATED).

| Bước | Ở đâu | Ghi lại/quan sát được? | Đo trên máy |
|---|---|---|---|
| DISCOVERED | `skills.DiscoverWithStates` (đệ quy, theo symlink); Manager lúc dựng workspace **và** prompt builder mỗi lần dựng prompt (hai registry) | trạng thái trong RAM, sidebar (tên + icon) | 92 SKILL.md, 0 lỗi; 51 hoạt động ở repo này |
| VALID | `Parse` + `Validate` (tên ≤ 64, khớp thư mục; description ≤ 1.024 B) | lỗi → `StateError` + Warn; UI không nói lý do | 0 lỗi |
| ELIGIBLE | khử trùng → `disabled_skills` → bỏ `disable-model-invocation` → `ToPromptXML` | chỉ template coder render; crush_info, log `active_total` | 48 trong prompt |
| SELECTED | **không có mã**: model tự chọn theo luật bắt buộc (`coder.md.tpl` luật 14 + `<skills_usage>`) | không ghi | — |
| ACTIVATED | model `view` SKILL.md → metadata `resource_type=skill` + `Tracker.MarkLoaded` | `session show` (Skills), mục "Loaded Skill" trong chat, log `loaded_this_turn`; **verifier:** đường dẫn `skills_paths` dạng `~`/`$VAR` không được nhận; agentic_fetch không theo dõi | 2/41 lượt |
| USED | **không có** | không | KHÔNG BIẾT |
| EFFECTIVE | **không có** | không | KHÔNG BIẾT |

## Kế thừa sang agent con

Không có: `task.md.tpl`, `plan.md.tpl`, prompt agentic_fetch đều không render skill (`grep AvailSkillXML` = 0). View của agent con vẫn đọc được SKILL.md nếu cha đưa đường dẫn.

## Chi phí

- Luôn trả: danh sách XML (31.860 B / 48 skill trước sửa; ~400–1.300 B mỗi skill, lớn nhất là các skill "synced" vì description dài).
- Trả khi nạp: thân SKILL.md có số dòng (1,8–32,6 KB mỗi skill; `subagent-driven-development` 32.577 B). Lượt nạp 5 skill superpowers ≈ hàng chục KB vào context một lần.
- Tracker theo workspace, mất khi khởi động lại (PRB-21); khám phá chạy lại mỗi lượt qua việc dựng lại agent con (g5).

## Điểm khác Claude Code (nơi superpowers/ponytail được thiết kế)

Claude Code có hook SessionStart tiêm nội dung "luôn bật" (ponytail, using-superpowers), có tool `Skill`, `Task` với agent con ghi được, `TodoWrite`. agent-cli không có các cơ chế này: skill chỉ là danh sách mà model phải tự nạp bằng `view`.
