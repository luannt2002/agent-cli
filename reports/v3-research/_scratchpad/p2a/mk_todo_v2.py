# Dựng cây todo v2 = cây P2a v2 + phần todo reminder (giống bản v1 đã đo).
import sys

S = sys.argv[1]
T = S + "/tree-todo-v2"

p = T + "/internal/agent/agent.go"
s = open(p).read()
for o, n in [
    ("\thistory, files := a.preparePrompt(msgs, largeModel.CatwalkCfg.SupportsImages, call.Attachments...)",
     "\t// Todo reminder chỉ cho lượt có tool todos (agentic_fetch, plan không có).\n"
     "\ttodoReminder := slices.ContainsFunc(agentTools, func(t fantasy.AgentTool) bool { return t.Info().Name == tools.TodosToolName })\n"
     "\thistory, files := a.preparePrompt(msgs, largeModel.CatwalkCfg.SupportsImages, todoReminder, call.Attachments...)"),
    ("\taiMsgs, _ := a.preparePrompt(msgs, largeModel.CatwalkCfg.SupportsImages)",
     "\t// Request tóm tắt không có tool nào, nên không nhắc tool todos.\n"
     "\taiMsgs, _ := a.preparePrompt(msgs, largeModel.CatwalkCfg.SupportsImages, false)"),
    ("func (a *sessionAgent) preparePrompt(msgs []message.Message, supportsImages bool, attachments ...message.Attachment) ([]fantasy.Message, []fantasy.FilePart) {\n\tvar history []fantasy.Message\n\tif !a.isSubAgent {",
     "func (a *sessionAgent) preparePrompt(msgs []message.Message, supportsImages, todoReminder bool, attachments ...message.Attachment) ([]fantasy.Message, []fantasy.FilePart) {\n\tvar history []fantasy.Message\n\tif todoReminder && !a.isSubAgent {"),
]:
    assert s.count(o) == 1, o[:60]
    s = s.replace(o, n)
open(p, "w").write(s)

# Test todo: đúng hàm của bản v1; promptRecorder nay đã có trong P2a.
v1 = open(S + "/tree-1plus2/internal/agent/tool_prompt_test.go").read()
i = v1.index("// Todo reminder bảo model dùng tool `todos`")
p = T + "/internal/agent/tool_prompt_test.go"
s = open(p).read().rstrip("\n") + "\n\n" + v1[i:]
assert s.count("type promptRecorder") == 1
open(p, "w").write(s)

p = T + "/CHANGELOG.md"
s = open(p).read()
anchor = "`TestMCPInstructionsFollowRunTools`, `TestRunMCPInstructionsFollowRunTools`, `TestMCPInstructionsOrderIsStable`.\n"
assert s.count(anchor) == 1
s = s.replace(anchor, anchor + """- **Lời nhắc todo chỉ gửi cho lượt có tool `todos`.** Trước đây mọi agent không phải agent con nhận lời nhắc "dùng
  tool `todos`", kể cả `agentic_fetch` và plan vốn không có tool đó; request tóm tắt (không có tool nào) cũng nhận. Nay
  lời nhắc đi theo bộ tool của chính lượt đó. Agent chính (có `todos`) không đổi. Test:
  `TestTodoReminderOnlyWithTodosTool`.
""")
open(p, "w").write(s)
print("ok")
