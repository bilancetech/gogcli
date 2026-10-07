package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	gapi "google.golang.org/api/googleapi"
	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/googleauth"
)

// sheetsBatchUpdateBaseURL is swapped in tests.
var sheetsBatchUpdateBaseURL = "https://sheets.googleapis.com/v4/spreadsheets/"

// newSheetsHTTPClient is swapped in tests to avoid real auth.
var newSheetsHTTPClient = func(ctx context.Context, email string) (*http.Client, error) {
	return googleapi.NewHTTPClient(ctx, googleauth.ServiceSheets, email)
}

// SheetsBatchUpdateCmd sends an arbitrary list of Sheets API batchUpdate
// requests (deleteDimension, duplicateSheet, updateSheetProperties, ...) and
// prints the raw response.
//
// The requests JSON is decoded into the Sheets Go types only to validate it;
// the original bytes are what gets sent. Round-tripping through the Go types
// would drop zero values (index 0, numberValue 0, startIndex 0) because of
// omitempty.
//
// REST reference: https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets/batchUpdate
type SheetsBatchUpdateCmd struct {
	SpreadsheetID string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	RequestsJSON  string `name:"requests-json" required:"" help:"JSON array of batchUpdate Request objects (inline, @file, or - for stdin)"`
}

func (c *SheetsBatchUpdateCmd) Run(ctx context.Context, flags *RootFlags) error {
	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	if spreadsheetID == "" {
		return usage("empty spreadsheetId")
	}

	b, err := resolveInlineOrFileBytes(c.RequestsJSON)
	if err != nil {
		return fmt.Errorf("read --requests-json: %w", err)
	}
	if len(b) == 0 {
		return usage("empty --requests-json")
	}

	// Reject unknown fields so a typo fails loudly instead of sending an empty request.
	var requests []*sheets.Request
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&requests); err != nil {
		return fmt.Errorf("invalid --requests-json: %w", err)
	}
	if len(requests) == 0 {
		return usage("--requests-json contains no requests")
	}

	payload := map[string]any{
		"spreadsheet_id": spreadsheetID,
		"requests":       json.RawMessage(b),
	}
	if err := dryRunAndConfirmDestructive(ctx, flags, "sheets.batch-update", payload, fmt.Sprintf("apply %d batchUpdate request(s) to spreadsheet %s", len(requests), spreadsheetID)); err != nil {
		return err
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	client, err := newSheetsHTTPClient(ctx, account)
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]json.RawMessage{"requests": b})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sheetsBatchUpdateBaseURL+url.PathEscape(spreadsheetID)+":batchUpdate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := gapi.CheckResponse(resp); err != nil {
		return err
	}
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, out); err != nil {
		return fmt.Errorf("decode batchUpdate response: %w", err)
	}
	compact.WriteByte('\n')
	_, err = os.Stdout.Write(compact.Bytes())
	return err
}
