package api

import (
	"archive/tar"
	"compress/gzip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/benjaminchristie/go-arxiv-tree/cache"
	"github.com/jschaf/bibtex"
	"github.com/jschaf/bibtex/ast"
)

type Author struct {
	Name        string `xml:"name"`
	Affiliation string `xml:"arxiv:affiliation"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

type Cat struct {
	V string `xml:"term,attr"`
}

type Entry struct {
	Title     string   `xml:"title"`
	ID        string   `xml:"id"`
	Links     []Link   `xml:"link"`
	Updated   string   `xml:"updated"`
	Published string   `xml:"published"`
	Summary   string   `xml:"summary"`
	Author    []Author `xml:"author"`
	Category  Cat      `xml:"category"`
}

type Host struct {
	Entries []Entry `xml:"entry"`
}

// see https://info.arxiv.org/help/api/user-manual.html
type QueryRequest struct {
	Author     string
	Title      string
	IDList     string // comma delimited string of IDs
	Start      int    // start idx (default 0)
	MaxResults int    // max results (default 10)
	Cat        string // category to search
}

var tarExtractRegexpHelper *regexp.Regexp

const ARXIV_API = "https://export.arxiv.org/api"

var queryCache *cache.Cache

var biber *bibtex.Biber

func init() {
	var err error
	tarExtractRegexpHelper, err = regexp.Compile("([a-zA-Z0-9.]+).tar.gz")
	if err != nil {
		panic(err)
	}
	biber = &bibtex.Biber{}
	queryCache = &cache.Cache{}
}

func ParseXML(s string) []Entry {
	host := Host{}
	err := xml.Unmarshal([]byte(s), &host)
	if err != nil {
		log.Printf("Error unmarshalling field %v", err)
	}
	return host.Entries
}

func ReadBibtexFile(filename string) ([]bibtex.Entry, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	astfptr, err := biber.Parse(f)
	if err != nil {
		return nil, err
	}
	entries, err := biber.Resolve(astfptr)
	return entries, err
}

func QueryBibtexEntry(b bibtex.Entry) (string, string, error) {
	author, ok := b.Tags[bibtex.FieldAuthor].(*ast.UnparsedText)
	if !ok {
		return "", "", errors.New("Conversion error in QueryBibtexEntry")
	}
	title, ok := b.Tags[bibtex.FieldTitle].(*ast.UnparsedText)
	if !ok {
		return "", "", errors.New("Conversion error in QueryBibtexEntry")
	}
	a := author.Value
	t := title.Value
	if a == "" || t == "" {
		return "", "", errors.New("Conversion error in QueryBibtexEntry")
	}
	a = strings.Replace(a, "{", "", -1)
	a = strings.Replace(a, "}", "", -1)
	t = strings.Replace(t, "{", "", -1)
	t = strings.Replace(t, "}", "", -1)
	return a, t, nil
}

func parseQueryRequest(q QueryRequest) (string, error) {
	s := ""
	use_amp := false
	cachedValue := queryCache.Get(q)
	if cachedValue != nil {
		return cachedValue.(string), nil
	}
	if q.Title != "" {
		s += fmt.Sprintf("search_query=ti:%s", q.Title)
		use_amp = true
	} else if q.Author != "" {
		s += fmt.Sprintf("search_query=au:%s", q.Author)
		use_amp = true
	}
	if q.IDList != "" {
		if use_amp {
			s += "&"
		} else {
			use_amp = true
		}
		s += fmt.Sprintf("id_list=%s", q.IDList)
	}
	if q.Start != 0 {
		if use_amp {
			s += "&"
		} else {
			use_amp = true
		}
		s += fmt.Sprintf("start=%d", q.Start)
	}
	if q.MaxResults != 0 {
		if use_amp {
			s += "&"
		} else {
			use_amp = true
		}
		s += fmt.Sprintf("max_results=%d", q.MaxResults)
	}
	if q.Cat != "" {
		if use_amp {
			s += "&"
		} else {
			use_amp = true
		}
		s += fmt.Sprintf("cat=%s", q.Cat)
	}
	if use_amp {
		queryCache.Set(q, s)
		return s, nil
	} else {
		return "", errors.New(fmt.Sprintf("Error parsing QueryRequest: %v", q))
	}
}

func Query(req QueryRequest) (string, error) {
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

func ExtractTargz(infile, outdir string) error {
	var err error
	var r *os.File
	var gzipStream *gzip.Reader
	var tarStream *tar.Reader
	var header *tar.Header

	r, err = os.Open(infile)
	if err != nil {
		return err
	}
	gzipStream, err = gzip.NewReader(r)
	if err != nil {
		return err
	}
	tarStream = tar.NewReader(gzipStream)

	for {
		header, err = tarStream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(fmt.Sprintf("%s/%s", outdir, header.Name), 0755)
			if err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := os.Create(fmt.Sprintf("%s/%s", outdir, header.Name))
			defer file.Close()
			if err != nil {
				return err
			}
			_, err = io.Copy(file, tarStream)
			if err != nil {
				return err
			}
		default:
			return errors.New(fmt.Sprintf("Unknown type in extractTargz: %v in %s", header.Typeflag, header.Name))
		}
	}
	return nil
}

// downloads tar.gz formatted source code
func DownloadSource(id, outfile string) error {
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

func DownloadPDF(id, outfile string) error {
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
