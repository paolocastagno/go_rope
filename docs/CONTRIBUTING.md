# Contributing to go_rope Documentation

Thank you for your interest in improving go_rope documentation! This guide helps you contribute effectively.

## Documentation Structure

```
docs/
├── INDEX.md           # Navigation guide
├── GETTING_STARTED.md # First-time user guide
├── ARCHITECTURE.md    # System design
├── CONFIGURATION.md   # Config reference
├── ROUTING.md         # Routing strategies
├── API.md            # Function reference
├── UTILITIES.md      # Utilities guide
├── EXAMPLES.md       # Runnable examples
└── CONTRIBUTING.md   # This file
```

## Before You Start

- Read existing documentation to avoid duplication
- Check if your topic is covered in [INDEX.md](INDEX.md)
- Review the style guide below
- Ensure examples are tested and correct

## Documentation Style Guide

### Writing Style

- **Clear and concise**: Use simple language
- **Active voice**: "Use this function" not "This function can be used"
- **Second person**: "You can configure..." not "One can configure..."
- **Practical focus**: Show real usage, not theory

### Markdown Format

```markdown
# Main Topic

Brief introduction sentence.

## Subtopic

**Bold for important terms**

`inline code` for variables, functions, filenames

### Code Examples

Wrapped in triple backticks with language:

\`\`\`go
// Go code
\`\`\`

\`\`\`toml
# TOML config
\`\`\`

### Lists

Use bullet points for unordered:
- Item 1
- Item 2

Use numbers for sequential:
1. First step
2. Second step
```

### Structure

Each documentation file should have:

1. **Title** (H1)
2. **Brief introduction**
3. **Table of contents** (for long docs)
4. **Main sections** (H2)
5. **Subsections** (H3) as needed
6. **Code examples** with context
7. **Best practices** section
8. **Related links** at the end

### Example Template

```markdown
# Topic Title

One-sentence summary of what this document covers.

## Section 1

Introductory explanation with context.

### Subsection 1.1

Detailed explanation with examples:

\`\`\`go
// Example code
\`\`\`

### Subsection 1.2

Additional details.

## Best Practices

- Practice 1: Explanation
- Practice 2: Explanation

## See Also

- [Related Doc](RELATED.md)
- [API Reference](API.md)
```

## Documentation Maintenance Checklist

When updating documentation, ensure:

- [ ] Code examples compile and run (or mark as pseudocode)
- [ ] Function signatures match actual code
- [ ] Parameter types are accurate
- [ ] Configuration examples are valid TOML
- [ ] Related documents are linked
- [ ] Cross-references are updated
- [ ] No broken links
- [ ] Consistent formatting with existing docs
- [ ] Updated [INDEX.md](INDEX.md) if needed
- [ ] Updated [README.md](../README.md) if needed

## Common Documentation Tasks

### Adding a New Topic

1. Create new `.md` file in `docs/`
2. Use [structure template](#structure) above
3. Add entry to [INDEX.md](INDEX.md)
4. Link from related documents
5. Update [README.md](../README.md) if top-level topic

### Updating Existing Documentation

1. Read the entire document for context
2. Make changes while preserving structure
3. Update any cross-references
4. Check code examples still work
5. Verify links are correct

### Fixing Errors

1. Identify the error clearly
2. Provide correction with context
3. Verify the fix against source code
4. Update related documentation
5. Test examples if applicable

## Code Examples

### Good Example

```go
// Demonstrates complete, runnable pattern
histogram := util.NewHistogram(0.01)  // 10ms bins
histogram.Add(0.125)                  // Add measurement
histogram.Print("output.txt")         // Save results
```

### Documentation of Example

```
[Description of what example does]

**Parameters:**
- `binSize` (float64): Size of each bin in seconds

**Returns:** Pointer to new Histogram

**Example:**
[Include code in triple backticks]
```

## Testing Documentation

### For Code Examples

1. Copy example code verbatim
2. Create test file
3. Verify it compiles: `go build`
4. Verify it runs: `go run`
5. Update documentation if needed

### For Configuration Examples

1. Create `.toml` file with example
2. Verify valid TOML syntax
3. Check all required fields present
4. Test with actual client/server if possible
5. Verify output in documentation

## Linking Guide

### Internal Links

```markdown
# Link to another doc
[Architecture Guide](ARCHITECTURE.md)

# Link to section in document
See [Routing Strategies](#routing-strategies) above

# Link from README to docs
Check the [CONFIGURATION Guide](docs/CONFIGURATION.md)
```

### External Links

```markdown
[QUIC Protocol](https://datatracker.ietf.org/doc/html/rfc9000)
```

## Common Mistakes to Avoid

❌ **Don't:**
- Copy outdated information from old docs
- Use jargon without explanation
- Mix code from different examples
- Leave broken links
- Add opinions instead of facts
- Use non-standard formatting

✅ **Do:**
- Verify against source code
- Explain acronyms on first use
- Provide complete, working examples
- Test all links
- Stick to facts and documentation
- Follow existing style

## Adding Examples

- Add to [EXAMPLES.md](EXAMPLES.md)
- Include complete, runnable code
- Explain what example demonstrates
- Show expected output
- Reference related documentation
- Keep examples focused and simple

## Changelog

Document significant documentation changes:

```markdown
## [Date]
- Added section on [topic]
- Updated [section] with new information
- Fixed incorrect example in [section]
- Improved clarity in [section]
```

## Review Process

Before submitting:

1. **Self-review**: Read through your changes
2. **Link check**: Verify all links work
3. **Format check**: Ensure consistent formatting
4. **Example check**: Test all code examples
5. **Cross-reference check**: Update related docs
6. **Spelling check**: Proofread content

## Questions?

If you're unsure about:
- **Content**: Check existing similar sections
- **Style**: Review other `.md` files
- **Structure**: Reference this guide or [ARCHITECTURE.md](ARCHITECTURE.md)
- **Technical accuracy**: Check source code

## Recognition

Contributors who significantly improve documentation will be:
- Mentioned in commit messages
- Recognized in the project README
- Added to CONTRIBUTORS file (coming soon)

## Documentation Improvement Ideas

Areas that could use improvement:
- [ ] More real-world examples
- [ ] Video tutorials (future)
- [ ] Performance benchmarking guide
- [ ] Troubleshooting FAQ
- [ ] Deployment guide
- [ ] Performance tuning guide

## Getting Your Changes Merged

1. Fork the repository
2. Create a branch: `git checkout -b docs/topic-name`
3. Make your changes
4. Follow the checklist above
5. Commit with clear message: `docs: Add [topic] documentation`
6. Push and create a pull request
7. Update based on review feedback

## Thank You!

Good documentation makes go_rope more accessible to everyone. Thank you for contributing! 🙏

---

**Quick Links:**
- [Documentation Index](INDEX.md)
- [Getting Started](GETTING_STARTED.md)
- [Architecture](ARCHITECTURE.md)
- [API Reference](API.md)
