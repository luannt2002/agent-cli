// Đo kích thước block routing (module thuần, chỉ tạo chuỗi)
const R = process.argv[2];
const rb = await import(R + "/hooks/routing-block.mjs");
const tn = await import(R + "/hooks/core/tool-naming.mjs");
const b = (s) => Buffer.byteLength(s, "utf8");
const t = tn.createToolNamer("claude-code");
const tc = tn.createToolNamer("codex");
console.log("ROUTING_BLOCK(sessionstart, claude-code) bytes", b(rb.ROUTING_BLOCK));
console.log("subagent block claude-code (includeCommands:false, bootstrap:true) bytes", b(rb.createRoutingBlock(t, {includeCommands:false, toolSearchBootstrap:true})));
console.log("subagent block codex (includeCommands:false) bytes", b(rb.createRoutingBlock(tc, {includeCommands:false})));
console.log("BASH_GUIDANCE bytes", b(rb.BASH_GUIDANCE), "READ", b(rb.READ_GUIDANCE), "GREP", b(rb.GREP_GUIDANCE), "EXT_MCP", b(rb.EXTERNAL_MCP_GUIDANCE));
