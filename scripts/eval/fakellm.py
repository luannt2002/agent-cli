#!/usr/bin/env python3
"""LLM giả tương thích OpenAI, GHI LẠI mọi request để đo hệ thống dew.

Dùng cho bộ đo của đợt tiến hoá (reports/system-evolution): trả lời câu hỏi "lượt chạy này gửi gì, tới
provider nào, model nào" bằng chính request thật mà dew gửi đi, không suy ra từ cấu hình.

    python3 -I scripts/eval/fakellm.py PORT_FILE LOG_DIR [RULES_JSON]

- Provider = đoạn đường dẫn trước /v1: base_url http://127.0.0.1:PORT/qwen/v1 -> provider "qwen".
  Một tiến trình phục vụ được nhiều provider giả, nên đo được cuộc gọi chéo provider.
- LOG_DIR/requests.jsonl: mỗi request một dòng (provider, model, vai trò đoán từ system prompt, số tin,
  tên tool, byte system prompt/tools, đầu tin user cuối). System prompt đầy đủ lưu một lần ở
  LOG_DIR/sys-<sha12>.txt.
- RULES_JSON (tuỳ chọn): danh sách {"match": regex trên tin user cuối, "steps": [reply, ...]}.
  steps[n] với n = số tin tool đứng sau tin user cuối (bước cuối lặp lại). reply là một trong:
  {"text": "..."} · {"tool": "bash", "args": {...}} · {"error": 400, "message": "..."} ·
  {"raw": "..."} (luồng SSE hỏng: gửi nguyên văn rồi đóng) · thêm {"sleep": giây} được.
  Không khớp luật nào thì trả "ECHO: <đầu tin user>".
- usage.prompt_tokens ~ byte request / 4 (để kịch bản compaction chạy được), ghi rõ là ước lượng.
"""
import hashlib
import http.server
import json
import os
import re
import sys
import threading
import time

ROLE_MARKERS = [  # đầu system prompt -> vai trò; khớp internal/agent/templates/*
    ("You are Crush, a powerful AI Assistant", "main"),
    ("You are Crush in plan mode", "plan"),
    ("You are an agent for Crush", "subagent"),
    ("You will generate a short title", "title"),
    ("You are summarizing a conversation", "summary"),
    ("You are a web content analysis agent", "agentic_fetch"),
]
LOCK = threading.Lock()
SEQ = [0]


def text_of(content):
    if isinstance(content, list):
        return "".join(p.get("text", "") for p in content if isinstance(p, dict))
    return content or ""


def role_of(system, last_user):
    for marker, role in ROLE_MARKERS:
        if system.startswith(marker) or marker in system[:400]:
            return role
    if last_user.startswith("Generate a concise title"):
        return "title"
    if "Analyze this codebase and create/update" in last_user:
        return "initialize"
    return "unknown"


class Handler(http.server.BaseHTTPRequestHandler):
    log_dir = "."
    rules = []

    def log_message(self, *a):
        pass

    def do_GET(self):  # /v1/models nếu client có hỏi
        provider = self.path.split("/v1")[0].strip("/") or "default"
        self.send_json(200, {"object": "list", "data": [{"id": "m", "object": "model", "owned_by": provider}]})

    def send_json(self, code, obj):
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("content-length", 0)) or 0)
        body = json.loads(raw or b"{}")
        provider = self.path.split("/v1")[0].strip("/") or "default"
        msgs = body.get("messages", [])
        system = "\n".join(text_of(m.get("content")) for m in msgs if m.get("role") in ("system", "developer"))
        users = [i for i, m in enumerate(msgs) if m.get("role") == "user"]
        last_user = text_of(msgs[users[-1]].get("content")) if users else ""
        tool_steps = sum(1 for m in msgs[(users[-1] + 1 if users else 0):] if m.get("role") == "tool")
        tools = [t.get("function", {}).get("name", "") for t in body.get("tools", [])]
        sha = hashlib.sha256(system.encode()).hexdigest()[:12]
        role = role_of(system, last_user)
        with LOCK:
            SEQ[0] += 1
            seq = SEQ[0]
            sys_path = os.path.join(self.log_dir, "sys-%s.txt" % sha)
            if system and not os.path.exists(sys_path):
                with open(sys_path, "w") as f:
                    f.write(system)
            rec = {
                "seq": seq, "ts": round(time.time(), 3), "provider": provider, "model": body.get("model"),
                "role": role, "stream": bool(body.get("stream")), "n_messages": len(msgs),
                "roles": "".join(m.get("role", "?")[0] for m in msgs), "tool_steps": tool_steps,
                "system_sha": sha, "system_bytes": len(system.encode()),
                "tools": tools, "tools_bytes": len(json.dumps(body.get("tools", [])).encode()),
                "request_bytes": len(raw), "last_user": last_user[:200],
                "user_texts": [text_of(m.get("content"))[:60] for m in msgs if m.get("role") == "user"],
                "auth_present": bool(self.headers.get("authorization")),
            }
            with open(os.path.join(self.log_dir, "requests.jsonl"), "a") as f:
                f.write(json.dumps(rec, ensure_ascii=False) + "\n")

        reply = {"text": "ECHO: " + last_user[:60]}
        if role == "title":
            reply = {"text": "Title"}
        elif role == "summary":
            reply = {"text": "## Current State\nSUMMARY-OF-SESSION"}
        else:
            for rule in self.rules:
                if re.search(rule.get("match", "$^"), last_user):
                    steps = rule.get("steps") or [{"text": "OK"}]
                    reply = steps[min(tool_steps, len(steps) - 1)]
                    break
        if reply.get("sleep"):
            time.sleep(float(reply["sleep"]))
        if "error" in reply:
            self.send_json(int(reply["error"]), {"error": {"message": reply.get("message", "fake error"), "type": "invalid_request_error"}})
            return
        if "raw" in reply:
            self.send_response(200)
            self.send_header("content-type", "text/event-stream")
            self.end_headers()
            self.wfile.write(reply["raw"].encode())
            return
        usage = {"prompt_tokens": len(raw) // 4, "completion_tokens": 7, "total_tokens": len(raw) // 4 + 7}
        model = body.get("model") or "m"
        if not body.get("stream"):
            msg = {"role": "assistant", "content": reply.get("text", "")}
            self.send_json(200, {"id": "c%d" % seq, "object": "chat.completion", "model": model,
                                 "choices": [{"index": 0, "message": msg, "finish_reason": "stop"}], "usage": usage})
            return
        self.send_response(200)
        self.send_header("content-type", "text/event-stream")
        self.end_headers()

        def ev(**k):
            chunk = {"id": "c%d" % seq, "object": "chat.completion.chunk", "model": model, **k}
            self.wfile.write(("data: " + json.dumps(chunk) + "\n\n").encode())

        if "tool" in reply:
            call = {"index": 0, "id": "call_%d" % seq, "type": "function",
                    "function": {"name": reply["tool"], "arguments": json.dumps(reply.get("args", {}))}}
            ev(choices=[{"index": 0, "delta": {"role": "assistant", "tool_calls": [call]}, "finish_reason": None}])
            ev(choices=[{"index": 0, "delta": {}, "finish_reason": "tool_calls"}])
        else:
            ev(choices=[{"index": 0, "delta": {"role": "assistant", "content": reply.get("text", "")}, "finish_reason": None}])
            ev(choices=[{"index": 0, "delta": {}, "finish_reason": "stop"}])
        ev(choices=[], usage=usage)
        self.wfile.write(b"data: [DONE]\n\n")


def main():
    port_file, log_dir = sys.argv[1], sys.argv[2]
    os.makedirs(log_dir, exist_ok=True)
    Handler.log_dir = log_dir
    if len(sys.argv) > 3:
        with open(sys.argv[3]) as f:
            Handler.rules = json.load(f)
    srv = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    with open(port_file, "w") as f:
        f.write(str(srv.server_address[1]))
    srv.serve_forever()


if __name__ == "__main__":
    main()
