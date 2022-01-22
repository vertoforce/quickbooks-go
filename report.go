package quickbooks

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// Generated by https://quicktype.io
//
// To change quicktype's target language, run command:
//
//   "Set quicktype target language"

// Report is a quickbooks report.
// It's very visual, and not super well structured.
// It's worth using the .FormattedReport()
type Report struct {
	Header  MainHeader `json:"Header"`
	Columns Columns    `json:"Columns"`
	Rows    Rows       `json:"Rows"`
}

type MainHeader struct {
	Time        string   `json:"Time"`
	ReportName  string   `json:"ReportName"`
	ReportBasis string   `json:"ReportBasis"`
	StartPeriod string   `json:"StartPeriod"`
	EndPeriod   string   `json:"EndPeriod"`
	Currency    string   `json:"Currency"`
	Customer    string   `json:"Customer"`
	Option      []Option `json:"Option"`
}

type Columns struct {
	Column []Column `json:"Column"`
}

type Column struct {
	ColTitle string   `json:"ColTitle"`
	ColType  string   `json:"ColType"`
	MetaData []Option `json:"MetaData"`
}

type Option struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type Rows struct {
	Row []Row `json:"Row"`
}

type Row struct {
	Header  Header `json:"Header"`
	Rows    Rows   `json:"Rows"`
	Summary Header `json:"Summary"`
	Type    string `json:"type"`
	Group   string `json:"group"`
}

type Header struct {
	ColData []SummaryColDatum `json:"ColData"`
}

type SummaryColDatum struct {
	Value string `json:"value"`
}

// FormattedReport is a simpler, easier to use report compared to the quickbooks reports
type FormattedReport struct {
	Rows []FormattedRow
}
type FormattedRow struct {
	Label   string
	Value   float64
	SubRows []FormattedRow
}

func (r *Report) FormattedReport() *FormattedReport {
	return &FormattedReport{
		Rows: convertRows(r.Rows.Row),
	}
}

func (r *FormattedReport) Print() string {
	ret := &bytes.Buffer{}
	r.print(ret, "", r.Rows)
	return ret.String()
}

func (r *FormattedReport) print(report io.Writer, prefix string, rows []FormattedRow) {
	for _, row := range rows {
		if row.Label != "" {
			fmt.Fprintf(report, "%s%s - %.2f\n", prefix, row.Label, row.Value)
		}
		r.print(report, prefix+" ", row.SubRows)
	}
}

func convertRows(rows []Row) []FormattedRow {
	retRows := []FormattedRow{}
	for _, row := range rows {
		ret := FormattedRow{}
		// The first col is usually the label, then try and find the number
		// in one of the remaining columns
		if len(row.Summary.ColData) > 0 {
			ret.Label = row.Summary.ColData[0].Value
			// Find number
			for _, col := range row.Summary.ColData[1:] {
				float, err := strconv.ParseFloat(col.Value, 64)
				if err == nil {
					ret.Value = float
					break
				}
			}
		}
		ret.SubRows = convertRows(row.Rows.Row)
		retRows = append(retRows, ret)
	}

	return retRows
}
