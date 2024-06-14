package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var queryTicker *time.Ticker

func init() {
	queryTicker = time.NewTicker(3 * time.Second)
}

func SafeQuery(req QueryRequest) (string, error) {
	var t any
	s, err := parseQueryRequest(req)
	if err != nil {
		return "", err
	}
	req_url := fmt.Sprintf("%s/query?%s", ARXIV_API, url.PathEscape(s))
	t = queryCache.Get(req_url)
	if t != nil {
		return t.(string), nil
	}
	<-queryTicker.C
	resp, err := http.Get(req_url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		result := string(bodyBytes)
		queryCache.Set(req_url, result)
		return result, nil
	}
	return "", err
}

func SafeDownloadSource(id, outfile string) error {
	var err error
	var resp *http.Response
	var body []byte

	idx := strings.LastIndex(outfile, "/")
	if idx != -1 {
		err = os.MkdirAll(outfile[0:idx], 0755)
		if err != nil {
			return err
		}
	}
	<-queryTicker.C
	resp, err = http.Get(fmt.Sprintf("https://arxiv.org/src/%s", id))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = os.WriteFile(outfile, body, 0644)
		if err != nil {
			os.Remove(outfile)
			return err
		}
	} else {
		return errors.New(fmt.Sprintf("Status not ok for ID:%s Code:%d", id, resp.StatusCode))
	}
	return nil
}
func SafeDownloadPDF(id, outfile string) error {
	var err error
	var resp *http.Response
	var body []byte

	idx := strings.LastIndex(outfile, "/")
	if idx != -1 {
		err = os.MkdirAll(outfile[0:idx], 0755)
		if err != nil {
			return err
		}
	}
	<-queryTicker.C
	resp, err = http.Get(fmt.Sprintf("https://arxiv.org/pdf/%s", id))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = os.WriteFile(outfile, body, 0644)
		if err != nil {
			os.Remove(outfile)
			return err
		}
	} else {
		return errors.New(fmt.Sprintf("Status not ok for ID:%s Code:%d", id, resp.StatusCode))
	}
	return nil
}

func SafeTuiDownloadSource(id, outfile string, netchan chan NetData) error {
	var err error
	var resp *http.Response
	var body []byte

	idx := strings.LastIndex(outfile, "/")
	if idx != -1 {
		err = os.MkdirAll(outfile[0:idx], 0755)
		if err != nil {
			return err
		}
	}
	<-queryTicker.C
	resp, err = http.Get(fmt.Sprintf("https://arxiv.org/src/%s", id))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		netchan <- NetData{
			Message: fmt.Sprintf("Binary Data for %s: %s", outfile, id),
			Size:    len(body),
		}
		err = os.WriteFile(outfile, body, 0644)
		if err != nil {
			os.Remove(outfile)
			return err
		}
	} else {
		return errors.New(fmt.Sprintf("Status not ok for ID:%s Code:%d", id, resp.StatusCode))
	}
	return nil
}
func SafeTuiDownloadPDF(id, outfile string, netchan chan NetData) error {
	var err error
	var resp *http.Response
	var body []byte

	idx := strings.LastIndex(outfile, "/")
	if idx != -1 {
		err = os.MkdirAll(outfile[0:idx], 0755)
		if err != nil {
			return err
		}
	}
	<-queryTicker.C
	resp, err = http.Get(fmt.Sprintf("https://arxiv.org/pdf/%s", id))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		netchan <- NetData{
			Message: fmt.Sprintf("Binary Data for %s: %s", outfile, id),
			Size:    len(body),
		}
		err = os.WriteFile(outfile, body, 0644)
		if err != nil {
			os.Remove(outfile)
			return err
		}
	} else {
		return errors.New(fmt.Sprintf("Status not ok for ID:%s Code:%d", id, resp.StatusCode))
	}
	return nil
}
