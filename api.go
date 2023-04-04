package edgar

import (
	"fmt"
	"io"
)

type Taxonomy string

const (
	TaxonomyGaap = "us-gaap"
	TaxonomyDei  = "dei"
	TaxonomyIfrs = "ifrs-full"
	TaxonomySrt  = "srt"
)

func getAndCopy(w io.Writer, client *Client, url string) error {
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		err = &RequestError{StatusCode: res.StatusCode}
	}

	_, err = io.Copy(w, res.Body)
	return err
}

// GetConcept accesses the Company Concept endpoint. The resulting json file is written to w.
// Returns RequestError on non-200 status.
func GetConcept(w io.Writer, client *Client, cik int, taxonomy Taxonomy, concept string) error {
	url := fmt.Sprintf("https://data.sec.gov/api/xbrl/companyconcept/CIK%010d/%s/%s.json", cik, taxonomy, concept)
	return getAndCopy(w, client, url)
}

// GetFacts accesses the Company Facts endpoint. The resulting json file is written to w.
// Returns RequestError on non-200 status.
func GetFacts(w io.Writer, client *Client, cik int) error {
	url := fmt.Sprintf("https://data.sec.gov/api/xbrl/companyfacts/CIK%010d.json", cik)
	return getAndCopy(w, client, url)
}

type Period struct {
	Year    int
	Quarter int
	Instant bool
}

func (p Period) String() string {
	switch {
	case p.Instant:
		return fmt.Sprintf("CY%dQ%dI", p.Year, p.Quarter)
	case p.Quarter > 0 && p.Quarter <= 4:
		return fmt.Sprintf("CY%dQ%d", p.Year, p.Quarter)
	default:
		return fmt.Sprintf("CY%d", p.Year)
	}
}

// GetFrame accesses the Frames endpoint. The resulting json file is written to w.
// Returns RequestError on non-200 status.
func GetFrame(w io.Writer, client *Client, taxonomy Taxonomy, concept string, units string, period Period) error {
	url := fmt.Sprintf("https://data.sec.gov/api/xbrl/frames/%s/%s/%s/%s.json", taxonomy, concept, units, period)
	return getAndCopy(w, client, url)
}
