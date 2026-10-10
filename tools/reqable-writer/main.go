// Command reqable-writer bypasses the Reqable MCP premium gate by writing
// the app's JSON config files directly, then asking for a restart.
//
// Same behavior as scripts/grok_direct_write.py, but as a single static Go
// binary (fast startup, no interpreter). Restart of Reqable is required
// after a write because the app caches config in Box LMDB at startup.
//
// Usage:
//
//	reqable-writer list
//	reqable-writer rewrite --name NAME --url URL [--payload BODY] [--method M] [--nowildcard]
//	reqable-writer rewrite-modify --name NAME --url URL --pattern P --replacement R
//	reqable-writer breakpoint --name NAME --url URL
//	reqable-writer collection --name NAME
//	reqable-writer script --name NAME --url URL --code FILE
//	reqable-writer delete-rewrite --id ID
//	reqable-writer delete-breakpoint --id ID
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const appSupport = "Library/Application Support/com.reqable.macosx"

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		fatal(err)
	}
	return h
}

// resolveUserDir finds the active Reqable user directory. The app keeps one
// directory per login; prefer the one with the most rewrites (the real
// account) and fall back to the first one that has a capture_rewrite file.
func resolveUserDir() string {
	base := filepath.Join(homeDir(), appSupport, "user")
	entries, err := os.ReadDir(base)
	if err != nil {
		fatal(err)
	}
	best := ""
	bestCount := -1
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(base, e.Name(), "config", "capture_rewrite")
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg struct {
			Rewrites []json.RawMessage `json:"rewrites"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			continue
		}
		if len(cfg.Rewrites) > bestCount {
			bestCount = len(cfg.Rewrites)
			best = filepath.Join(base, e.Name())
		}
	}
	if best == "" {
		fatalf("no Reqable user dir with config/capture_rewrite under %s", base)
	}
	return best
}

func loadJSON(path string, v any) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(v); err != nil {
		fatalf("%s: %v", path, err)
	}
}

// saveJSON writes atomically (tmp + rename) so a crash cannot leave a half
// written config behind.
func saveJSON(path string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		fatal(err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "reqable-writer:", err)
	os.Exit(1)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "reqable-writer: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "list":
		cmdList(flag.NewFlagSet("list", flag.ExitOnError))
	case "rewrite":
		cmdRewrite(newFlagSet("rewrite"))
	case "rewrite-modify":
		cmdRewriteModify(newFlagSet("rewrite-modify"))
	case "breakpoint":
		cmdBreakpoint(newFlagSet("breakpoint"))
	case "collection":
		cmdCollection(newFlagSet("collection"))
	case "script":
		cmdScript(newFlagSet("script"))
	case "delete-rewrite":
		cmdDeleteRewrite(newFlagSet("delete-rewrite"))
	case "delete-breakpoint":
		cmdDeleteBreakpoint(newFlagSet("delete-breakpoint"))
	default:
		usage()
	}
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	return fs
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: reqable-writer <command> [flags]

commands:
  list                                   show rewrites and breakpoints
  rewrite          --name N --url U [--payload B] [--method M] [--nowildcard]
  rewrite-modify   --name N --url U --pattern P --replacement R
  breakpoint       --name N --url U
  collection       --name N
  script           --name N --url U --code FILE
  delete-rewrite   --id ID
  delete-breakpoint --id ID

restart Reqable after any write so it reloads config from disk.`)
	os.Exit(2)
}
