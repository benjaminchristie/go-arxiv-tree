package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"github.com/benjaminchristie/go-arxiv-tree/internal/cache"
)

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
