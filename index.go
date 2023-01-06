package edgar

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Index represents an index file downloaded from EDGAR.
type Index struct {
	file    *os.File
	Entries []IndexEntry
}

// IndexEntry represents report found in the index.
type IndexEntry struct {
	FormType        string
	CompanyName     string
	CIK             [10]byte
	DateFiled       time.Time
	AccessionNumber [20]byte
}

type EnumQuarter int

const (
	FirstQuarter EnumQuarter = iota
	SecondQuarter
	ThirdQuarter
	FourthQuarter
)

func (q EnumQuarter) String() string {
	switch q {
	case FirstQuarter:
		return "QTR1"
	case SecondQuarter:
		return "QTR2"
	case ThirdQuarter:
		return "QTR3"
	case FourthQuarter:
		return "QTR4"
	default:
		return ""
	}
}

var AllQuarters = []EnumQuarter{FirstQuarter, SecondQuarter, ThirdQuarter, FourthQuarter}

type IndexOpts struct {
	Year    int
	Quarter EnumQuarter
	Current bool
}

// DownloadIndex downloads an index based on opts and writes the file to f.
// If the "Current" option is true, the index for the current quarter is downloaded.
// Otherwise, the Year and Quarter options are used.
func DownloadIndex(opts IndexOpts, client *Client, f *os.File) (idx *Index, err error) {
	var idxUrl string
	if opts.Current {
		idxUrl = "https://www.sec.gov/Archives/edgar/full-index/form.idx"
	} else {
		idxUrl = fmt.Sprintf("https://www.sec.gov/Archives/edgar/full-index/%d/%s/form.idx", opts.Year, opts.Quarter)
	}

	res, err := client.Get(idxUrl)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = &RequestError{StatusCode: res.StatusCode}
		return
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		fmt.Println(res.Body)
		return
	}
	if err = f.Sync(); err != nil {
		return
	}
	_, _ = f.Seek(0, io.SeekStart)

	idx = &Index{file: f}
	return
}

// Process reads the index file and generates a list of 10-K and 10-Q entries.
// The file is not closed after processing!
// It is the user's responsibility to close the file after processing is complete!
func (idx *Index) Process() {
	var counter int
	scanner := bufio.NewScanner(idx.file)
	for scanner.Scan() {
		if counter < 10 { // Skip the first 10 lines
			counter += 1
			continue
		}

		line := scanner.Text()
		formType := strings.TrimSpace(line[:12])

		// This filters out reports that are not 10-K or 10-Q.
		switch formType {
		case "10-K", "10-Q": // More report types could be added here. I am only interested in 10-K/Qs.
			var cik [10]byte
			copy(cik[:], fmt.Sprintf("%010s", strings.TrimSpace(line[74:86])))

			dateFiled, _ := time.Parse("2006-01-02", line[86:96])

			href := strings.TrimSpace(line[98:])
			var accessionNumber [20]byte
			copy(accessionNumber[:], href[len(href)-24:len(href)-4])

			entry := IndexEntry{
				FormType:        formType,
				CompanyName:     line[12:74],
				CIK:             cik,
				DateFiled:       dateFiled,
				AccessionNumber: accessionNumber,
			}

			idx.Entries = append(idx.Entries, entry)
		}
	}
}
