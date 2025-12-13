# Code Instructions:
## General
- Don't overcomplicate or overengineer. Write simple, clean, readable code
- Only write ESSENTIAL comments. avoid slop and descriptive comments
- Follow existing code style and conventions in the project
- No workarounds, fallbacks, or fooling me
- Do not simplify code to cheat fixing errors without explicit user approval
- Don't use emojis
- Always use the latest feature. Read relevant code if necessary.

## Go-Specific

### Style & Conventions
- Follow official Go style guide and common conventions
- Use `gofmt` formatting - proper indentation and spacing
- Name interfaces with -er suffix when single method (Reader, Writer)
- Use short variable names in small scopes (i, j, err, ctx), descriptive names in larger scopes
- Package names: lowercase, single word, no underscores
- Avoid stuttering: `user.UserID` → `user.ID`

### Error Handling
- Always check errors, never ignore with `_`
- Return errors, don't panic except in truly unrecoverable situations
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Use custom error types when callers need to distinguish error cases
- Don't use `panic()` for normal error handling

### Concurrency
- Use context.Context for cancellation and timeouts
- Always handle context cancellation in goroutines
- Avoid goroutine leaks - ensure all goroutines can exit
- Use channels for communication, mutexes for state
- Prefer sync.Once, sync.WaitGroup over manual coordination

### Modern Go (1.18+)
- Use generics where they reduce duplication without adding complexity
- Use any instead of interface{} (Go 1.18+)
- Use slog for structured logging (Go 1.21+)
- Use errors.Is() and errors.As() for error checking (Go 1.13+)
- Use time.Until() and time.Since() for time calculations

### Best Practices
- Accept interfaces, return structs
- Keep interfaces small and focused
- Use functional options pattern for complex constructors
- Prefer composition over inheritance
- Make zero values useful when possible
- Close resources with defer immediately after creation
- Use table-driven tests
- Avoid global mutable state
- Group imports: stdlib, external, internal (with blank lines)

### Project Structure
- Keep main.go minimal - just wiring
- Put business logic in /internal
- Use /cmd for multiple binaries
- Don't export unless necessary
- Group by feature/domain, not by layer (when it makes sense)

### Anti-patterns to Avoid
- No init() functions unless absolutely necessary
- Don't use reflection unless there's no alternative
- Avoid premature abstraction
- Don't create interfaces before you need them
- No method receivers on functions that don't need state
- Avoid naked returns in long functions
