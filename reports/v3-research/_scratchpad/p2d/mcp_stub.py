# MCP stdio tối thiểu cho e2e P2d. argv: tên server, đường crush.db hoặc "-", tệp log.
# Có đường DB: lúc initialize ghi dòng tắt server này vào mcp_disabled_servers, mô phỏng một tiến trình
# agent-cli khác trên cùng repo vừa tắt nó bằng "Toggle MCPs" trong khi tiến trình này vẫn giữ kết nối.
import json, sqlite3, sys
name, dbp, logp = sys.argv[1], sys.argv[2], sys.argv[3]
def log(s):
    with open(logp, "a") as f:
        f.write(s + "\n")
for line in sys.stdin:
    try:
        msg = json.loads(line)
    except Exception:
        continue
    mid, method = msg.get("id"), msg.get("method", "")
    if mid is None:
        continue
    if method == "initialize":
        if dbp != "-":
            c = sqlite3.connect(dbp, timeout=10)
            c.execute("INSERT OR IGNORE INTO mcp_disabled_servers(name) VALUES (?)", (name,))
            c.commit(); c.close()
            log("disabled-in-db " + name)
        res = {"protocolVersion": msg.get("params", {}).get("protocolVersion", "2025-06-18"),
               "capabilities": {"tools": {}}, "serverInfo": {"name": name, "version": "0.0.1"}}
    elif method == "tools/list":
        res = {"tools": [{"name": "send", "description": "Send a message", "inputSchema": {"type": "object", "properties": {}}}]}
    elif method == "tools/call":
        log("tools/call " + name + " " + str(msg.get("params", {}).get("name")))
        res = {"content": [{"type": "text", "text": "sent-by-" + name}]}
    elif method == "ping":
        res = {}
    else:
        sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": mid, "error": {"code": -32601, "message": "method not found"}}) + "\n")
        sys.stdout.flush()
        continue
    sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": mid, "result": res}) + "\n")
    sys.stdout.flush()
