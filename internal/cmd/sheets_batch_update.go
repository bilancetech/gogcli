package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
)

// SheetsBatchUpdateCmd sends an arbitrary list of Sheets API batchUpdate
// requests (deleteDimension, duplicateSheet, updateSheetProperties, ...) and
// prints the raw response.
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
		"requests":       requests,
	}
	if err := dryRunAndConfirmDestructive(ctx, flags, "sheets.batch-update", payload, fmt.Sprintf("apply %d batchUpdate request(s) to spreadsheet %s", len(requests), spreadsheetID)); err != nil {
		return err
	}

	_, svc, err := requireSheetsService(ctx, flags)
	if err != nil {
		return err
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}).Context(ctx).Do()
	if err != nil {
		return err
	}
	return outfmt.WriteRaw(ctx, os.Stdout, resp, outfmt.RawOptions{})
}
