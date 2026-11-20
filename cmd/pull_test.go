package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oschwand/gristdl/internal/grist"
)

type fakePullClient struct {
	orgs      []grist.Org
	ws        map[int64][]grist.Workspace
	contents  map[string]string
	downloads map[string]int
}

func newFakePullClient() *fakePullClient {
	return &fakePullClient{
		ws:        map[int64][]grist.Workspace{},
		contents:  map[string]string{},
		downloads: map[string]int{},
	}
}

func (c *fakePullClient) Orgs(ctx context.Context) ([]grist.Org, error) {
	return c.orgs, nil
}

func (c *fakePullClient) Workspaces(ctx context.Context, orgID int64) ([]grist.Workspace, error) {
	return c.ws[orgID], nil
}

func (c *fakePullClient) DownloadDoc(ctx context.Context, docID string, w io.Writer) error {
	c.downloads[docID]++
	_, err := io.WriteString(w, c.contents[docID])
	return err
}

func newPullFixture() *fakePullClient {
	client := newFakePullClient()
	client.orgs = []grist.Org{{ID: 1, Name: "Acme"}}
	client.ws[1] = []grist.Workspace{
		{
			ID:   10,
			Name: "Home",
			Docs: []grist.Doc{
				{ID: "docAlpha", Name: "Alpha", UpdatedAt: "2026-01-01T00:00:00.000Z"},
				{ID: "docBeta", Name: "Beta", UpdatedAt: "2026-01-02T00:00:00.000Z"},
			},
		},
		{
			ID:   11,
			Name: "Other",
			Docs: []grist.Doc{},
		},
	}
	client.contents["docAlpha"] = "alpha-content"
	client.contents["docBeta"] = "beta-content"
	return client
}

func runPullTest(t *testing.T, client *fakePullClient, outDir string, force bool) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := runPull(context.Background(), client, pullOptions{Output: outDir, Force: force}, &stdout, &stderr)
	return stdout.String(), err
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestPullFreshDownload(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	stdout, err := runPullTest(t, client, outDir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for doc, want := range map[string]string{
		"Acme/Home/Alpha.grist": "alpha-content",
		"Acme/Home/Beta.grist":  "beta-content",
	} {
		if got := readFile(t, filepath.Join(outDir, filepath.FromSlash(doc))); got != want {
			t.Errorf("%s = %q, want %q", doc, got, want)
		}
	}
	if client.downloads["docAlpha"] != 1 || client.downloads["docBeta"] != 1 {
		t.Errorf("unexpected download counts: %v", client.downloads)
	}
	if !strings.Contains(stdout, "2 downloaded") {
		t.Errorf("summary = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(outDir, pullStateFile)); err != nil {
		t.Errorf("state file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "Acme", "Other")); err == nil {
		t.Errorf("empty workspace should not create a directory")
	}
}

func TestPullUnchangedSkips(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	stdout, err := runPullTest(t, client, outDir, false)
	if err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if client.downloads["docAlpha"] != 1 || client.downloads["docBeta"] != 1 {
		t.Errorf("re-downloaded unchanged docs: %v", client.downloads)
	}
	if !strings.Contains(stdout, "2 up to date") || !strings.Contains(stdout, "0 downloaded") {
		t.Errorf("summary = %q", stdout)
	}
	if readFile(t, filepath.Join(outDir, "Acme", "Home", "Alpha.grist")) != "alpha-content" {
		t.Errorf("file modified on no-op pull")
	}
}

func TestPullForceRedownloads(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	client.contents["docAlpha"] = "alpha-v2"
	if _, err := runPullTest(t, client, outDir, true); err != nil {
		t.Fatalf("force pull: %v", err)
	}
	if client.downloads["docAlpha"] != 2 || client.downloads["docBeta"] != 2 {
		t.Errorf("force should re-download everything: %v", client.downloads)
	}
}

func TestPullChangedDocRedownloads(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	client.contents["docBeta"] = "beta-v2"
	client.ws[1][0].Docs[1].UpdatedAt = "2026-02-01T00:00:00.000Z"
	stdout, err := runPullTest(t, client, outDir, false)
	if err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if client.downloads["docBeta"] != 2 || client.downloads["docAlpha"] != 1 {
		t.Errorf("unexpected download counts: %v", client.downloads)
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Home", "Beta.grist")); got != "beta-v2" {
		t.Errorf("Beta.grist = %q", got)
	}
	if !strings.Contains(stdout, "1 up to date") {
		t.Errorf("summary = %q", stdout)
	}
}

func TestPullRenameMovesFileWithoutDownload(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	client.ws[1][0].Docs[0].Name = "Alpha Renamed"
	client.ws[1][0].Docs[0].UpdatedAt = "2026-03-01T00:00:00.000Z"
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if client.downloads["docAlpha"] != 1 {
		t.Errorf("rename should not re-download: %v", client.downloads)
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Home", "Alpha Renamed.grist")); got != "alpha-content" {
		t.Errorf("renamed file content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(outDir, "Acme", "Home", "Alpha.grist")); !os.IsNotExist(err) {
		t.Errorf("old file still exists after rename")
	}
}

func TestPullMoveBetweenWorkspaces(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	moved := client.ws[1][0].Docs[1]
	moved.UpdatedAt = "2026-04-01T00:00:00.000Z"
	client.ws[1][0].Docs = []grist.Doc{client.ws[1][0].Docs[0]}
	client.ws[1][1].Docs = []grist.Doc{moved}
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if client.downloads["docBeta"] != 1 {
		t.Errorf("move should not re-download: %v", client.downloads)
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Other", "Beta.grist")); got != "beta-content" {
		t.Errorf("moved file content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(outDir, "Acme", "Home", "Beta.grist")); !os.IsNotExist(err) {
		t.Errorf("old file still exists after move")
	}
}

func TestPullRefusesForeignFileAndForceOverwrites(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outDir, "Acme", "Home"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "Acme", "Home", "Alpha.grist"), []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runPull(context.Background(), client, pullOptions{Output: outDir}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("want refusal error, got %v", err)
	}
	if !strings.Contains(stderr.String(), "refused") {
		t.Errorf("stderr = %q", stderr.String())
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Home", "Alpha.grist")); got != "precious" {
		t.Errorf("refused doc overwrote existing file: %q", got)
	}
	if client.downloads["docAlpha"] != 0 {
		t.Errorf("refused doc should not be downloaded: %v", client.downloads)
	}
	if client.downloads["docBeta"] != 1 {
		t.Errorf("other docs should still be pulled: %v", client.downloads)
	}

	stderr.Reset()
	if err := runPull(context.Background(), client, pullOptions{Output: outDir, Force: true}, &stdout, &stderr); err != nil {
		t.Fatalf("force pull: %v", err)
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Home", "Alpha.grist")); got != "alpha-content" {
		t.Errorf("force did not overwrite: %q", got)
	}
}

func TestPullDuplicateDocNamesGetIDSuffix(t *testing.T) {
	client := newPullFixture()
	client.ws[1][0].Docs = []grist.Doc{
		{ID: "doc1", Name: "Same", UpdatedAt: "t1"},
		{ID: "doc2", Name: "Same", UpdatedAt: "t1"},
	}
	client.contents["doc1"] = "one"
	client.contents["doc2"] = "two"
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Home", "Same.grist")); got != "one" {
		t.Errorf("Same.grist = %q", got)
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Home", "Same_doc2.grist")); got != "two" {
		t.Errorf("Same_doc2.grist = %q", got)
	}
}

func TestPullSanitizesNames(t *testing.T) {
	client := newPullFixture()
	client.orgs[0].Name = "My: Org/../Ltd"
	client.ws[1][0].Name = `a\b?`
	client.ws[1][0].Docs = []grist.Doc{{ID: "docX", Name: " 2026 Report ", UpdatedAt: "t1"}}
	client.contents["docX"] = "x"
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "My_ Org_.._Ltd", "a_b_", "2026 Report.grist")); err != nil {
		t.Errorf("sanitized hierarchy missing: %v", err)
	}
}

func TestPullPersonalOrgLevelStripped(t *testing.T) {
	client := newPullFixture()
	client.orgs = []grist.Org{{ID: 0, Name: "Personal"}, {ID: 1, Name: "Acme"}}
	client.ws[0] = []grist.Workspace{
		{
			ID:   20,
			Name: "Home",
			Docs: []grist.Doc{{ID: "docP", Name: "Notes", UpdatedAt: "t1"}},
		},
	}
	client.contents["docP"] = "personal"
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if got := readFile(t, filepath.Join(outDir, "Home", "Notes.grist")); got != "personal" {
		t.Errorf("personal doc should sit directly under output: %q", got)
	}
	if _, err := os.Stat(filepath.Join(outDir, "Personal")); err == nil {
		t.Errorf("Personal directory should not be created")
	}
}

func TestPullDeletedLocalFileIsRestored(t *testing.T) {
	client := newPullFixture()
	outDir := t.TempDir()
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	if err := os.Remove(filepath.Join(outDir, "Acme", "Home", "Alpha.grist")); err != nil {
		t.Fatal(err)
	}
	if _, err := runPullTest(t, client, outDir, false); err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if client.downloads["docAlpha"] != 2 {
		t.Errorf("deleted local file should be re-downloaded: %v", client.downloads)
	}
	if got := readFile(t, filepath.Join(outDir, "Acme", "Home", "Alpha.grist")); got != "alpha-content" {
		t.Errorf("restored content = %q", got)
	}
}
