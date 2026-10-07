package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func newSheetsBatchUpdateTestServer(t *testing.T, gotBody *map[string]any, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/spreadsheets/s1:batchUpdate") {
			http.NotFound(w, r)
			return
		}
		hits.Add(1)
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, gotBody); err != nil {
			t.Errorf("request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"spreadsheetId": "s1",
			"replies": []map[string]any{
				{},
				{"duplicateSheet": map[string]any{"properties": map[string]any{"sheetId": 99, "title": "Copy"}}},
				{},
			},
		})
	}))
}

func installMockSheetsBatchUpdate(t *testing.T, srv *httptest.Server) {
	t.Helper()
	origURL, origClient := sheetsBatchUpdateBaseURL, newSheetsHTTPClient
	t.Cleanup(func() { sheetsBatchUpdateBaseURL, newSheetsHTTPClient = origURL, origClient })
	sheetsBatchUpdateBaseURL = srv.URL + "/v4/spreadsheets/"
	newSheetsHTTPClient = func(context.Context, string) (*http.Client, error) { return srv.Client(), nil }
}

const sheetsBatchUpdateRequests = `[
  {"deleteDimension": {"range": {"sheetId": 7, "dimension": "COLUMNS", "startIndex": 4, "endIndex": 91}}},
  {"duplicateSheet": {"sourceSheetId": 7, "insertSheetIndex": 1, "newSheetName": "Copy"}},
  {"updateSheetProperties": {"properties": {"sheetId": 7, "index": 2}, "fields": "index"}}
]`

func TestSheetsBatchUpdate_SendsRequestsAndPrintsResponse(t *testing.T) {
	var got map[string]any
	var hits atomic.Int32
	srv := newSheetsBatchUpdateTestServer(t, &got, &hits)
	defer srv.Close()
	installMockSheetsBatchUpdate(t, srv)

	flags := &RootFlags{Account: "a@b.com", Force: true}
	out := captureStdout(t, func() {
		if err := runKong(t, &SheetsBatchUpdateCmd{}, []string{"s1", "--requests-json", sheetsBatchUpdateRequests}, rawTestContext(t), flags); err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	var want map[string]any
	if err := json.Unmarshal([]byte(`{"requests":`+sheetsBatchUpdateRequests+`}`), &want); err != nil {
		t.Fatalf("want: %v", err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("request body mismatch\n got: %s\nwant: %s", gotJSON, wantJSON)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if !strings.Contains(out, `"sheetId":99`) {
		t.Fatalf("expected duplicateSheet reply in output, got: %s", out)
	}
}

func TestSheetsBatchUpdate_RejectsUnknownFields(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com", Force: true}
	err := runKong(t, &SheetsBatchUpdateCmd{}, []string{"s1", "--requests-json", `[{"deleteDimenson": {}}]`}, rawTestContext(t), flags)
	if err == nil || !strings.Contains(err.Error(), "deleteDimenson") {
		t.Fatalf("expected unknown field error, got: %v", err)
	}
}

func TestSheetsBatchUpdate_RejectsEmptyList(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com", Force: true}
	if err := runKong(t, &SheetsBatchUpdateCmd{}, []string{"s1", "--requests-json", `[]`}, rawTestContext(t), flags); err == nil {
		t.Fatalf("expected error on empty request list")
	}
}

func TestSheetsBatchUpdate_DryRunAvoidsMutation(t *testing.T) {
	var got map[string]any
	var hits atomic.Int32
	srv := newSheetsBatchUpdateTestServer(t, &got, &hits)
	defer srv.Close()
	installMockSheetsBatchUpdate(t, srv)

	flags := &RootFlags{Account: "a@b.com", DryRun: true}
	_ = captureStdout(t, func() {
		_ = runKong(t, &SheetsBatchUpdateCmd{}, []string{"s1", "--requests-json", sheetsBatchUpdateRequests}, rawTestContext(t), flags)
	})
	if hits.Load() != 0 {
		t.Fatalf("dry-run sent %d batchUpdate call(s)", hits.Load())
	}
}

func TestSheetsBatchUpdate_KeepsZeroValues(t *testing.T) {
	var got map[string]any
	var hits atomic.Int32
	srv := newSheetsBatchUpdateTestServer(t, &got, &hits)
	defer srv.Close()
	installMockSheetsBatchUpdate(t, srv)

	reqs := `[
  {"updateSheetProperties": {"properties": {"sheetId": 7, "index": 0}, "fields": "index"}},
  {"deleteDimension": {"range": {"sheetId": 7, "dimension": "ROWS", "startIndex": 0, "endIndex": 1}}},
  {"updateCells": {"start": {"sheetId": 7, "rowIndex": 0, "columnIndex": 0}, "rows": [{"values": [{"userEnteredValue": {"numberValue": 0}}]}], "fields": "userEnteredValue"}}
]`
	flags := &RootFlags{Account: "a@b.com", Force: true}
	_ = captureStdout(t, func() {
		if err := runKong(t, &SheetsBatchUpdateCmd{}, []string{"s1", "--requests-json", reqs}, rawTestContext(t), flags); err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	body, _ := json.Marshal(got)
	for _, want := range []string{`"index":0`, `"startIndex":0`, `"numberValue":0`, `"rowIndex":0`, `"columnIndex":0`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("outgoing body lost %s: %s", want, body)
		}
	}
}
