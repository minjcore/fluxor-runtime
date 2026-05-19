// Package main builds an LLMWiki inventory: a unified metadata warehouse (JSON) + INDEX.md.
// The default provider scans repo .md files as source "local_markdown". Additional items
// (Jira, Confluence, crawled web pages, etc.) can be merged from JSON via -merge.
//
//	go run ./cmd/llmwiki-inventory [-root=.] [-out=LLMWiki] [-skip=...] [-merge=a.json,b.json]
//
// See cmd/llmwiki-inventory/README.md and LLMWiki/SCHEMA.md.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxTitleScan = 8192

// SchemaVersion documents the inventory.json shape for downstream tools.
const SchemaVersion = "2.0"

// Known source values (extend as you add connectors). Not enforced — any string is allowed.
const (
	SourceLocalMarkdown = "local_markdown"
)

// InventoryItem is one row in the shared warehouse. External systems should emit the same shape.
type InventoryItem struct {
	ID         string         `json:"id"`                    // globally unique, e.g. local_md:docs/a.md, jira:PROJ-1
	Source     string         `json:"source"`                // local_markdown, jira, confluence, web, ...
	Ref        string         `json:"ref"`                   // repo-relative path, issue key, page id, etc.
	Title      string         `json:"title"`
	URL        string         `json:"url,omitempty"`         // optional browse URL
	ModifiedAt string         `json:"modified_at,omitempty"` // RFC3339 UTC
	SizeBytes  int64          `json:"size_bytes,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"` // source-specific (project, space, author, ...)
}

type inventoryDoc struct {
	SchemaVersion string          `json:"schema_version"`
	GeneratedAt   string          `json:"generated_at"`
	RepoRoot      string          `json:"repo_root"` // absolute; used to resolve local_markdown paths
	ItemCount     int             `json:"item_count"`
	Items         []InventoryItem `json:"items"`
}

type mergeFile struct {
	Items []InventoryItem `json:"items"`
}

func main() {
	rootFlag := flag.String("root", ".", "Repository root to scan for local_markdown")
	outFlag := flag.String("out", "LLMWiki", "Output directory for inventory.json and INDEX.md")
	skipExtra := flag.String("skip", "", "Comma-separated extra directory base names to skip")
	mergeFlag := flag.String("merge", "", "Comma-separated paths to JSON files with {\"items\":[...]} to merge")
	flag.Parse()

	absRoot, err := filepath.Abs(*rootFlag)
	if err != nil {
		log.Fatalf("root path: %v", err)
	}
	if st, err := os.Stat(absRoot); err != nil || !st.IsDir() {
		log.Fatalf("root %q missing or not a directory: %v", absRoot, err)
	}

	absOut, err := filepath.Abs(*outFlag)
	if err != nil {
		log.Fatalf("out path: %v", err)
	}

	skipNames := defaultSkipNames()
	for _, p := range strings.Split(*skipExtra, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			skipNames[p] = struct{}{}
		}
	}

	outRel, err := filepath.Rel(absRoot, absOut)
	if err != nil {
		outRel = ""
	}

	items := scanLocalMarkdown(absRoot, outRel, skipNames)

	for _, p := range strings.Split(*mergeFlag, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		extra, err := loadMergeFile(p)
		if err != nil {
			log.Fatalf("merge %q: %v", p, err)
		}
		items = append(items, extra...)
	}

	items = dedupeByID(items)

	sort.Slice(items, func(i, j int) bool {
		if items[i].Source != items[j].Source {
			return items[i].Source < items[j].Source
		}
		if items[i].Ref != items[j].Ref {
			return items[i].Ref < items[j].Ref
		}
		return items[i].ID < items[j].ID
	})

	doc := inventoryDoc{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		RepoRoot:      absRoot,
		ItemCount:     len(items),
		Items:         items,
	}

	if err := os.MkdirAll(absOut, 0o755); err != nil {
		log.Fatalf("mkdir out: %v", err)
	}

	jsonPath := filepath.Join(absOut, "inventory.json")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatalf("json: %v", err)
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		log.Fatalf("write inventory.json: %v", err)
	}

	indexPath := filepath.Join(absOut, "INDEX.md")
	if err := writeIndex(indexPath, items); err != nil {
		log.Fatalf("write INDEX.md: %v", err)
	}

	fmt.Printf("Wrote %d items to %s and %s\n", len(items), jsonPath, indexPath)
}

func scanLocalMarkdown(absRoot, outRel string, skipNames map[string]struct{}) []InventoryItem {
	var items []InventoryItem
	_ = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipOutputTree(rel, outRel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if _, skip := skipNames[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		slashPath := filepath.ToSlash(rel)
		title := firstMarkdownTitle(path)
		items = append(items, InventoryItem{
			ID:         "local_md:" + slashPath,
			Source:     SourceLocalMarkdown,
			Ref:        slashPath,
			Title:      title,
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
			SizeBytes:  info.Size(),
		})
		return nil
	})
	return items
}

func loadMergeFile(path string) ([]InventoryItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m mergeFile
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m.Items, nil
}

func dedupeByID(items []InventoryItem) []InventoryItem {
	seen := make(map[string]int) // id -> index in out
	var out []InventoryItem
	for _, it := range items {
		if it.ID == "" {
			log.Printf("skip item with empty id (source=%q ref=%q)", it.Source, it.Ref)
			continue
		}
		if i, ok := seen[it.ID]; ok {
			out[i] = it // later wins
			continue
		}
		seen[it.ID] = len(out)
		out = append(out, it)
	}
	return out
}

func defaultSkipNames() map[string]struct{} {
	names := []string{
		"node_modules", "vendor", "dist", ".git", ".cursor", "target",
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}

func shouldSkipOutputTree(rel, outRel string) bool {
	if outRel == "" || strings.HasPrefix(outRel, "..") {
		return false
	}
	r := filepath.ToSlash(rel)
	o := filepath.ToSlash(outRel)
	return r == o || strings.HasPrefix(r, o+"/")
}

func firstMarkdownTitle(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, maxTitleScan)
	n, _ := f.Read(buf)
	s := bufio.NewScanner(bytes.NewReader(buf[:n]))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			line = strings.TrimLeft(line, "#")
			line = strings.TrimSpace(line)
			return line
		}
	}
	return ""
}

func writeIndex(path string, items []InventoryItem) error {
	var b strings.Builder
	b.WriteString("# LLMWiki index\n\n")
	b.WriteString("Unified inventory: repo markdown (relative links) and external rows (URL when present).\n\n")

	bySource := make(map[string][]InventoryItem)
	for _, it := range items {
		src := it.Source
		if src == "" {
			src = "(unknown_source)"
		}
		bySource[src] = append(bySource[src], it)
	}
	sources := make([]string, 0, len(bySource))
	for s := range bySource {
		sources = append(sources, s)
	}
	sort.Strings(sources)

	for _, src := range sources {
		list := bySource[src]
		fmt.Fprintf(&b, "## source: `%s` (%d)\n\n", src, len(list))

		if src == SourceLocalMarkdown {
			writeLocalMarkdownSection(&b, list)
			continue
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Ref < list[j].Ref })
		for _, it := range list {
			label := it.Title
			if label == "" {
				label = it.Ref
			}
			if it.URL != "" {
				fmt.Fprintf(&b, "- [%s](%s) — `%s`\n", escapeMdLinkLabel(label), it.URL, escapeMdLinkLabel(it.Ref))
			} else {
				fmt.Fprintf(&b, "- %s — `%s`\n", escapeMdLinkLabel(label), escapeMdLinkLabel(it.Ref))
			}
		}
		b.WriteString("\n")
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeLocalMarkdownSection(b *strings.Builder, list []InventoryItem) {
	byTop := make(map[string][]InventoryItem)
	for _, it := range list {
		top := topSegment(it.Ref)
		byTop[top] = append(byTop[top], it)
	}
	tops := make([]string, 0, len(byTop))
	for k := range byTop {
		tops = append(tops, k)
	}
	sort.Strings(tops)

	for _, top := range tops {
		fmt.Fprintf(b, "### %s\n\n", top)
		sub := byTop[top]
		sort.Slice(sub, func(i, j int) bool { return sub[i].Ref < sub[j].Ref })
		for _, it := range sub {
			label := it.Title
			if label == "" {
				label = filepath.Base(it.Ref)
			}
			link := "../" + it.Ref
			fmt.Fprintf(b, "- [%s](%s)\n", escapeMdLinkLabel(label), link)
		}
		b.WriteString("\n")
	}
}

func topSegment(rel string) string {
	rel = filepath.ToSlash(rel)
	i := strings.Index(rel, "/")
	if i < 0 {
		return "(root)"
	}
	return rel[:i]
}

func escapeMdLinkLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	return s
}
