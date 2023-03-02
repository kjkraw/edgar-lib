package edgar

import (
	"fmt"
	"github.com/beevik/etree"
	"io"
	"net/http"
	"strings"
)

// getInstanceUrl returns the URI of the XBRL Instance file for a given filing.
// The function requests and processes a filing summary to find the name of the instance.
// If the function returns a RequestError with NotFound status, this may mean that the filing does not contain XBRL.
func getInstanceUrl(cik int, accessionNumber string, client *Client) (url string, err error) {
	baseUrl := fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/%010d/%s/", cik, strings.Replace(accessionNumber, "-", "", 2))

	res, err := client.Get(baseUrl + "FilingSummary.xml")
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = &RequestError{StatusCode: res.StatusCode}
		return
	}

	filingSum := etree.NewDocument()
	if _, err = filingSum.ReadFrom(res.Body); err != nil {
		return
	}

	// Best case, the instance is defined as an attribute on the report elements
	instance := filingSum.Root().FindElement("//MyReports").SelectElements("Report")[0].SelectAttr("instance")
	if instance != nil {
		instVal := instance.Value
		if strings.HasSuffix(instVal, ".htm") {
			url = baseUrl + fmt.Sprintf("%s_htm.xml", strings.TrimSuffix(instVal, ".htm"))
			return
		}
	}

	// Worst case, we use the input files
	inputs := filingSum.Root().FindElement("//InputFiles").SelectElements("File")
	for _, f := range inputs {
		fn := f.Text()
		if strings.HasSuffix(fn, ".xml") {
			switch fn[len(fn)-8 : len(fn)-4] {
			case "_cal", "_def", "_lab", "_pre":
				continue
			default:
				url = baseUrl + fn
				break
			}
		}
	}

	return
}

// GetReport copies the XBRL instance file for a given filing into w.
// The content of w can then be processed using the xbrl package.
// This function cannot download reports that do not have an XBRL instance file.
func GetReport(w io.Writer, cik int, accessionNumber string, client *Client) (err error) {
	reportUrl, err := getInstanceUrl(cik, accessionNumber, client)
	if err != nil {
		return
	}

	res, err := client.Get(reportUrl)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = &RequestError{StatusCode: res.StatusCode}
		return
	}

	_, err = io.Copy(w, res.Body)
	return
}
