# Lấy code và đồng bộ DEW

**DEW — Daily Engineering Workspace** là tên mới của dự án trước đây gọi là agent-cli.

- **Repo:** `luannt2002/dew` (trước là `luannt2002/agent-cli`; GitHub tự chuyển hướng URL cũ sau khi đổi tên).
- **Lệnh, cấu hình và biến môi trường vẫn mang tên cũ** cho tới khi đổi tên sản phẩm xong: lệnh `agent-cli`, cấu hình
  `~/.config/agent-cli/`, biến `AGENT_CLI_*`. Phần nào đổi lúc nào: [kế hoạch đổi
  tên](../reports/documentation/RENAME_PLAN.md).

## 1. Lấy code lần đầu

```bash
git clone https://github.com/luannt2002/dew.git && cd dew     # chỉ đọc; repo công khai
# chủ repo push bằng SSH: git clone git@github.com-luannt2002:luannt2002/dew.git
```

Cài và chạy lần đầu: [README, mục Cài](../README.md#cài).

## 2. Đồng bộ về sau

Chỉ có một nhánh dùng chung là `main`, lịch sử thẳng (không merge commit). Cập nhật:

```bash
git pull --ff-only
bash scripts/install-agent-cli.sh   # build lại và cài; in sẵn lệnh quay về bản cũ
```

- Sau khi cài, thoát mọi client của các dự án đang mở (`agent-cli ps`) để server nền dựng lại bằng bản mới.
- `git pull --ff-only` từ chối nếu máy bạn có commit riêng trên `main`. Giữ commit đó ở nhánh riêng rồi đưa `main` về
  bản chung (cây làm việc phải sạch trước, kiểm bằng `git status`):
  ```bash
  git branch viec-cua-toi            # giữ commit riêng
  git fetch origin && git reset --hard origin/main
  ```

## 3. Máy đang có bản clone tên cũ (`agent-cli`)

```bash
cd <thư mục clone cũ>
git remote set-url origin https://github.com/luannt2002/dew.git   # hoặc URL SSH ở trên
git fetch origin && git pull --ff-only
# tuỳ chọn: đổi tên thư mục cho khớp (thư mục đang mở trong agent-cli thì Home còn mục cũ; mở lại bằng đường mới)
mv ../agent-cli ../dew
git worktree repair   # chỉ cần khi repo có worktree phụ
```

## 4. Repo có gì để làm tiếp

- Mã nguồn (Go 1.27, fork của Crush); `CLAUDE.md` ở gốc là luật làm việc và lệnh kiểm của dự án.
- `reports/`: báo cáo V2/V3, bằng chứng, công cụ kiểm.
  - Đọc trước `reports/v3-research/HANDOFF_2026-10-20.md`: trạng thái, việc còn lại, quyết định đã chốt.
  - Kiểm toàn bộ một commit: `reports/v3-research/tools/verify-final.sh` (cách dùng ở đầu tệp).
  - Mã nguồn ba dự án đã nghiên cứu không nằm trong repo: tải lại bằng `reports/v3-research/tools/fetch-research-sources.sh`.

## 5. Đóng góp thay đổi

- Làm trên nhánh riêng; mỗi commit một thay đổi, hoàn tác riêng được.
- Trước khi gửi:
  - chạy cổng ở `CLAUDE.md` mục P5;
  - so test theo TÊN với `main`, không theo số đếm (`reports/v3-research/tools/cmp.py`).
- Mã an ninh, đồng thời hay vòng đời run cần một lượt soát độc lập (người soát không sửa mã).
- Chỉ chủ repo đưa thay đổi vào `main`.
