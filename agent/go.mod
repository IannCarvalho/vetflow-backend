// This nested module intentionally isolates the `agent/` subtree (agent skills
// installed via `npx skills`) from the main vetflow-backend module.
//
// Those skills ship example .go files that import packages not in the main
// go.mod (e.g. spf13/cobra, fatih/color), which otherwise break
// `go build ./...` and `go mod tidy` at the repository root. A nested go.mod
// makes the Go toolchain treat this directory as a separate module and stop
// descending into it from the root module's `./...` wildcard.
//
// It is not part of the build, has no dependencies declared, and exists purely
// as a boundary. The `agent/` directory itself is also gitignored.
module vetflow-backend-agent-skills-isolation

go 1.23
