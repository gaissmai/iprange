# Language

- ALWAYS respond in English, regardless of the language used in questions or comments.
- Code comments and documentation must always be written in English.
- Variable, function, and all identifier names must always be in English.

# AI Assistant Rules

When generating or modifying code:

- Do not use emojis or typographic quotes
- NEVER use Unicode characters, ALWAYS use plain ASCII.
- Do not flag issues in PR reviews that are already covered by linters.
- Only comment on logic, correctness, architecture, security, and issues that linters would not catch.
- Whenever you conduct a PR review, ALWAYS post a message at the end, even if you didn't find anything.
- Always format Go source files using `gofmt` after making modifications.
- When writing Go tests, if a top-level test calls `t.Parallel()`, all of its child subtests (`t.Run`) must also call `t.Parallel()`.
- Refer to [.greptile/codebase_summary.md](file:///home/gaissmai/project/iprange/.greptile/codebase_summary.md) for a detailed architectural and API reference of this package.


