KET-LUAN: KHONG-DAT (soát P2-c vòng 2, diff sha 6a11268e…41f8). CHẶN vòng 1 đã đóng; local ≡ server ở workspace/DB/ngữ cảnh.

CHẶN MỚI — @-mention trong TUI ra đường sai gốc (local và server) [đọc mã, chưa chạy]:
- completions.go:408 `loadFiles` liệt kê `ListDirectory(".")` = cwd tiến trình; `cd sub && agent-cli` (không -c) thì ResolveCwd không chdir → `sub`.
- ui.go:5162-5187 đính kèm `FilePath: path` tương đối theo `sub`; message/content.go:549 model nhận `<file path='foo.go'>`.
- Agent working dir = gốc (InitProject); tool ghép `SmartJoin(workingDir, …)` (tools/view.go:112).
- Hậu quả: nhắc `sub/package.json` → model đọc/sửa `<gốc>/package.json`. Trước P2c không có (cwd tiến trình = workspace). Home không bị (gọi --cwd <gốc>, có chdir).
- Sửa rẻ: sau khi dựng workspace `os.Chdir(ws.WorkingDir())` (mẫu chdir của ResolveCwd), hoặc completion liệt kê theo WorkingDir(). Test: setupLocalWorkspace từ `sub` → `os.Getwd()==gốc`.

NÊN SỬA:
- (a) lệnh trong cảnh báo (`-D sub/.crush`) nay mở session cũ với working dir là GỐC; đường tương đối trong lịch sử tính theo `sub` → tiếp tục có thể sửa nhầm tệp. Câu cảnh báo + CHANGELOG nói rõ (chỉ nên đọc / cwd đã đổi).
- (b) lệnh phụ lệch: session.go:116, stats.go:163,223, models.go:36 gọi `Init(cwd)`; logs.go:56 `Load(cwd)` → nạp `sub/crush.json` trong khi TUI/run thì không. Đổi 5 lời gọi sang `InitProject` (logs: `Load(ProjectRoot(cwd), SmartJoin(cwd,-D))`).
- (c) `worktreeRoot` lưu "" vào cache vĩnh viễn khi git lỗi/quá hạn (load.go:1391-1398): server sống lâu gặp git >5s với repo/sub → ProjectRoot(sub)=sub tới khi server tắt (T3 quay lại). Load rồi Store không nguyên tử; backend gọi ProjectRoot 2 lần (resolveWorkspaceKey rồi InitProject). Sửa: không cache khi ctx.Err()!=nil; LoadOrStore; truyền key vào InitProject. TestProjectRootGitTimeout chỉ đo thời gian, không kiểm lần sau hỏi git lại.
- (d) `recreateArgs` gửi `DataDir: ws.DataDir` khi khôi phục (client_workspace.go:1083-1087) → workspace dựng lại mang dataDirFromFlag=true dù không ai dùng -D → client sau nhận cảnh báo sai. Chỉ gửi DataDir khi ws.DataDirFromFlag.
GHI CHÚ (CHANGELOG): quá hạn 5s (nếu còn) ; mở session cũ bằng lệnh cảnh báo thì cwd là gốc; tên test `TestSessionOpensWhereItLives` mô tả cơ chế cũ.
DataDirFromFlag tương thích (omitempty, không DisallowUnknownFields; server tự tính từ args.DataDir). Răng test vòng 2 đạt; 2 test đổi không phải sửa-cho-xanh.
