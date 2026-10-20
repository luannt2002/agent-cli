Nếu mục tiêu của dự án là **“tôi chỉ cần vào đúng một chỗ để làm toàn bộ engineering; provider chỉ là backend thực thi”**, thì tôi nghĩ **Agent CLI không còn là tên chuẩn nhất**. Tôi vừa check và `Agent CLI` đã là một namespace khá generic, thậm chí có nhiều project công khai đang dùng đúng tên này. [GitHub](https://github.com/agent-cli/agent-cli?utm_source=chatgpt.com)

**DEW** thì đúng concept hơn, nhưng tôi cũng không muốn khóa bạn vào chữ “CLI”, vì roadmap V2/V3 đã vượt khỏi phạm vi một CLI đơn thuần.

Tôi sẽ chuẩn hóa như này:

```text
DEW
Daily Engineering Workspace

"One place for daily engineering."

                USER
                  │
                  ▼
                 DEW
      Daily Engineering Workspace
                  │
     ┌────────────┼─────────────┐
     │            │             │
 Workspace     Session        Routine
     │            │             │
     └────────────┼─────────────┘
                  │
            Engineering Runtime
                  │
     ┌────────────┼─────────────┐
     ▼            ▼             ▼
   Skills       Agents       Tools
     │            │             │
     └────────────┼─────────────┘
                  │
         Context / Continuity
             / Knowledge
                  │
          Verification/Evidence
                  │
          Provider Governance
                  │
      ┌───────────┼────────────┐
      ▼           ▼            ▼
     Qwen       Claude       GPT / GLM
```

Tên chính thức khi đó là:

> **DEW — Daily Engineering Workspace**

Chứ **không phải “DEW CLI”**.

`DEW CLI` chỉ là **một interface**:

```text
Product
└── DEW
    └── Daily Engineering Workspace

Interfaces
├── DEW CLI
├── DEW TUI
└── future desktop/web

Core
├── DEW Runtime
├── DEW Workspace
├── DEW Session
├── DEW Routine
├── DEW Context
├── DEW Knowledge
└── DEW Verification
```

Điểm này quan trọng vì project của bạn sinh ra từ đúng vấn đề:

```text
TRƯỚC

Claude Code ── context riêng
Codex       ── context riêng
Qwen        ── context riêng
GLM         ── context riêng
ChatGPT     ── context riêng

        ↓

fragmented engineering work
```

Còn thứ bạn đang xây là:

```text
SAU

                 DEW
                  │
          ONE PLACE TO WORK
                  │
    Workspace / Session / Run
                  │
     Skills / Agents / Routine
                  │
 Context / Continuity / Knowledge
                  │
      Verification / Evidence
                  │
        Provider Governance
                  │
      ┌─────┬─────┼─────┐
      ▼     ▼     ▼     ▼
    Qwen  Claude GPT   GLM
```

Đây mới chính xác là **Daily Engineering Workspace**.

Và V3 vừa thiết kế cũng khớp tên này rất đẹp:

```text
DEW V1
Agent CLI / Crush-derived foundation
        ↓
DEW V2
Daily Engineering Workspace
        ↓
Workspace + Session
Provider Governance
Skills + Agents
Routine
Verification + Evidence
        ↓
DEW V3
Engineering Information Architecture
        ↓
┌─────────────────────────────────┐
│ Working Context                 │ ← Context Mode lessons
│ Engineering Continuity          │ ← claude-mem lessons
│ Historical Knowledge            │ ← Hindsight lessons
└─────────────────────────────────┘
        ↓
One coherent engineering workspace
```

### Nhưng có một điểm tôi chưa chốt

**DEW rất đúng về meaning nhưng chưa chắc là brand public cuối cùng.**

Nếu đây là project **private/personal**, tôi chọn ngay:

> **DEW — Daily Engineering Workspace**

Nếu bạn định biến nó thành **open-source project lớn**, tôi sẽ tách:

> **Brand độc nhất** — tên riêng  
> **Category/tagline** — Daily Engineering Workspace

Bởi vì một tên viết tắt 3 chữ như `DEW` khá generic, tương tự việc `Agent CLI` hiện đã bị nhiều project sử dụng. Search hiện tại cũng cho thấy không gian coding-agent CLI đã rất đông với Codex CLI, Claude Code, OpenCode, Goose, Aider và nhiều tool khác. [AgentBoards](https://agentboards.org/category/cli?utm_source=chatgpt.com)

Khi đó cấu trúc đẹp nhất sẽ là:

```text
<UNIQUE BRAND>
Daily Engineering Workspace

The engineering workspace where
providers execute the work,
rather than own the work.
```

Tức là **“Daily Engineering Workspace” tôi nghĩ nên giữ chắc** — nó mô tả cực đúng thứ bạn đang xây. Cái đáng cân nhắc thêm chỉ là **brand ngắn thay cho DEW**, nếu sau này public.

Còn ở thời điểm V2/V3 hiện tại, tôi sẽ đổi cách gọi nội bộ từ **Agent CLI → DEW**, nhưng **chưa cần rename repo/binary ngay** cho đến khi V2 hoàn thành và rename được làm thành một migration riêng.
làm xong thì đổi tên dự án và viết lại cho tôi cái mới đi nehs?


Đúng. Sau khi **V3 hoàn tất và đã VERIFIED**, tôi nghĩ nên coi đây là một phase riêng: **Documentation Reconstruction**, chứ không phải sửa vài dòng README.

Mục tiêu là người ngoài lần đầu mở repo phải hiểu được: **DEW là gì → vì sao tồn tại → fork từ đâu → đã tiến hóa ra sao → kiến trúc hiện tại gồm những tầng nào → mỗi tầng giải quyết failure/case study nào → ý tưởng học từ project public nào → DEW giữ/sửa/bỏ gì → trade-off là gì → hiện tại cái gì đã VERIFIED và cái gì vẫn experimental.**

Và đúng như bạn nói: **không nên che nguồn gốc**. Nếu một concept được học từ Context Mode, claude-mem, Hindsight, Superpowers, Ponytail hay Crush thì docs nên nói rõ, gắn link upstream, giải thích **“we learned X; DEW adapted it into Y because…”**. Đó vừa minh bạch vừa giúp kiến trúc có lịch sử và lý do tồn tại.

Tôi sẽ thêm một phase sau V3 như sau:

```text
V3 IMPLEMENTATION
        ↓
V3 VERIFICATION
        ↓
V3 FINAL AUDIT
        ↓
STABLE COMMIT / CHECKPOINT
        ↓
PROJECT IDENTITY FINALIZATION
        ↓
HISTORY + COMMIT ARCHAEOLOGY
        ↓
ARCHITECTURE RECONSTRUCTION
        ↓
UPSTREAM / INFLUENCE ATTRIBUTION
        ↓
CASE-STUDY / FAILURE MAPPING
        ↓
README REWRITE
        ↓
DOCS REWRITE
        ↓
DOCS-AS-CODE VERIFICATION
        ↓
FRESH-READER REVIEW
        ↓
FINAL DOCUMENTATION AUDIT
```

Và naming tôi sẽ chuẩn hóa theo hướng:

```text
DEW
Daily Engineering Workspace

"One place for engineering.
Providers are how the work gets done."
```

Hoặc giữ câu North Star mạnh hơn:

```text
DEW — Daily Engineering Workspace

DEW is where engineering work lives.
Providers are how the work gets done.
```

Câu thứ hai chính xác hơn với kiến trúc của bạn: **DEW sở hữu continuity/state/workflow; provider chỉ thực thi AI workload.**

### README mới không nên là feature list

Nó nên kể câu chuyện:

```text
Why DEW Exists
      ↓
Where It Came From
      ↓
What Problem It Solves
      ↓
How You Work With It
      ↓
Architecture at a Glance
      ↓
Layer-by-Layer Explanation
      ↓
Design Influences
      ↓
What DEW Changed
      ↓
Case Studies / Failures Solved
      ↓
Trade-offs
      ↓
Verification / Evidence
      ↓
Current Status
      ↓
Roadmap
```

Ví dụ phần architecture nên cho người đọc thấy ngay:

```text
┌───────────────────────────────────────────────┐
│                    DEW                        │
│          Daily Engineering Workspace          │
├───────────────────────────────────────────────┤
│  Daily UX                                     │
│  Workspace · Session · Task · Routine         │
├───────────────────────────────────────────────┤
│  Engineering Runtime                          │
│  Run · TaskGraph · Attempt · Agent            │
├───────────────────────────────────────────────┤
│  Skills & Methodology                         │
│  Personal · Superpowers · Ponytail            │
├───────────────────────────────────────────────┤
│  Working Context                              │
│  Artifact · Search · Select · Context Budget  │
│  ← lessons from Context Mode                  │
├───────────────────────────────────────────────┤
│  Engineering Continuity                       │
│  Events · SessionDigest · Resume              │
│  ← lessons from claude-mem                    │
├───────────────────────────────────────────────┤
│  Historical Knowledge                         │
│  Provenance · Lifecycle · Recall              │
│  ← lessons from Hindsight                     │
├───────────────────────────────────────────────┤
│  Verification & Evidence                      │
│  Tests · Review · Revision-aware Evidence     │
├───────────────────────────────────────────────┤
│  Provider Governance                          │
│  Qwen · Claude · OpenAI · GLM · future        │
└───────────────────────────────────────────────┘
```

Nhưng **agent không được copy sơ đồ này rồi coi là sự thật**. Sau V3 nó phải đọc source/commit/tests/runtime và dựng lại sơ đồ từ implementation thực tế.

Phần attribution cũng không nên chỉ có một dòng “inspired by X”. Nên làm kiểu:

```text
## Design Influence: Context Mode

Upstream:
https://github.com/mksglu/context-mode

What we learned:
- keep large tool output outside active model context
- search/select before injection
- context is a limited engineering resource
- agent capability matters when composing context

Problems observed upstream:
- routing overhead
- subagent capability mismatch
- host-specific injection behavior
- context containment is not the same as security isolation
- lifecycle and routing complexity can grow quickly

What DEW adopted:
- artifact-oriented tool output
- bounded context composition
- capability-aware context planning

What DEW changed:
- context belongs to DEW's Workspace/Session/Run model
- no independent Context Mode runtime
- provider policy participates in context processing
- current verified source/evidence outranks historical context

What DEW rejected:
- ...

Why:
- ...

Trade-offs:
- ...
```

Tương tự với **claude-mem** và **Hindsight**.

Đặc biệt tôi muốn docs có một bảng **“Idea Lineage”**:

| DEW capability | Problem | Learned from | DEW adaptation | Why not copied directly |
|---|---|---|---|---|
| Working Context | Tool output làm phình context | Context Mode | Artifact → Select → Inject | DEW cần chung scope/runtime |
| Session Continuity | Mất trạng thái khi đổi session | claude-mem | Structured SessionDigest | Không muốn một memory runtime thứ hai |
| Historical Knowledge | Lặp lại lỗi/rediscovery | Hindsight | Verified knowledge + provenance | Historical memory không được làm current truth |
| Methodology | Agent làm việc thiếu discipline | Superpowers | Selective skills | Không biến skill thành orchestrator |
| Simplicity | Architecture phình | Ponytail | Simplicity advisor | Advisor, không sở hữu runtime |
| Provider isolation | Hidden model calls/cost | DEW-specific pain | Execution policy boundary | Core invariant của DEW |

Và một bảng khác rất quan trọng:

**Failure → External Evidence → DEW Invariant → Implementation → Regression Test**

Như vậy người ngoài không chỉ thấy “feature”, mà hiểu **tại sao feature tồn tại**.

Tôi nghĩ docs cuối cùng nên có cấu trúc khoảng:

```text
README.md

docs/
├── getting-started.md
├── daily-workflow.md
├── why-dew.md
├── project-history.md
├── architecture.md
├── architecture/
│   ├── current-state.md
│   ├── evolution.md
│   ├── runtime-model.md
│   ├── authority-model.md
│   ├── scope-model.md
│   ├── context-model.md
│   ├── continuity-model.md
│   ├── knowledge-model.md
│   └── provider-boundary.md
│
├── concepts/
│   ├── workspaces.md
│   ├── sessions.md
│   ├── runs-and-tasks.md
│   ├── agents.md
│   ├── skills.md
│   ├── routines.md
│   ├── context.md
│   ├── memory.md
│   ├── verification.md
│   └── evidence.md
│
├── design/
│   ├── influences.md
│   ├── context-mode.md
│   ├── claude-mem.md
│   ├── hindsight.md
│   ├── superpowers.md
│   ├── ponytail.md
│   ├── crush-origin.md
│   ├── case-studies.md
│   ├── trade-offs.md
│   └── rejected-designs.md
│
├── providers.md
├── configuration.md
├── troubleshooting.md
└── roadmap.md

reports/
├── system-evolution/
├── daily-work/
├── v3-research/
└── documentation/
```

Quan trọng nhất là `project-history.md`. Agent phải dùng **Git archaeology thực sự**:

```text
Original upstream / fork point
        ↓
Initial customization
        ↓
Provider unification
        ↓
Workspace / Session
        ↓
Multi-agent
        ↓
Provider governance
        ↓
Verification / Evidence
        ↓
Routine / long-running work
        ↓
Daily Engineering Workspace
        ↓
V3 Context / Continuity / Knowledge
        ↓
Current HEAD
```

Mỗi era phải link commit/PR nếu repo cho phép, giải thích **WHAT / WHY / WHAT CHANGED LATER**. Không được viết lịch sử tưởng tượng từ README.

Và tôi sẽ đặt một rule rất cứng:

> **Documentation must describe the verified system that exists at HEAD, not the system we hoped to build.**

Feature chưa có → `PLANNED`.  
Có code chưa test → `IMPLEMENTED — UNVERIFIED`.  
POC → `EXPERIMENTAL`.  
Đã runtime verify → `VERIFIED`.  
Đã bỏ → đưa vào history/rejected design, không mô tả như current architecture.

Cuối cùng, README phải đủ tốt để một người ngoài đọc khoảng 5–10 phút hiểu:

> **DEW là một Daily Engineering Workspace thống nhất engineering state thay vì để project bị chia cắt giữa Claude/Qwen/GPT/GLM. Context Mode cho DEW bài học về working context; claude-mem về session continuity; Hindsight về historical knowledge; Superpowers/Ponytail về methodology và simplicity. Nhưng DEW không bundle các project đó thành một hệ thống Frankenstein—nó học từ failure, issue và architecture của họ rồi reimplement những phần chứng minh được giá trị bên trong một runtime, một scope model và một authority model duy nhất.**

Đó mới là README/docs “chuẩn” cho V3. Và phần cuối docs nên có **Credits & Design Influences** rõ ràng, link từng upstream và phân biệt `forked from`, `adapted from`, `inspired by`, `compared against`; như vậy vừa tôn trọng tác giả gốc vừa không gây hiểu nhầm rằng DEW là bản đóng gói lại của họ.
Đúng ý. Sau V3 bạn cần thêm **một prompt cuối kiểu “Project Reconstruction + Rename + Documentation Constitution”**. Nó không chỉ viết README mà phải đọc **toàn bộ Git history từ fork point → HEAD**, dựng lại kiến trúc thật, giải thích từng layer/flow/capability/failure/trade-off/nguồn ý tưởng, rename project đồng bộ, rồi đặt ra **mindset để từ nay feature mới bắt buộc cập nhật docs**, tránh code tiến lên còn docs đứng lại.

Tôi sẽ dùng tên **DEW — Daily Engineering Workspace** trong prompt, nhưng bắt agent kiểm tra rename collision/impact trước khi đổi repo/binary/package. Đây là prompt riêng để chạy **sau khi V3 hoàn tất và stable**:

```text
# DEW — PROJECT RECONSTRUCTION, RENAME, ARCHITECTURE DOCUMENTATION
# AND DOCUMENTATION-CONSTITUTION WORKFLOW

PROJECT TARGET NAME:
DEW — Daily Engineering Workspace

PRODUCT NORTH STAR:

    DEW IS WHERE ENGINEERING WORK LIVES.
    PROVIDERS ARE HOW THE WORK GETS DONE.

SHORT PRODUCT IDEA:

    ONE PLACE FOR ENGINEERING.

======================================================================
0. PURPOSE
======================================================================

This workflow runs AFTER:

V2
→ V3 research
→ V3 implementation
→ V3 verification
→ V3 red-team
→ V3 pruning
→ FINAL_V3_AUDIT
→ stable checkpoint

Do NOT use this workflow to invent unfinished V3 architecture.

Its job is to reconstruct and document THE SYSTEM THAT ACTUALLY EXISTS
at the latest verified HEAD.

The final repository must make sense to a new engineer who knows
nothing about its history.

That engineer should be able to understand:

1. What is DEW?
2. Why does DEW exist?
3. What problem did the original project solve?
4. What project was DEW originally forked from?
5. How did the project evolve from that fork to current HEAD?
6. Why did each major architectural capability appear?
7. Which real failure/problem motivated it?
8. Which public projects influenced the design?
9. What exactly did DEW learn from those projects?
10. What did DEW copy, adapt, reimplement, simplify, or reject?
11. What are the major architectural layers?
12. How does data/control flow through those layers?
13. Who owns each piece of state?
14. Which component is authoritative?
15. What happens during a normal engineering request?
16. What happens across sessions?
17. What happens across providers?
18. What happens when an agent fails?
19. What happens when DEW restarts?
20. What happens when historical memory disagrees with current code?
21. How is DONE determined?
22. What evidence proves the work?
23. What are the trade-offs?
24. What complexity was intentionally rejected?
25. What is verified today?
26. What remains experimental?
27. What remains planned?
28. How should future contributors evolve the architecture?
29. Which documentation must change when code changes?
30. How do we prevent documentation drift forever?

This is NOT a cosmetic README rewrite.

This is:

    PROJECT RECONSTRUCTION
    +
    ARCHITECTURE EXPLANATION
    +
    DESIGN LINEAGE
    +
    PRODUCT RENAME
    +
    DOCUMENTATION MIGRATION
    +
    DOCUMENTATION GOVERNANCE

======================================================================
1. FIRST PRINCIPLE
======================================================================

DO NOT DOCUMENT WHAT WE WANTED TO BUILD.

DOCUMENT WHAT EXISTS.

Truth order:

CURRENT SOURCE / RUNTIME
>
CURRENT VERIFIED TEST / EVIDENCE
>
CURRENT VERIFIED ARCHITECTURE REPORT
>
GIT HISTORY
>
OLDER DOCUMENTATION
>
DESIGN INTENTION
>
ASSUMPTION

README is NOT architecture authority.

Old docs are NOT architecture authority.

V3 design documents are NOT automatically architecture authority.

The latest verified implementation is authoritative.

======================================================================
2. DOCUMENTATION STATUS LANGUAGE
======================================================================

Every major capability must be classified when necessary as:

VERIFIED
IMPLEMENTED
IMPLEMENTED — UNVERIFIED
EXPERIMENTAL
PARTIAL
PLANNED
DEPRECATED
REMOVED
HISTORICAL
UNKNOWN

Never present:

PLANNED

as:

IMPLEMENTED.

Never present:

IMPLEMENTED

as:

VERIFIED

without evidence.

======================================================================
3. RECONSTRUCT THE ENTIRE PROJECT HISTORY
======================================================================

Find the actual Git ancestry.

Determine:

ORIGINAL UPSTREAM
FORK POINT
FIRST LOCAL COMMIT
MAJOR CUSTOMIZATION COMMITS
ARCHITECTURAL TRANSITIONS
REWRITES
DELETIONS
REVERTS
CURRENT HEAD

Do not infer this from README.

Use:

git log
git show
git diff
git blame
git merge-base
git branch history
tags
release history
commit messages
changed files
tests
documentation history
configuration history

where appropriate.

======================================================================
4. COMMIT-BY-COMMIT ARCHAEOLOGY
======================================================================

Inspect every meaningful commit from fork/base to current HEAD.

For every meaningful change determine:

COMMIT
DATE
FILES
WHAT CHANGED
WHY IT APPEARS TO HAVE CHANGED
WHAT PROBLEM IT ADDRESSED
WHAT ARCHITECTURAL PRESSURE CREATED IT
WHAT CAPABILITY IT INTRODUCED
WHAT STATE IT CREATED
WHAT AUTHORITY IT CREATED
WHAT DEPENDENCIES IT INTRODUCED
WHAT TESTS IT ADDED
WHAT FAILURE MODES IT INTRODUCED
WHAT LATER COMMITS CHANGED IT
WHETHER IT STILL EXISTS
CURRENT OWNER
CURRENT STATUS

Do NOT produce one giant useless commit list.

First understand every meaningful commit.

Then group them into evolutionary eras.

======================================================================
5. RECONSTRUCT EVOLUTIONARY ERAS
======================================================================

Do not assume these eras are correct.

Verify them against Git.

Possible hypothesis:

ERA 0
Original upstream / Crush foundation

ERA 1
Personal customization

ERA 2
Provider unification

ERA 3
Workspace / Session continuity

ERA 4
Multi-agent orchestration

ERA 5
Provider governance

ERA 6
Verification / Evidence

ERA 7
Routine / long-running engineering

ERA 8
Daily Engineering Workspace alignment

ERA 9
Context / Continuity / Historical Knowledge research

ERA 10
DEW V3

Rewrite the eras if Git evidence shows a different history.

For every era explain:

BEFORE
PROBLEM
CHANGE
AFTER
NEW CAPABILITY
NEW COMPLEXITY
LESSONS
WHAT SURVIVED
WHAT WAS REMOVED
WHAT WAS LATER REWORKED

======================================================================
6. CREATE PROJECT HISTORY
======================================================================

Create:

docs/project-history.md

It must tell the real engineering evolution:

UPSTREAM
    ↓
FORK
    ↓
CUSTOMIZATION
    ↓
PROBLEM DISCOVERED
    ↓
CHANGE
    ↓
NEW FAILURE
    ↓
NEW ARCHITECTURAL PRESSURE
    ↓
NEXT EVOLUTION
    ↓
...
    ↓
CURRENT DEW

Use important commits as evidence.

Do not turn it into marketing prose.

======================================================================
7. CREDIT THE ORIGINAL PROJECT
======================================================================

DEW must NEVER hide its origin.

Document clearly:

- original upstream project
- original authors
- repository URL
- fork/base relationship
- what original code provided
- what DEW changed after the fork
- what DEW no longer shares architecturally
- license obligations

Use precise language:

FORKED FROM
ADAPTED FROM
INSPIRED BY
LEARNED FROM
COMPARED AGAINST
REIMPLEMENTED
INDEPENDENT IMPLEMENTATION

These terms are NOT interchangeable.

======================================================================
8. DESIGN INFLUENCE POLICY
======================================================================

DEW learns from public engineering work.

Do not hide those influences.

When an external project materially influenced a DEW capability,
document:

PROJECT
AUTHOR / ORGANIZATION
UPSTREAM URL
RELEVANT CAPABILITY
WHAT THEY BUILT
WHAT PROBLEM THEY SOLVED
WHAT DEW LEARNED
WHAT DEW DID DIFFERENTLY
WHAT DEW DID NOT ADOPT
WHY
TRADE-OFF

Never imply that DEW invented an external concept.

Never imply DEW contains external code if it only learned an idea.

Never imply direct derivation when it is only conceptual influence.

======================================================================
9. REQUIRED DESIGN INFLUENCE RESEARCH
======================================================================

At minimum document verified influence from:

Original upstream / Crush

Context Mode
https://github.com/mksglu/context-mode

claude-mem
https://github.com/thedotmack/claude-mem

Hindsight
https://github.com/vectorize-io/hindsight

Superpowers
https://github.com/obra/superpowers

Ponytail
https://github.com/dietrichgebert/ponytail

Add other repositories ONLY if Git/reports/source prove
material influence.

======================================================================
10. CREATE DESIGN LINEAGE DOCUMENT
======================================================================

Create:

docs/design/influences.md

Include:

CAPABILITY
PROBLEM
INFLUENCE
UPSTREAM
WHAT WE LEARNED
DEW ADAPTATION
WHAT CHANGED
WHAT WAS REJECTED
WHY
CURRENT STATUS

Example conceptual mapping to VERIFY:

Context Mode
→ Working Context / Artifact containment

claude-mem
→ Engineering Continuity / Session continuity

Hindsight
→ Historical Knowledge / provenance / lifecycle

Superpowers
→ engineering methodology

Ponytail
→ simplicity / complexity discipline

Crush
→ original application/runtime foundation

Agent CLI / DEW-specific experience
→ provider governance
→ unified workspace
→ verification/evidence
→ Routine
→ cross-provider execution isolation

DO NOT accept this mapping blindly.

Verify against source/history.

======================================================================
11. EXPLAIN ARCHITECTURE AS LAYERS
======================================================================

The architecture documentation must not be a flat feature list.

Find the ACTUAL architectural layers.

For each layer document:

NAME
PURPOSE
PROBLEM SOLVED
INPUT
OUTPUT
STATE OWNED
AUTHORITATIVE DATA
COMPONENTS
MAIN TYPES
MAIN PACKAGES
PERSISTENCE
DEPENDENCIES
UPSTREAM INFLUENCE
FAILURE MODES
RECOVERY
SECURITY BOUNDARY
PROVIDER INTERACTION
TESTS
EVIDENCE
TRADE-OFFS
NON-GOALS

======================================================================
12. THREE INFORMATION LAYERS MUST BE EXPLAINED CLEARLY
======================================================================

If V3 verification confirms this architecture, explain explicitly that:

WORKING CONTEXT

ENGINEERING CONTINUITY

HISTORICAL KNOWLEDGE

are NOT three competing memory systems.

They are three different information lifecycles.

--------------------------------------------------
LAYER: WORKING CONTEXT
--------------------------------------------------

Question:

WHAT DOES THE MODEL NEED RIGHT NOW?

Examples:

current task
relevant source
selected tool output
relevant evidence
active skills

Possible lineage:

Context Mode lessons.

Lifecycle:

PRODUCED
→ STORED
→ INDEXED
→ SELECTED
→ INJECTED
→ EXPIRED

--------------------------------------------------
LAYER: ENGINEERING CONTINUITY
--------------------------------------------------

Question:

WHAT DOES THIS ENGINEERING SESSION NEED TO CONTINUE?

Examples:

decisions
progress
failed approaches
pending tasks
changed files
verification status
session digest

Possible lineage:

claude-mem lessons.

Lifecycle:

EVENT
→ OBSERVED
→ DIGESTED
→ SESSION-PERSISTED
→ RETRIEVED
→ REVALIDATED

--------------------------------------------------
LAYER: HISTORICAL KNOWLEDGE
--------------------------------------------------

Question:

WHAT VERIFIED LESSON SHOULD SURVIVE ACROSS SESSIONS?

Examples:

verified architectural decisions
reusable failure lessons
project conventions
known historical constraints
validated approaches

Possible lineage:

Hindsight lessons.

Lifecycle:

CANDIDATE
→ VERIFIED
→ RETAINED
→ RECALLED
→ REVALIDATED
→ CORRECTED
→ SUPERSEDED
→ INVALIDATED
→ FORGOTTEN

--------------------------------------------------

Explain why combining these three blindly creates problems.

Explain why:

CONTEXT
≠
SESSION CONTINUITY
≠
HISTORICAL MEMORY.

======================================================================
13. APPLY THE SAME LAYER THINKING TO THE ENTIRE SYSTEM
======================================================================

Do not stop at Context / Continuity / Knowledge.

Reconstruct the whole system using clear layers.

Possible hypothesis to verify:

LAYER 0 — CURRENT TRUTH

Git
files
runtime
tests
evidence

LAYER 1 — PRODUCT / DAILY WORKSPACE

Workspace
Session
user intent
daily UX

LAYER 2 — EXECUTION

Run
TaskGraph
Task
Attempt

LAYER 3 — AGENT EXECUTION

Agent
Reviewer
Verifier
specialized workers

LAYER 4 — METHODOLOGY

Skills
Superpowers
Ponytail
personal/project policies

LAYER 5 — TOOL / WORLD ACTION

filesystem
shell
Git
MCP
browser
external tools

LAYER 6 — WORKING CONTEXT

Artifact
search
selection
budget
composition

LAYER 7 — ENGINEERING CONTINUITY

events
SessionDigest
resume state

LAYER 8 — HISTORICAL KNOWLEDGE

verified historical lessons
provenance
lifecycle
retrieval

LAYER 9 — VERIFICATION

tests
lint
typecheck
review
behavior validation

LAYER 10 — EVIDENCE

revision-bound proof
verification results

LAYER 11 — PROVIDER GOVERNANCE

provider policy
model policy
fallback
invocation enforcement

LAYER 12 — PERSISTENCE / RECOVERY

durable state
leases
attempts
restart
recovery

This is ONLY a hypothesis.

Use actual code to produce the final architecture.

Merge layers if implementation shows they belong together.

Split layers if necessary.

Remove layers that do not exist.

======================================================================
14. FOR EVERY LAYER EXPLAIN THE FLOW
======================================================================

Every architectural layer must answer:

WHY DOES THIS EXIST?

WHAT ENTERS?

WHAT HAPPENS?

WHAT LEAVES?

WHO OWNS STATE?

WHAT IS AUTHORITATIVE?

WHAT HAPPENS IF IT FAILS?

HOW IS IT RECOVERED?

WHAT EXTERNAL IDEA INFLUENCED IT?

WHAT DID DEW CHANGE?

WHAT TRADE-OFF DID WE ACCEPT?

======================================================================
15. DOCUMENT END-TO-END FLOWS
======================================================================

Create:

docs/architecture/flows.md

Do not document architecture only as boxes.

Document actual end-to-end flows.

At minimum:

FLOW 1
New user request

FLOW 2
Continue existing session

FLOW 3
Workspace switch

FLOW 4
Provider switch

FLOW 5
Qwen-only execution

FLOW 6
Agent spawning

FLOW 7
Skill selection

FLOW 8
Tool execution

FLOW 9
Large tool output

FLOW 10
Context composition

FLOW 11
Session digest

FLOW 12
Historical recall

FLOW 13
Memory revalidation

FLOW 14
Code modification

FLOW 15
Verification

FLOW 16
Evidence recording

FLOW 17
DONE decision

FLOW 18
Routine / unattended run

FLOW 19
Crash and same-worker recovery

FLOW 20
Crash and replacement-worker recovery

FLOW 21
Stale evidence

FLOW 22
Stale historical knowledge

FLOW 23
Branch switch

FLOW 24
Worktree isolation

FLOW 25
Failure / retry

FLOW 26
Provider unavailable

FLOW 27
Forbidden cross-provider fallback

FLOW 28
Application restart

FLOW 29
Fresh-session resume

FLOW 30
Multi-workspace operation

======================================================================
16. EVERY FLOW MUST SHOW BOUNDARIES
======================================================================

For every important flow show:

USER
↓
WORKSPACE
↓
SESSION
↓
RUN
↓
TASK
↓
AGENT
↓
SKILLS
↓
CONTEXT
↓
PROVIDER
↓
TOOLS
↓
SOURCE/RUNTIME
↓
VERIFICATION
↓
EVIDENCE
↓
SESSION CONTINUITY
↓
OPTIONAL HISTORICAL KNOWLEDGE

But adapt the flow to reality.

Mark:

STATE READ
STATE WRITE
LLM CALL
TOOL CALL
PERSISTENCE
TRUST BOUNDARY
PROVIDER BOUNDARY
SECURITY BOUNDARY
VERIFICATION BOUNDARY

======================================================================
17. FEATURE DOCUMENTATION MUST EXPLAIN THE PROBLEM
======================================================================

Never document:

"DEW has SessionDigest."

Instead:

PROBLEM

Large transcripts are expensive and poor resume artifacts.

FAILURE

New sessions must reconstruct progress manually.

DESIGN

Structured SessionDigest.

FLOW

events → digest → persistence → resume → revalidation

INFLUENCE

claude-mem / internal research.

DEW CHANGE

...

TRADE-OFF

...

FAILURE MODE

...

TEST

...

STATUS

VERIFIED / etc.

Every major capability should follow this structure.

======================================================================
18. CASE STUDIES
======================================================================

Create:

docs/design/case-studies.md

Explain important real engineering failures.

Examples to verify:

fragmented provider state

hidden cross-provider calls

Qwen execution unexpectedly invoking Claude

subagent provider leakage

session resume failure

workspace contamination

branch/worktree contamination

false DONE

stale evidence

long-running zombie task

replacement-worker recovery

context explosion

skill collision

multiple planning authorities

memory overriding current source

stale historical knowledge

memory feedback loops

partial-success persistence

accepted != durable

retrieval scope failure

host adapter mismatch

cold-start race

For each:

PROBLEM
REAL OBSERVATION
ROOT CAUSE
DESIGN RESPONSE
IMPLEMENTATION
TEST
RESULT
TRADE-OFF

======================================================================
19. EXTERNAL ISSUE → DEW CASE STUDY
======================================================================

Where V3 research learned from external issues:

link the upstream issue.

Explain:

WHAT FAILED UPSTREAM?

WHY?

WHAT GENERAL LESSON DID DEW EXTRACT?

DOES DEW HAVE THE SAME FAILURE?

WHAT TEST DID DEW CREATE?

WHAT DID DEW IMPLEMENT?

Never write:

"We fixed issue #X"

unless DEW literally owns that issue.

Write:

"DEW's design was informed by failure pattern X observed in ..."

======================================================================
20. TRADE-OFF DOCUMENT
======================================================================

Create:

docs/design/trade-offs.md

For every significant architectural decision:

BENEFIT
COST
ALTERNATIVE
WHY CHOSEN
WHEN IT BECOMES WRONG
ROLLBACK / SIMPLIFICATION PATH

Examples:

artifact storage vs raw context

structured digest vs full transcript

deterministic retrieval vs semantic retrieval

single agent vs multi-agent

single process vs background worker

historical knowledge vs repo-native documentation

event log vs event sourcing

provider flexibility vs provider isolation

automatic skills vs explicit skills

automatic memory vs verified promotion

======================================================================
21. REJECTED DESIGN DOCUMENT
======================================================================

Create:

docs/design/rejected-designs.md

Document ideas intentionally NOT adopted.

Possible examples only if verified:

full event sourcing

graph memory

Reflect

mental models

semantic retrieval everywhere

always-on memory injection

always-on multi-agent

one worktree per agent role

cross-provider silent fallback

multiple orchestration engines

multiple scope resolvers

provider-owned session continuity

memory as current truth

Explain why rejected.

This prevents future contributors from unknowingly
reintroducing previously rejected complexity.

======================================================================
22. ARCHITECTURE OWNERSHIP MAP
======================================================================

Create:

docs/architecture/capability-ownership.md

For each capability:

CAPABILITY
PRIMARY OWNER
ADVISORS
SOURCE OF TRUTH
WRITERS
READERS
PERSISTENCE
CONFLICT POLICY

Include:

workspace
session
run
task
attempt
agent
planning
skills
provider selection
model selection
tools
Git
worktree
context
compaction
session continuity
historical knowledge
review
verification
evidence
Routine
permissions
recovery

One capability should not accidentally have multiple authorities.

======================================================================
23. STATE MODEL
======================================================================

Create:

docs/architecture/state-model.md

Clearly distinguish:

CURRENT STATE

EVENT HISTORY

SESSION CONTINUITY

HISTORICAL KNOWLEDGE

DERIVED KNOWLEDGE

VERIFICATION

EVIDENCE

LLM INFERENCE

Document authority ordering.

Explain stale-state rules.

======================================================================
24. CURRENT TRUTH HIERARCHY
======================================================================

If verified by implementation, document:

CURRENT VERIFIED SOURCE/RUNTIME
>
CURRENT VERIFIED EVIDENCE
>
CURRENT WORKING CONTEXT
>
VERIFIED SESSION CONTINUITY
>
VERIFIED HISTORICAL KNOWLEDGE
>
DERIVED HISTORICAL KNOWLEDGE
>
LLM INFERENCE

Explain examples.

Example:

Memory says function A still exists.

Git says function A was removed.

Git wins.

Tests passed on revision A.

Current revision is B.

Evidence from A is stale.

======================================================================
25. PROVIDER ARCHITECTURE
======================================================================

Create or rewrite:

docs/providers.md

Explain:

provider != workspace

provider != session identity

provider is execution backend.

Explain policy resolution.

Explain model allowlist.

Explain fallback.

Explain forbidden silent cross-provider fallback.

Explain local/free/offline profiles.

Explain provider observability.

Explain how background LLM operations inherit policy.

Document the Qwen-only regression case.

======================================================================
26. SKILLS ARCHITECTURE
======================================================================

Create or rewrite:

docs/skills.md

Explain:

Skill
vs
Policy
vs
Agent
vs
Workflow
vs
Runtime.

Document:

Ponytail
Superpowers
personal skills
project skills

where actually used.

Explain selective activation.

Explain why more skills != better.

Explain conflict resolution.

Explain capability ownership.

======================================================================
27. AGENT ARCHITECTURE
======================================================================

Create or rewrite:

docs/agents.md

Explain:

minimum sufficient agents.

Explain:

planner
implementer
reviewer
verifier
research agents

ONLY if these roles actually exist.

Explain agent independence.

Explain write ownership.

Explain worktrees.

Explain provider inheritance.

Explain context specialization.

Explain why multi-agent is optional.

======================================================================
28. ROUTINE ARCHITECTURE
======================================================================

Create or rewrite:

docs/routines.md

Explain:

RoutineDefinition
→ Run
→ TaskGraph
→ Task
→ Attempt
→ Agent

if verified.

Explain long-running work.

Explain crash recovery.

Explain retry budgets.

Explain stall detection.

Explain why Routine is not a second runtime.

======================================================================
29. VERIFICATION AND EVIDENCE
======================================================================

Create:

docs/verification-and-evidence.md

Explain:

model says DONE
≠
DONE.

Document:

REQUESTED
ACCEPTED
STARTED
EXECUTED
PERSISTED
VERIFIED
EVIDENCE_RECORDED
COMPLETED

Explain revision-bound evidence.

Explain stale evidence.

Explain reviewer vs verifier.

Explain completion gate.

======================================================================
30. CONTEXT DOCUMENT
======================================================================

Create:

docs/context.md

Explain:

raw tool output
artifact
index
search
selection
context budget
composition
injection
expiration

Explain influence from Context Mode.

Explain what DEW changed.

Explain why context optimization is not a security sandbox.

======================================================================
31. CONTINUITY DOCUMENT
======================================================================

Create:

docs/continuity.md

Explain:

engineering events
session state
SessionDigest
resume
revalidation

Explain influence from claude-mem.

Explain scope:

workspace
session
run
task
branch
worktree
revision

Explain how DEW avoids cross-session/project contamination.

======================================================================
32. HISTORICAL KNOWLEDGE DOCUMENT
======================================================================

Create:

docs/knowledge.md

Explain:

historical knowledge is advisory.

Explain promotion criteria.

Explain provenance.

Explain lifecycle.

Explain correction.

Explain supersession.

Explain invalidation.

Explain deletion.

Explain revalidation.

Explain influence from Hindsight.

Explain which Hindsight capabilities DEW rejected.

======================================================================
33. README REWRITE
======================================================================

Rewrite README.md from scratch based on verified current reality.

README should answer in this order:

1. DEW
2. One-sentence explanation
3. Why DEW exists
4. The fragmentation problem
5. What DEW changes
6. Quick start
7. Daily workflow
8. Architecture overview
9. Major layers
10. Important flows
11. Provider model
12. Skills
13. Agents
14. Routine
15. Context / Continuity / Knowledge
16. Verification / Evidence
17. Design influences
18. Project history
19. Current status
20. Documentation map
21. Roadmap
22. Credits
23. License

Do not make README a 20,000-line architecture manual.

README explains.

Detailed docs prove.

======================================================================
34. README ARCHITECTURE OVERVIEW
======================================================================

README should contain ONE clear architecture diagram.

It should allow a new reader to understand the product in under
approximately one minute.

Then link to:

docs/architecture.md

for depth.

======================================================================
35. ARCHITECTURE DOCUMENT
======================================================================

Create/rewrite:

docs/architecture.md

This is the canonical architectural overview.

It must show:

LAYERS
FLOWS
STATE
AUTHORITY
BOUNDARIES
FAILURE MODEL
PERSISTENCE
PROVIDER POLICY
VERIFICATION
INFORMATION LIFECYCLES

Link to deeper docs.

======================================================================
36. DOCUMENTATION INFORMATION ARCHITECTURE
======================================================================

Target structure MAY resemble:

README.md

docs/
├── getting-started.md
├── why-dew.md
├── daily-workflow.md
├── project-history.md
├── architecture.md
├── providers.md
├── skills.md
├── agents.md
├── routines.md
├── context.md
├── continuity.md
├── knowledge.md
├── verification-and-evidence.md
├── configuration.md
├── troubleshooting.md
├── roadmap.md
│
├── architecture/
│   ├── current-state.md
│   ├── evolution.md
│   ├── flows.md
│   ├── state-model.md
│   ├── authority-model.md
│   ├── capability-ownership.md
│   ├── scope-model.md
│   ├── provider-boundary.md
│   └── persistence-and-recovery.md
│
└── design/
    ├── influences.md
    ├── crush-origin.md
    ├── context-mode.md
    ├── claude-mem.md
    ├── hindsight.md
    ├── superpowers.md
    ├── ponytail.md
    ├── case-studies.md
    ├── trade-offs.md
    └── rejected-designs.md

Adapt to actual repository.

Do not create empty documentation for aesthetic symmetry.

======================================================================
37. DOCUMENTATION DEDUPLICATION
======================================================================

Audit every Markdown file.

Classify:

CURRENT
PARTIALLY_STALE
STALE
CONTRADICTORY
DUPLICATED
HISTORICAL_ONLY
GENERATED_REPORT
SHOULD_MERGE
SHOULD_ARCHIVE
SHOULD_DELETE
UNKNOWN

Do not leave three docs describing the same architecture differently.

Choose canonical owners.

Example:

README
→ product overview

architecture.md
→ architecture overview

providers.md
→ provider architecture

context.md
→ working context

continuity.md
→ session continuity

knowledge.md
→ historical knowledge

reports/
→ evidence/history/research

======================================================================
38. PRODUCT DOCS VS REPORTS
======================================================================

Separate:

PRODUCT DOCUMENTATION

from:

ENGINEERING REPORTS.

Product docs describe the current system.

Reports preserve:

research
experiments
benchmarks
historical audits
migration evidence
V2/V3 work.

Do not force users to read research reports to understand DEW.

======================================================================
39. RENAME PLAN
======================================================================

Before renaming anything:

audit every occurrence of:

Agent CLI
agent-cli
AgentCLI
old binary name
old module/package identifiers
repo URLs
docs
scripts
CI
release config
Docker
environment variables
config directories
database namespaces
telemetry
user-agent strings
shell completion
examples
tests

Create:

reports/documentation/RENAME_PLAN.md

Classify each rename:

DISPLAY_NAME
REPOSITORY_NAME
BINARY_NAME
PACKAGE_NAME
MODULE_PATH
CONFIG_PATH
ENV_PREFIX
DATABASE_NAMESPACE
API_IDENTIFIER
USER_AGENT
DOC_REFERENCE

======================================================================
40. RENAME SAFETY
======================================================================

Do not blindly rename internal identifiers.

Some renames can break:

imports
Go module paths
GitHub references
configuration
existing user data
scripts
CI
plugins
MCP
database state
shell completion.

For each rename determine:

SAFE_NOW
MIGRATION_REQUIRED
BACKWARD_COMPATIBILITY_REQUIRED
DEFER
DO_NOT_RENAME

======================================================================
41. TARGET PRODUCT IDENTITY
======================================================================

Unless current evidence/repository constraints require reconsideration:

Product:

DEW

Full name:

Daily Engineering Workspace

Tagline:

One place for engineering.

North Star:

DEW is where engineering work lives.
Providers are how the work gets done.

Product category:

AI-native Daily Engineering Workspace

Current primary interface:

DEW CLI

Do not put CLI into the fundamental product identity.

CLI is an interface.

======================================================================
42. REPOSITORY RENAME
======================================================================

If repository rename is authorized and technically available:

prepare rename from:

agent-cli

to the verified chosen repository name.

Candidate:

dew

BUT:

before performing remote rename,
check namespace collision,
package/module implications,
links,
automation,
CI,
release workflows,
and remote permissions.

If remote repository rename requires an external permission/action
that is unavailable:

DO NOT fabricate success.

Prepare the exact migration.

Rename local product references that are safe.

Report:

REMOTE RENAME REQUIRES HUMAN ACTION

only if truly necessary.

======================================================================
43. BACKWARD COMPATIBILITY
======================================================================

Where appropriate consider temporary compatibility:

agent-cli command
→ warning / alias
→ dew

old config path
→ migration
→ new path

old environment variables
→ compatibility period

But do not add compatibility complexity automatically.

Measure actual need.

======================================================================
44. CHANGELOG RECONSTRUCTION
======================================================================

Create or completely reconstruct:

CHANGELOG.md

Do not write only the latest release.

Reconstruct meaningful evolution from Git.

Prefer architecture/product eras over noisy commit dumps.

Possible structure:

Unreleased

DEW V3
- Added
- Changed
- Fixed
- Removed
- Security
- Architecture

DEW V2

Agent CLI era

Fork foundation

Use actual Git evidence.

======================================================================
45. CHANGELOG MUST EXPLAIN WHY
======================================================================

Important changes should not only say:

"Added SessionDigest."

Prefer:

"Added structured session continuity to reduce transcript replay
and improve resume correctness across fresh sessions."

For architectural changes include short WHY where useful.

======================================================================
46. CHANGELOG TRACEABILITY
======================================================================

Where useful link:

commit
issue
report
benchmark
upstream influence.

Do not overload every tiny entry.

Trace major architectural changes.

======================================================================
47. CREATE ARCHITECTURE CHANGELOG
======================================================================

In addition to normal CHANGELOG.md create:

docs/architecture/evolution.md

This document answers:

HOW DID THE ARCHITECTURE CHANGE?

Normal changelog answers:

WHAT CHANGED?

Architecture evolution answers:

WHY DID THE SYSTEM CHANGE SHAPE?

======================================================================
48. DOCUMENTATION MINDSET
======================================================================

Create:

docs/DOCUMENTATION_MINDSET.md

This becomes a permanent project rule.

Core rule:

IF YOU CHANGE THE SYSTEM,
CHANGE THE EXPLANATION OF THE SYSTEM.

======================================================================
49. FEATURE CHANGE DOCUMENTATION RULE
======================================================================

Every future meaningful feature/change must answer:

WHAT PROBLEM ARE WE SOLVING?

WHERE DOES IT BELONG?

WHICH LAYER OWNS IT?

WHAT FLOW CHANGES?

WHAT STATE CHANGES?

WHAT AUTHORITY CHANGES?

WHAT PERSISTENCE CHANGES?

WHAT PROVIDER BEHAVIOR CHANGES?

WHAT SECURITY BOUNDARY CHANGES?

WHAT FAILURE MODES ARE INTRODUCED?

WHAT TEST PROVES IT?

WHAT EVIDENCE PROVES IT?

WHAT TRADE-OFF DOES IT ADD?

WHAT DOCS MUST CHANGE?

WHAT CHANGELOG ENTRY IS REQUIRED?

======================================================================
50. NEW FEATURE ARCHITECTURE CHECK
======================================================================

Before implementing a major new capability:

PROBLEM
↓
CURRENT OWNER
↓
CURRENT FLOW
↓
GAP
↓
SIMPLEST DESIGN
↓
TARGET LAYER
↓
AUTHORITY CHECK
↓
STATE CHECK
↓
FAILURE CHECK
↓
TEST
↓
IMPLEMENT
↓
VERIFY
↓
DOCUMENT
↓
CHANGELOG

No:

FEATURE
↓
CODE
↓
DONE.

======================================================================
51. LAYER RULE
======================================================================

Every significant capability must belong somewhere.

Do not allow:

"utility that just exists."

For every capability identify:

PRODUCT LAYER
EXECUTION LAYER
AGENT LAYER
METHODOLOGY LAYER
TOOL LAYER
CONTEXT LAYER
CONTINUITY LAYER
KNOWLEDGE LAYER
VERIFICATION LAYER
EVIDENCE LAYER
PROVIDER LAYER
PERSISTENCE/RECOVERY LAYER

or whatever final architecture establishes.

If no layer makes sense:

question whether the capability belongs in DEW.

======================================================================
52. FLOW RULE
======================================================================

Every significant new feature must update at least one documented flow.

If a developer cannot answer:

WHERE DOES THIS FEATURE ENTER THE SYSTEM?

WHAT CALLS IT?

WHAT DOES IT CALL?

WHAT STATE DOES IT READ?

WHAT STATE DOES IT WRITE?

WHAT HAPPENS AFTER IT?

then the design is incomplete.

======================================================================
53. SOURCE-OF-TRUTH RULE
======================================================================

Every new persistent state must declare:

OWNER
SOURCE OF TRUTH
WRITERS
READERS
LIFECYCLE
INVALIDATION
RECOVERY
MIGRATION
RETENTION

Do not create orphan state.

======================================================================
54. AUTHORITY RULE
======================================================================

Before adding a new engine ask:

Does another component already own this capability?

Examples:

new planner
new memory system
new context injector
new provider resolver
new task scheduler
new session store
new verification engine.

If YES:

extend existing owner unless evidence proves replacement is needed.

======================================================================
55. EXTERNAL IDEA MINDSET
======================================================================

When learning from a new public repository:

DO NOT:

repo
→ copy feature.

Use:

repo
→ understand problem
→ inspect source
→ inspect history
→ inspect issues
→ inspect fixes
→ inspect forks
→ inspect regressions
→ extract invariant
→ compare DEW
→ reproduce DEW gap
→ design simplest adaptation
→ POC
→ benchmark
→ red team
→ keep/reject
→ document attribution.

======================================================================
56. EXTERNAL IDEA ATTRIBUTION
======================================================================

If a future change is materially influenced by another project,
update:

docs/design/influences.md

Include upstream link.

If specific issue materially informed a failure model,
link that issue.

If specific implementation/code was reused,
follow license requirements and make derivation explicit.

======================================================================
57. ARCHITECTURE DECISION RECORDS
======================================================================

For major irreversible/high-impact architecture decisions,
use ADRs.

Suggested:

docs/adr/

Each ADR:

CONTEXT
PROBLEM
OPTIONS
DECISION
WHY
TRADE-OFFS
FAILURE MODES
ROLLBACK
EVIDENCE
RELATED COMMITS
RELATED DOCS
EXTERNAL INFLUENCES

Do not create ADRs for trivial changes.

======================================================================
58. CHANGELOG MINDSET
======================================================================

Every user-visible or architecture-significant change must update
CHANGELOG.md.

Categories:

ADDED
CHANGED
FIXED
REMOVED
SECURITY
PERFORMANCE
ARCHITECTURE
DOCUMENTATION

Do not put implementation noise into the changelog.

======================================================================
59. COMMIT MINDSET
======================================================================

Commits should make architectural history recoverable.

Prefer commits that answer one coherent change.

Commit messages should make it possible later to reconstruct:

problem
subsystem
change
reason.

Avoid meaningless history such as:

update
fix
stuff
changes
misc.

======================================================================
60. COMMIT → CHANGELOG → DOCS TRACE
======================================================================

For architecture-significant work maintain conceptual trace:

COMMIT
→ CHANGELOG
→ ARCHITECTURE/FLOW DOC
→ TEST
→ EVIDENCE

Not every commit needs all five.

Every major system change should.

======================================================================
61. DOCUMENTATION UPDATE TRIGGERS
======================================================================

Automatically consider documentation updates when changing:

workspace behavior
session behavior
run/task lifecycle
provider policy
model policy
fallback
agent behavior
skills
Routine
context
compaction
artifact storage
continuity
memory/knowledge
verification
evidence
persistence
recovery
security
permissions
configuration
CLI
defaults
user-visible workflow
architecture ownership

======================================================================
62. DOCUMENTATION CHANGE MATRIX
======================================================================

Create:

docs/DOCUMENTATION_CHANGE_MATRIX.md

Example:

CHANGE                         DOCS
--------------------------------------------------------
Workspace semantics            workspaces + architecture
Session semantics              continuity + architecture
Provider resolution            providers + flows
Skill activation               skills + daily workflow
Context composition            context + architecture
Historical knowledge           knowledge + state model
Verification                   verification-and-evidence
Routine recovery               routines + recovery
CLI command                    README/getting-started
Architecture owner             capability-ownership
External influence             influences
Breaking change                changelog + migration

Adapt to final documentation structure.

======================================================================
63. DOCS-AS-CODE VERIFICATION
======================================================================

Documentation must be tested against current implementation.

Verify:

CLI commands exist

flags exist

configuration keys exist

defaults match

file paths match

provider names match

model names match where documented

state names match

examples run where feasible

internal links resolve

external links resolve

architecture type/package names exist

no deleted feature is described as current

no experimental feature is described as verified

======================================================================
64. DOCUMENTATION DRIFT TEST
======================================================================

Create an automated or semi-automated docs validation strategy.

At minimum check:

broken internal links

known command examples

config examples

referenced files

referenced packages

renamed product strings

old repository URLs

old binary names

stale Agent CLI branding

duplicated canonical architecture docs.

Automate what is cheap and deterministic.

======================================================================
65. OLD NAME SWEEP
======================================================================

After rename:

search entire repository for:

Agent CLI
agent-cli
old upstream branding

Classify every remaining occurrence:

INTENTIONAL HISTORICAL REFERENCE

UPSTREAM ATTRIBUTION

BACKWARD COMPATIBILITY

STALE — FIX

Do not blindly delete historical attribution.

======================================================================
66. LINK VALIDATION
======================================================================

Validate links to:

original upstream

Context Mode

claude-mem

Hindsight

Superpowers

Ponytail

issues referenced

commits referenced

internal docs.

======================================================================
67. CREDIT / ACKNOWLEDGEMENTS
======================================================================

Create an appropriate:

ACKNOWLEDGEMENTS.md

or credits section.

Clearly distinguish:

ORIGINAL FOUNDATION

DESIGN INFLUENCE

METHODOLOGY INFLUENCE

RESEARCH REFERENCE

DIRECT DEPENDENCY

CODE DERIVATION

Do not over-credit trivial similarities.

Do not under-credit material influence.

======================================================================
68. LICENSE AUDIT
======================================================================

Before claiming rename/release complete:

inspect licenses of:

original fork

directly reused code

vendored code

copied snippets

external assets.

Ensure required notices/attribution remain.

Design inspiration alone should be documented honestly,
but do not falsely claim code derivation.

======================================================================
69. FRESH-READER TEST
======================================================================

After docs are rewritten:

start a fresh-context reviewer.

Give them ONLY:

README
public docs
repository.

Do not give V2/V3 internal discussion.

Ask them:

What is DEW?

Why does it exist?

How is it different from a provider CLI?

What is Workspace?

What is Session?

What is Run?

What is an Agent?

What is a Skill?

What is Routine?

What is Working Context?

What is Engineering Continuity?

What is Historical Knowledge?

Why are those three different?

Who owns current truth?

Can memory override Git?

What does provider mean?

Can Qwen silently call Claude?

What does DONE mean?

How does verification work?

How does crash recovery work?

What projects influenced DEW?

What did DEW change from those projects?

What are the main trade-offs?

What is current vs experimental?

If they cannot answer accurately:

docs are not finished.

======================================================================
70. ARCHITECTURE FRESH-READER TEST
======================================================================

Give a second fresh-context engineer:

docs/architecture.md
docs/architecture/flows.md
docs/architecture/state-model.md

Ask them to draw DEW architecture independently.

Compare their diagram to actual implementation.

Large disagreement means documentation is unclear.

======================================================================
71. CONTRIBUTOR TEST
======================================================================

Ask a fresh-context agent:

"Add a hypothetical capability X."

Do NOT let it implement.

Ask where X belongs.

It should be able to identify:

layer
owner
flow
state
tests
docs
changelog

using documentation alone.

If not:

documentation does not sufficiently communicate architecture.

======================================================================
72. README SHOULD NOT HIDE COMPLEXITY
======================================================================

README should make DEW understandable.

It should NOT expose every internal mechanism immediately.

Use progressive disclosure:

README
→ architecture overview
→ subsystem docs
→ design docs
→ reports/research.

======================================================================
73. README SHOULD NOT BECOME MARKETING FICTION
======================================================================

Avoid unsupported claims such as:

"perfect memory"

"zero context loss"

"fully autonomous"

"always correct"

"production ready"

unless evidence genuinely supports them.

Prefer measurable claims.

======================================================================
74. REQUIRED DOCUMENTATION REPORTS
======================================================================

Create:

reports/documentation/
00_DOCS_BASELINE.md
01_GIT_HISTORY_RECONSTRUCTION.md
02_PROJECT_EVOLUTION_ERAS.md
03_ARCHITECTURE_RECONSTRUCTION.md
04_LAYER_MAP.md
05_FLOW_MAP.md
06_DESIGN_LINEAGE.md
07_CASE_STUDY_MAP.md
08_TRADEOFF_MAP.md
09_RENAME_PLAN.md
10_RENAME_RESULTS.md
11_CHANGELOG_RECONSTRUCTION.md
12_DOCS_MIGRATION.md
13_DOCS_VALIDATION.md
14_LINK_VALIDATION.md
15_FRESH_READER_REVIEW.md
16_ARCHITECTURE_READER_REVIEW.md
17_CONTRIBUTOR_REVIEW.md
18_ATTRIBUTION_AUDIT.md
19_LICENSE_AUDIT.md
20_DOCS_CONSISTENCY_AUDIT.md
FINAL_DOCUMENTATION_AUDIT.md

======================================================================
75. DOCUMENTATION PROGRESS LOG
======================================================================

Maintain:

reports/documentation/WORKFLOW_PROGRESS.md

Use only engineering-level reasoning/results.

Tags:

DISCOVERY
DECISION
DEVIATION
NEW_UNKNOWN
ASSUMPTION_INVALIDATED
HISTORY_FOUND
ARCHITECTURE_FOUND
DOC_DRIFT
ATTRIBUTION
RENAME
REGRESSION
ROLLBACK
VERIFIED_RESULT

Do not dump private chain-of-thought.

======================================================================
76. FINAL DOCUMENTATION CONSISTENCY GATE
======================================================================

Before completion verify:

README
docs
source
tests
config
CLI
changelog
architecture diagrams
history
credits
repository naming

describe the SAME system.

There must not be:

README architecture A

docs architecture B

source architecture C

V3 report presented as architecture D.

Research reports may preserve old ideas,
but must be clearly historical/research artifacts.

======================================================================
77. FINAL CHANGELOG GATE
======================================================================

Ensure CHANGELOG reflects:

major evolution

major fixes

major removals

breaking changes

rename

V3 architecture

documentation reconstruction

without pretending every commit was a release.

======================================================================
78. FINAL RENAME GATE
======================================================================

Verify:

product name
README
docs
binary
help text
version output
config
CI
release
package/module
repository links
examples
shell completion
tests

are consistent with the approved rename plan.

======================================================================
79. FINAL ARCHITECTURE GATE
======================================================================

A new engineer must be able to answer:

WHAT ARE THE LAYERS?

WHY DOES EACH LAYER EXIST?

WHAT FLOWS THROUGH EACH LAYER?

WHO OWNS STATE?

WHAT IS CURRENT TRUTH?

WHAT IS CONTEXT?

WHAT IS CONTINUITY?

WHAT IS HISTORICAL KNOWLEDGE?

WHAT IS PROVIDER EXECUTION?

WHAT IS VERIFICATION?

WHAT IS EVIDENCE?

WHAT HAPPENS WHEN SOMETHING FAILS?

======================================================================
80. FINAL HISTORY GATE
======================================================================

A contributor must be able to understand:

WHERE THE PROJECT CAME FROM

WHY IT FORKED

HOW IT EVOLVED

WHICH MAJOR PROBLEMS SHAPED IT

WHICH EXTERNAL PROJECTS INFLUENCED IT

WHAT WAS ADOPTED

WHAT WAS CHANGED

WHAT WAS REJECTED

WHY CURRENT ARCHITECTURE EXISTS.

======================================================================
81. FINAL PROJECT STORY
======================================================================

The documentation should make this evolution understandable:

SEPARATE AI TOOLS
        ↓
FRAGMENTED ENGINEERING STATE
        ↓
AGENT CLI
        ↓
MULTI-PROVIDER WORKSPACE
        ↓
WORKSPACE + SESSION CONTINUITY
        ↓
PROVIDER GOVERNANCE
        ↓
MULTI-AGENT / SKILLS / ROUTINE
        ↓
VERIFICATION + EVIDENCE
        ↓
DAILY ENGINEERING WORKSPACE
        ↓
WORKING CONTEXT
        +
ENGINEERING CONTINUITY
        +
HISTORICAL KNOWLEDGE
        ↓
DEW

But VERIFY this story against Git history.

Rewrite it if the repository proves a different evolution.

======================================================================
82. FINAL PRODUCT EXPLANATION
======================================================================

Target explanation, only if verified:

DEW is a Daily Engineering Workspace.

It provides one place for engineering work across projects,
sessions, agents, skills and AI providers.

DEW owns the continuity and execution model of the work.

Providers execute AI workloads inside that model.

Working Context controls what information is useful now.

Engineering Continuity preserves the state needed to resume work.

Historical Knowledge preserves verified lessons across sessions.

Skills guide methodology.

Agents perform bounded roles.

Tools interact with the external world.

Verification checks the result.

Evidence proves the current revision is ready.

Historical knowledge never outranks current verified source/runtime.

======================================================================
83. FINAL TAGLINE
======================================================================

Preferred:

DEW — Daily Engineering Workspace

One place for engineering.

North Star:

DEW is where engineering work lives.
Providers are how the work gets done.

If branding review discovers a materially better or safer naming choice,
document the evidence before changing this.

Do not casually rename again.

======================================================================
84. PERMANENT PROJECT MINDSET
======================================================================

From this point forward:

CODE IS NOT THE WHOLE SYSTEM.

The system is:

CODE
+
STATE
+
FLOW
+
AUTHORITY
+
FAILURE MODEL
+
TESTS
+
EVIDENCE
+
DOCUMENTATION
+
HISTORY.

A feature is not complete merely because code compiles.

A feature is complete when:

PROBLEM UNDERSTOOD
+
OWNER IDENTIFIED
+
LAYER IDENTIFIED
+
FLOW DEFINED
+
STATE DEFINED
+
FAILURE MODES DEFINED
+
IMPLEMENTED
+
TESTED
+
VERIFIED
+
DOCUMENTED
+
CHANGELOG UPDATED.

======================================================================
85. PERMANENT DOCUMENTATION INVARIANT
======================================================================

CODE CHANGED
+
ARCHITECTURE CHANGED
+
DOCS NOT CHANGED

=

WORK NOT COMPLETE.

======================================================================
86. PERMANENT HISTORY INVARIANT
======================================================================

Important engineering decisions must remain reconstructable later.

Do not destroy useful architectural history during cleanup.

Archive historical reports when appropriate.

Do not confuse archived history with current documentation.

======================================================================
87. PERMANENT ATTRIBUTION INVARIANT
======================================================================

IF WE LEARNED SOMETHING MATERIAL FROM PUBLIC WORK:

CREDIT IT.

Explain:

WHAT THEY DID

WHAT WE LEARNED

WHAT WE CHANGED

WHY WE CHANGED IT.

Open-source influence is part of the engineering history,
not something to hide.

======================================================================
88. PERMANENT SIMPLICITY INVARIANT
======================================================================

A new layer must earn its existence.

A new engine must earn its existence.

A new background worker must earn its existence.

A new persistent state must earn its existence.

A new LLM call must earn its existence.

A new abstraction must earn its existence.

If an existing owner can solve the problem cleanly:

extend it.

Do not create a second authority.

======================================================================
89. PERMANENT FLOW INVARIANT
======================================================================

For every future major capability:

SHOW THE FLOW.

Do not document architecture only as nouns.

Document:

TRIGGER
→ INPUT
→ OWNER
→ PROCESS
→ STATE READ
→ STATE WRITE
→ SIDE EFFECT
→ VERIFICATION
→ OUTPUT
→ FAILURE
→ RECOVERY.

======================================================================
90. PERMANENT LAYER INVARIANT
======================================================================

For every future major capability:

SHOW THE LAYER.

Explain:

WHY IT BELONGS THERE.

Explain:

WHAT LAYERS IT MAY READ.

Explain:

WHAT LAYERS IT MAY WRITE.

Explain:

WHAT IT MUST NEVER OVERRIDE.

======================================================================
91. PERMANENT CHANGELOG INVARIANT
======================================================================

Every architecture-significant or user-visible change:

MUST update CHANGELOG.

Every breaking change:

MUST include migration impact.

Every removal:

MUST explain replacement or reason where relevant.

======================================================================
92. PERMANENT DOC SYNC INVARIANT
======================================================================

At the end of every substantial development workflow:

RUN DOC SYNC.

DOC SYNC must inspect:

changed files
changed public types
changed CLI
changed config
changed state
changed architecture
changed flows
changed provider behavior
changed persistence
changed recovery
changed verification

and update canonical documentation.

======================================================================
93. PRE-COMMIT / PRE-MERGE DOCUMENTATION QUESTION
======================================================================

For meaningful changes ask:

DID THIS CHANGE:

product behavior?
architecture?
layer ownership?
flow?
state?
provider behavior?
config?
CLI?
failure handling?
recovery?
verification?
security?
external influence?

If YES:

which docs changed?

If NONE changed:

record why documentation update is unnecessary.

======================================================================
94. FINAL AUDIT
======================================================================

Create:

reports/documentation/FINAL_DOCUMENTATION_AUDIT.md

Answer:

What is the final product name?

What is the final repository name?

What is the binary name?

What was the original upstream?

What was the exact fork point?

What major eras occurred?

What does current architecture look like?

What are its layers?

Why does every layer exist?

What are the important flows?

What owns current truth?

What owns continuity?

What owns historical knowledge?

What owns provider execution?

What owns verification?

What owns evidence?

Which public projects materially influenced DEW?

What exactly came from each?

What was adapted?

What was reimplemented?

What was rejected?

What trade-offs remain?

Which major case studies shaped the architecture?

What is VERIFIED?

What is EXPERIMENTAL?

What is PLANNED?

What docs were deleted?

What docs were archived?

What docs became canonical?

Was README fresh-reader tested?

Were architecture docs fresh-reader tested?

Was attribution audited?

Was license compliance checked?

Was rename completed?

Were old names swept?

Was CHANGELOG reconstructed?

Are docs consistent with current HEAD?

======================================================================
95. FINAL STATUS
======================================================================

Choose exactly one:

DEW DOCUMENTATION READY — VERIFIED

DEW DOCUMENTATION READY — WITH KNOWN LIMITATIONS

DOCUMENTATION INCOMPLETE — MORE RECONSTRUCTION REQUIRED

RENAME INCOMPLETE — MIGRATION REQUIRED

DOCUMENTATION DRIFT DETECTED — NOT READY

======================================================================
96. COMPLETION RULE
======================================================================

Do NOT declare completion because README looks good.

Completion requires:

GIT HISTORY RECONSTRUCTED
+
CURRENT ARCHITECTURE RECONSTRUCTED
+
LAYERS DOCUMENTED
+
FLOWS DOCUMENTED
+
STATE/AUTHORITY DOCUMENTED
+
CASE STUDIES DOCUMENTED
+
TRADE-OFFS DOCUMENTED
+
DESIGN INFLUENCES ATTRIBUTED
+
CHANGELOG RECONSTRUCTED
+
RENAME VERIFIED
+
DOCS SYNCHRONIZED
+
DOCS-AS-CODE CHECKED
+
FRESH-READER REVIEWED
+
FINAL CONSISTENCY AUDITED.

======================================================================
97. START
======================================================================

Start from the latest verified V3 checkpoint.

Do not begin by writing README.

Begin by reconstructing truth:

1. verify HEAD
2. verify V3 completion
3. find upstream/fork point
4. inspect complete Git evolution
5. reconstruct architectural eras
6. reconstruct current architecture from source/runtime/tests
7. map layers
8. map flows
9. map authority/state
10. map case studies
11. map design influences
12. audit current docs
13. design rename migration
14. reconstruct CHANGELOG
15. rewrite canonical docs
16. rewrite README
17. perform rename safely
18. synchronize remaining docs
19. validate docs against code
20. run fresh-reader reviews
21. run attribution/license audit
22. run final consistency audit
23. produce FINAL_DOCUMENTATION_AUDIT.md
24. choose final status

DO NOT FABRICATE HISTORY.

DO NOT FABRICATE ATTRIBUTION.

DO NOT FABRICATE IMPLEMENTATION STATUS.

DO NOT HIDE THE ORIGINAL AUTHORS.

DO NOT HIDE PUBLIC PROJECT INFLUENCES.

DO NOT DOCUMENT DESIGN INTENT AS CURRENT REALITY.

DO NOT LEAVE MULTIPLE CONFLICTING ARCHITECTURE DOCS.

DO NOT RENAME REMOTE RESOURCES WITHOUT AUTHORIZATION.

DO NOT CALL CODE DONE WHILE DOCUMENTATION IS STALE.

SHOW THE LAYERS.

SHOW THE FLOWS.

SHOW THE OWNERS.

SHOW THE STATE.

SHOW THE FAILURE MODEL.

SHOW THE TRADE-OFFS.

SHOW THE HISTORY.

SHOW THE EVIDENCE.

CREDIT THE PEOPLE AND PROJECTS WE LEARNED FROM.

MAKE THE REPOSITORY ITSELF EXPLAIN WHY DEW EXISTS AND HOW IT WORKS.
```

Điểm quan trọng nhất của prompt này là nó tạo thành **luật lâu dài của DEW**, không chỉ dọn docs một lần.

Từ sau V3, mindset phát triển sẽ trở thành:

**Problem → Owner → Layer → Flow → State → Failure → Implementation → Test → Evidence → Docs → Changelog.**

Như vậy sau này bạn lại nghiên cứu một repo mới và lấy một ý tưởng hay, agent không được đơn giản “thêm feature”. Nó phải xác định **feature nằm tầng nào, flow nào thay đổi, ai sở hữu state, có tạo authority thứ hai không, failure mới là gì, học từ repo/issue nào, trade-off gì, test nào chứng minh, rồi cập nhật docs + changelog**. Đây chính là thứ giúp DEW vài chục hoặc vài trăm commit sau vẫn đọc được lịch sử kiến trúc thay vì lại biến thành một repo đầy tính năng nhưng không ai hiểu tại sao chúng tồn tại.

contorl sau khi cái v3 xong nhé và nên tận dùng thêm model haiku 5.5 sonnet 5.5 để giảm tỉa opus nhé 