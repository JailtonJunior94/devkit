---
name: crafting-effective-readmes
description: Crafts or improves README files with audience-aware structure, project-type templates, and review guidance. Use when creating, updating, extending, or auditing a README. Don't use for general prose editing that is unrelated to README structure or project documentation.
---

# Crafting Effective READMEs

## Input
- Identify whether the task is `creating`, `adding`, `updating`, or `reviewing`.
- Identify the project type as `open-source`, `personal`, `internal`, or `config`.
- Read `references/section-checklist.md` to confirm which sections fit the selected project type.
- Read `references/style-guide.md` before finalizing prose and structure.
- Read `references/using-references.md` only when deeper material is needed.

## Procedures

**Step 1: Classify the README task**
1. Determine whether the request is `creating`, `adding`, `updating`, or `reviewing`.
2. Determine the target audience and the project type before drafting.
3. If the project type is unclear, ask a single clarifying question instead of assuming an open-source template.

**Step 2: Gather project context**
1. For `creating`, identify the problem solved, the quickest path to successful usage, and anything notable to highlight.
2. For `adding`, identify what needs documentation, where it fits in the current README, and who needs the new section.
3. For `updating`, read the current README, identify stale sections, and compare them against the current project state.
4. For `reviewing`, audit the README against the codebase, commands, entry points, and operational reality.

**Step 3: Load the correct template**
1. Read `assets/oss-template.md` for open-source projects.
2. Read `assets/personal-template.md` for personal or portfolio projects.
3. Read `assets/internal-template.md` for team or company projects.
4. Read `assets/xdg-config-template.md` for config directories, dotfiles, or local tool folders.

**Step 4: Draft or revise the README**
1. Start with the minimum universal sections: name, description, and usage.
2. Add the sections required by `references/section-checklist.md` for the selected project type.
3. Prefer concrete setup instructions, examples, and operational guidance over generic marketing language.
4. Keep content audience-specific; document only what helps the intended reader succeed faster.

**Step 5: Validate the result**
1. Check that installation, usage, and examples match the actual project state.
2. Remove stale, duplicated, or speculative content.
3. If the README includes operational or internal guidance, verify owners, commands, and links.
4. End by asking whether anything important is still missing.

## Error Handling
- If the project type is ambiguous, ask one focused question instead of selecting a template by guesswork.
- If the codebase and README disagree, treat the current codebase and executable commands as the source of truth.
- If no README exists and the project context is too thin, produce a minimal skeleton with explicit placeholders rather than fabricating details.
- If the request is only about generic writing style, use README-specific structure guidance here and defer pure prose coaching to a writing skill when available.
