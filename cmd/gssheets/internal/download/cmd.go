package download

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dictav/go-gssheets/cmd/gssheets/internal/auth"

	"google.golang.org/api/sheets/v4"
)

var (
	flagSet      = flag.NewFlagSet("load", flag.ExitOnError)
	credential   = flagSet.String("credential", "client-credential.json", "Google OAuth Client Credential")
	output       = flagSet.String("out", "out.csv", "output filename")
	sheetID      = flagSet.String("sheet", "", "Google Spread Sheets ID")
	showProperty = flagSet.Bool("property", true, "show property")

	flagValidate = func() error {
		if len(*credential) == 0 {
			return fmt.Errorf("credential is required")
		}

		if len(*output) == 0 {
			return fmt.Errorf("out is required")
		}

		if _, err := os.Stat(*output); err == nil {
			return fmt.Errorf("%s already exists", *output)
		}

		if len(*sheetID) == 0 {
			return fmt.Errorf("sheet is required")
		}

		return nil
	}
)

// subcommand interface
var (
	Name        = "download"
	Description = "download from Google Spread Sheets"
	Usage       = flagSet.PrintDefaults
)

var valueToString func(interface{}) string

func defaultValueToString(v interface{}) string {
	switch v.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	default:
		fmt.Printf("value to string: %#v\n", v)
		return ""
	}
}

func init() {
	valueToString = defaultValueToString
}

// Run command
func Run(args []string) (err error) {
	if err = flagSet.Parse(args); err != nil {
		return err
	}

	if err = flagValidate(); err != nil {
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

	// TODO: download some sheets
	sh, err := getSheet(srv, *sheetID, *showProperty)
	if err != nil {
		return err
	}

	print("downloading sheet ", sh.Properties.Title, "...")
	rng := rangeFromSheet(sh)
	res, err := srv.Spreadsheets.Values.Get(*sheetID, rng).Do()
	if err != nil {
		return err
	}
	println(" done")

	if len(res.Values) <= 1 {
		return fmt.Errorf("the sheet is empty")
	}

	print("writing data...")
	f, err := os.OpenFile(*output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if e := f.Close(); e != nil && err == nil {
			err = e
		}
	}()

	for _, row := range res.Values {
		values := make([]string, len(row))
		for i, v := range row {
			values[i] = valueToString(v)
		}

		if _, err = fmt.Fprintln(f, strings.Join(values, ",")); err != nil {
			return err
		}
	}
	println(" done")

	return nil
}

func rangeFromSheet(sh *sheets.Sheet) string {
	cols := sh.Properties.GridProperties.ColumnCount
	end := '@' + cols
	return fmt.Sprintf("%s!A:%s", sh.Properties.Title, string(end))
}

func getSheet(srv *sheets.Service, sheetID string, showProperty bool) (*sheets.Sheet, error) {
	print("getting sheet ", sheetID, "...")
	ss, err := srv.Spreadsheets.Get(sheetID).Do()
	if err != nil {
		return nil, err
	}
	if len(ss.Sheets) == 0 {
		return nil, fmt.Errorf("spreadsheet %s is empty", sheetID)
	}

	sh := ss.Sheets[0]
	str := "done"
	if showProperty {
		data, err := sh.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf := make([]byte, len(data)*2)
		dst := bytes.NewBuffer(buf)
		if err = json.Indent(dst, data, "", "  "); err != nil {
			return nil, err
		}

		str = dst.String()
	}
	println(str)

	return sh, nil
}
