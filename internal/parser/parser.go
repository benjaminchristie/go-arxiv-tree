package parser

import (
	"encoding/xml"
	"log"
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
	Id        string   `xml:"id"`
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

func ParseXML(s string) []Entry {
	host := Host{}
	err := xml.Unmarshal([]byte(s), &host)
	if err != nil {
		log.Printf("Error unmarshalling field %v", err)
	}
	return host.Entries
}
