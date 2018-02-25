package upload

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dictav/go-gssheets/cmd/gssheets/internal/auth"

	"google.golang.org/api/sheets/v4"
)

var (
	flagSet    = flag.NewFlagSet("upload", flag.ExitOnError)
	credential = flagSet.String("credential", "client-credential.json", "Google OAuth Client Credential")
	input      = flagSet.String("in", "", "input file")
	frozen     = flag.Bool("frozen", true, "froze first column and first row")

	flagValidate = func() error {
		if len(*credential) == 0 {
			return fmt.Errorf("credential is required")
		}

		if len(*input) == 0 {
			return fmt.Errorf("in is required")
		}

		return nil
	}
)

type categoryFile struct {
	key  string
	file string
}

const (
	defaultSheetName = "Data"
)

// subcommand interface
var (
	Name        = "upload"
	Description = "upload dict/ to Google Spread Sheets"
	Usage       = flagSet.PrintDefaults
)

// Run command
func Run(args []string) error {
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if err := flagValidate(); err != nil {
		return err
	}

	ctx := context.Background()

	config, err := auth.ConfigFromJSON(*credential)
	if err != nil {
		return err
	}

	tk, err := auth.TokenFromCache()
	if err != nil {
		return err
	}

	client := config.Client(ctx, tk)
	srv, err := sheets.New(client)
	if err != nil {
		return err
	}

	r, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer func() {
		if e := r.Close(); e != nil && err == nil {
			err = e
		}
	}()

	s := bufio.NewScanner(r)

	// count number of lines
	rlen := 0
	for s.Scan() {
		rlen++
	}

	if s.Err() != nil {
		return s.Err()
	}

	r.Seek(0, 0)
	s = bufio.NewScanner(r)

	rows := make([][]interface{}, rlen)
	cols := 0
	n := 0
	for s.Scan() {
		line := s.Text()
		println("line", line)
		if len(line) == 0 {
			return fmt.Errorf("has empty line")
		}

		columns := strings.Split(line, ",")
		if cols == 0 {
			cols = len(columns)
		} else if cols != len(columns) {
			return fmt.Errorf("invalid number of columns: %s", line)
		}

		cells := make([]interface{}, len(columns))
		for i, v := range columns {
			cells[i] = v
		}
		rows[n] = cells
		n++
	}

	if s.Err() != nil {
		return s.Err()
	}

	println("read", len(rows), "rows")

	vRange := &sheets.ValueRange{
		MajorDimension: "ROWS",
		Values:         rows,
	}

	rng, err := rangeFromRows(defaultSheetName, rows)
	if err != nil {
		fmt.Printf("rows: %v\n", rows)
		return err
	}

	sh := &sheets.Sheet{
		Properties: &sheets.SheetProperties{
			Title: defaultSheetName,
		},
	}

	ss := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: "test-title",
		},
		Sheets: []*sheets.Sheet{sh},
	}

	print("creating new spread sheet...")
	cres, err := srv.Spreadsheets.Create(ss).Do()
	if err != nil {
		return err
	}
	println("done")

	sheetID := cres.SpreadsheetId
	call := srv.Spreadsheets.Values.Update(sheetID, rng, vRange).ValueInputOption("RAW")
	ures, err := call.Do()
	if err != nil {
		return err
	}

	println("saved URL:", cres.SpreadsheetUrl, "status:", ures.HTTPStatusCode)

	return nil
}

func rangeFromRows(key string, rows [][]interface{}) (string, error) {
	if len(rows) == 0 {
		return "", fmt.Errorf("at least one row is required")
	}
	cols := rows[0]
	if len(cols) == 0 {
		return "", fmt.Errorf("at least one column is required")
	}
	end := '@' + len(cols)
	return fmt.Sprintf("%s!A:%s", key, string(end)), nil
}
