package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/oschwand/gristdl/internal/grist"
	"github.com/spf13/cobra"
)

const pullStateFile = ".gristdl-pull.json"

var (
	pullOutput string
	pullForce  bool
)

type pullOptions struct {
	Output string
	Force  bool
}

type pullClient interface {
	Orgs(ctx context.Context) ([]grist.Org, error)
	Workspaces(ctx context.Context, orgID int64) ([]grist.Workspace, error)
	DownloadDoc(ctx context.Context, docID string, w io.Writer) error
}

type pullDocState struct {
	Path      string `json:"path"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type pullState struct {
	Version int                     `json:"version"`
	Docs    map[string]pullDocState `json:"docs"`
}

type pullStats struct {
	Downloaded int
	Moved      int
	Skipped    int
	Refused    int
	Failed     int
}

var pullCmd = &cobra.Command{
	Use:          "pull",
	Short:        "Download all documents into a folder hierarchy mirroring orgs and workspaces",
	SilenceUsage: true,
	Long: "Download every accessible document as a .grist file under OUTPUT/ORG/WORKSPACE/.\n" +
		"Documents are tracked by ID in " + pullStateFile + ", so renamed or moved documents\n" +
		"are renamed or moved locally instead of downloaded again. Files are never\n" +
		"overwritten unless --force is given.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newGristClient()
		if err != nil {
			return err
		}
		return runPull(cmd.Context(), client, pullOptions{Output: pullOutput, Force: pullForce},
			cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	pullCmd.Flags().StringVarP(&pullOutput, "output", "o", ".",
		"target directory (org and workspace folders are created inside)")
	pullCmd.Flags().BoolVar(&pullForce, "force", false,
		"overwrite existing files and re-download up-to-date documents")
	rootCmd.AddCommand(pullCmd)
}

func runPull(ctx context.Context, client pullClient, opts pullOptions, stdout, stderr io.Writer) error {
	info, err := os.Stat(opts.Output)
	if err == nil && !info.IsDir() {
		return fmt.Errorf("output path %s is not a directory", opts.Output)
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	orgs, err := client.Orgs(ctx)
	if err != nil {
		return err
	}

	state, err := loadPullState(opts.Output)
	if err != nil {
		return err
	}

	type pullTarget struct {
		doc grist.Doc
		rel string
	}
	var targets []pullTarget
	orgNames := newNameAlloc()
	for _, org := range orgs {
		wsList, err := client.Workspaces(ctx, org.ID)
		if err != nil {
			fmt.Fprintf(stderr, "warning: skipping org %d: %v\n", org.ID, err)
			continue
		}
		orgID := strconv.FormatInt(org.ID, 10)
		var orgDir string
		if name := sanitizeName(org.Name); name != "Personal" {
			orgDir = orgNames.alloc(name, "org:"+orgID, orgID)
		}
		wsNames := newNameAlloc()
		for _, ws := range wsList {
			wsID := strconv.FormatInt(ws.ID, 10)
			wsDir := wsNames.alloc(sanitizeName(ws.Name), "ws:"+wsID, wsID)
			docNames := newNameAlloc()
			for _, doc := range ws.Docs {
				base := strings.TrimSuffix(sanitizeName(doc.Name), ".grist")
				if base == "" {
					base = "unnamed"
				}
				file := docNames.alloc(base, doc.ID, doc.ID) + ".grist"
				targets = append(targets, pullTarget{
					doc: doc,
					rel: path.Join(orgDir, wsDir, file),
				})
			}
		}
	}

	var stats pullStats
	seen := make(map[string]bool)
	for _, t := range targets {
		seen[t.doc.ID] = true
		if err := syncDoc(ctx, client, opts, state, t.doc, t.rel, &stats, stdout, stderr); err != nil {
			stats.Failed++
			fmt.Fprintf(stderr, "failed:   %s: %v\n", t.rel, err)
		}
	}

	if err := savePullState(opts.Output, state); err != nil {
		return err
	}

	stale := len(state.Docs) - len(seen)
	fmt.Fprintf(stdout, "%d downloaded, %d moved, %d up to date, %d refused, %d failed",
		stats.Downloaded, stats.Moved, stats.Skipped, stats.Refused, stats.Failed)
	if stale > 0 {
		fmt.Fprintf(stdout, ", %d no longer on the server (kept)", stale)
	}
	fmt.Fprintln(stdout)

	if stats.Refused > 0 || stats.Failed > 0 {
		return fmt.Errorf("%d document(s) not saved (%d refused: target file exists, use --force to overwrite; %d failed)",
			stats.Refused+stats.Failed, stats.Refused, stats.Failed)
	}
	return nil
}

func syncDoc(ctx context.Context, client pullClient, opts pullOptions, state *pullState,
	doc grist.Doc, rel string, stats *pullStats, stdout, stderr io.Writer) error {

	targetAbs := filepath.Join(opts.Output, filepath.FromSlash(rel))
	prev, known := state.Docs[doc.ID]

	if known && prev.Path != rel {
		oldAbs := filepath.Join(opts.Output, filepath.FromSlash(prev.Path))
		if _, err := os.Lstat(oldAbs); err == nil {
			if _, err := os.Lstat(targetAbs); err == nil && !opts.Force {
				stats.Refused++
				fmt.Fprintf(stderr, "refused: %s: file already exists (use --force to overwrite)\n", rel)
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
				return err
			}
			if err := os.Rename(oldAbs, targetAbs); err != nil {
				return err
			}
			state.Docs[doc.ID] = pullDocState{Path: rel, UpdatedAt: doc.UpdatedAt}
			stats.Moved++
			fmt.Fprintf(stdout, "moved:   %s (was %s)\n", rel, prev.Path)
			removeEmptyParents(opts.Output, filepath.Dir(oldAbs))
			return nil
		}
		known = false
	}

	_, statErr := os.Lstat(targetAbs)
	exists := statErr == nil
	if exists && !opts.Force {
		if known && prev.Path == rel && prev.UpdatedAt != "" && prev.UpdatedAt == doc.UpdatedAt {
			stats.Skipped++
			return nil
		}
		if !known {
			stats.Refused++
			fmt.Fprintf(stderr, "refused: %s: file already exists (use --force to overwrite)\n", rel)
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return err
	}
	if err := downloadAtomic(ctx, client, doc.ID, targetAbs); err != nil {
		return err
	}
	state.Docs[doc.ID] = pullDocState{Path: rel, UpdatedAt: doc.UpdatedAt}
	stats.Downloaded++
	fmt.Fprintf(stdout, "saved:   %s\n", rel)
	return nil
}

func downloadAtomic(ctx context.Context, client pullClient, docID, dst string) error {
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".gristdl-download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			os.Remove(tmpName)
		}
	}()
	if err := client.DownloadDoc(ctx, docID, tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	renamed = true
	return nil
}

func removeEmptyParents(root, dir string) {
	for {
		rel, err := filepath.Rel(root, dir)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func loadPullState(output string) (*pullState, error) {
	data, err := os.ReadFile(filepath.Join(output, pullStateFile))
	if errors.Is(err, fs.ErrNotExist) {
		return &pullState{Version: 1, Docs: map[string]pullDocState{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var state pullState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse %s: %w (delete it to start from scratch)", pullStateFile, err)
	}
	if state.Docs == nil {
		state.Docs = map[string]pullDocState{}
	}
	return &state, nil
}

func savePullState(output string, state *pullState) error {
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(output, ".gristdl-state-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(output, pullStateFile)); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

type nameAlloc struct {
	used map[string]string
}

func newNameAlloc() nameAlloc {
	return nameAlloc{used: map[string]string{}}
}

func (a nameAlloc) alloc(name, key, suffix string) string {
	if owner, ok := a.used[name]; !ok || owner == key {
		a.used[name] = key
		return name
	}
	for i := 1; ; i++ {
		cand := fmt.Sprintf("%s_%s", name, suffix)
		if i > 1 {
			cand = fmt.Sprintf("%s_%s_%d", name, suffix, i)
		}
		if owner, ok := a.used[cand]; !ok || owner == key {
			a.used[cand] = key
			return cand
		}
	}
}

func sanitizeName(name string) string {
	s := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if unicode.IsControl(r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	s = strings.Trim(s, " .")
	if s == "" {
		return "unnamed"
	}
	if r := []rune(s); len(r) > 100 {
		s = strings.Trim(string(r[:100]), " .")
		if s == "" {
			return "unnamed"
		}
	}
	return s
}
