package api

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/benjaminchristie/go-arxiv-tree/internal/cache"
)

var tarExtractRegexpHelper *regexp.Regexp

// see https://info.arxiv.org/help/api/user-manual.html
type QueryRequest struct {
	SearchQuery string // search query
	IDList      string // comma delimited string of IDs
	Start       int    // start idx (default 0)
	MaxResults  int    // max results (default 10)
	Cat         string // category to search
}

const ARXIV_API = "https://export.arxiv.org/api"

var queryCache *cache.Cache

func init() {
	var err error
	tarExtractRegexpHelper, err = regexp.Compile("([a-zA-Z0-9.]+).tar.gz")
	if err != nil {
		panic(err)
	}
	queryCache = &cache.Cache{}
}

// TODO: verify that using the cache is faster
func parseQueryRequest(q QueryRequest) (string, error) {
	s := ""
	use_amp := false
	cachedValue := queryCache.Get(q)
	if cachedValue != nil {
		return cachedValue.(string), nil
	}
	if q.SearchQuery != "" {
		s = fmt.Sprintf("search_query=%s", q.SearchQuery)
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

// TODO: verify that using the cache is faster (should be!)
func genericQuery(methodName string, parameters QueryRequest) (string, error) {
	var t any
	s, err := parseQueryRequest(parameters)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/%s?%s", ARXIV_API, methodName, s)
	log.Printf("Querying %s", url)
	t = queryCache.Get(url)
	if t != nil {
		return t.(string), nil
	}
	resp, err := http.Get(url)
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
		queryCache.Set(url, result)
		return result, nil
	}
	return "", err
}

func Query(methodName string, parameters QueryRequest) (string, error) {
	return genericQuery(methodName, parameters)
}

// assumes infile is of the form XXX/XXX.tar.gz and XXX are matching (although
// match is not required) extracts infile to %s/*
func extractTargz(infile string) error {
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

	idx := strings.LastIndex(infile, "/")
	if idx <= 0 {
		return errors.New("filename does not include directory, ignoring")
	}
	dir := infile[0:idx]

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
			err = os.MkdirAll(fmt.Sprintf("%s/%s", dir, header.Name), 0755)
			if err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := os.Create(fmt.Sprintf("%s/%s", dir, header.Name))
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

// performs download and extraction of remote arxiv
// source to client. Extracted files are in ./id/*
func DownloadSource(id string) (string, error) {
	var err error
	var resp *http.Response
	var body []byte

	res := tarExtractRegexpHelper.FindStringSubmatch(id)
	if len(res) != 2 {
		return "", errors.New(fmt.Sprintf("Unable to find match for regexp in DownloadSource for %s", id))
	}
	s := res[1]
	err = os.MkdirAll(s, 0755)
	if err != nil {
		return "", err
	}
	resp, err = http.Get(fmt.Sprintf("https://arxiv.org/src/%s", s))
	if err != nil {
		return "", err
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	archive := fmt.Sprintf("%s/%s.tar.gz", s, s)
	err = os.WriteFile(archive, body, 0644)
	if err != nil {
		return "", err
	}
	err = extractTargz(archive)
	if err != nil {
		return "", err
	}
	return archive, nil
}
