package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/api/docs/v1"
)

func TestDocsAddTab(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	var batchReq docs.BatchUpdateDocumentRequest
	var rawBody string

	docSvc, cleanup := newDocsServiceForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, ":batchUpdate") {
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		rawBody = string(body)
		if err := json.Unmarshal(body, &batchReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"documentId": "doc1",
			"replies": []any{
				map[string]any{
					"addDocumentTab": map[string]any{
						"tabProperties": map[string]any{
							"tabId":       "t.new",
							"title":       "Notes",
							"index":       0,
							"parentTabId": "t.parent",
							"iconEmoji":   "📝",
						},
					},
				},
			},
		})
	}))
	defer cleanup()
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	flags := &RootFlags{Account: "a@b.com"}
	ctx, out := newDocsCmdOutputContext(t)

	if err := runKong(t, &DocsAddTabCmd{}, []string{"doc1", "Notes", "--parent-tab-id", "t.parent", "--index", "0", "--emoji", "📝"}, ctx, flags); err != nil {
		t.Fatalf("docs add-tab: %v", err)
	}

	if len(batchReq.Requests) != 1 || batchReq.Requests[0].AddDocumentTab == nil {
		t.Fatalf("unexpected request payload: %#v", batchReq.Requests)
	}
	props := batchReq.Requests[0].AddDocumentTab.TabProperties
	if props == nil {
		t.Fatalf("missing tab properties")
	}
	if props.Title != "Notes" || props.ParentTabId != "t.parent" || props.IconEmoji != "📝" || props.Index != 0 {
		t.Fatalf("unexpected tab properties: %#v", props)
	}
	if !strings.Contains(rawBody, "\"index\":0") {
		t.Fatalf("expected raw request to include index=0, got %q", rawBody)
	}

	got := out.String()
	for _, want := range []string{"documentId\tdoc1", "tabId\tt.new", "title\tNotes", "index\t0", "parentTabId\tt.parent", "emoji\t📝"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}
}

func TestDocsAddTab_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	docSvc, cleanup := newDocsServiceForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"documentId": "doc1",
			"replies": []any{
				map[string]any{
					"addDocumentTab": map[string]any{
						"tabProperties": map[string]any{
							"tabId": "t.new",
							"title": "Notes",
							"index": 2,
						},
					},
				},
			},
		})
	}))
	defer cleanup()
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	flags := &RootFlags{Account: "a@b.com"}
	out := captureStdout(t, func() {
		if err := runKong(t, &DocsAddTabCmd{}, []string{"doc1", "Notes", "--index", "2"}, newDocsJSONContext(t), flags); err != nil {
			t.Fatalf("docs add-tab json: %v", err)
		}
	})

	if !strings.Contains(out, "\"tabId\": \"t.new\"") || !strings.Contains(out, "\"index\": 2") {
		t.Fatalf("unexpected JSON output: %q", out)
	}
}
