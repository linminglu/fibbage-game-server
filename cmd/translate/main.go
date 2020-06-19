package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Final struct {
	Category           string   `json:"category,omitempty"`
	Question           string   `json:"question,omitempty"`
	Answer             string   `json:"answer,omitempty"`
	AlternateSpellings []string `json:"alternateSpellings,omitempty"`
	Suggestions        []string `json:"suggestions,omitempty"`
}
type Normal struct {
	Category           string   `json:"category,omitempty"`
	Question           string   `json:"question,omitempty"`
	Answer             string   `json:"answer,omitempty"`
	AlternateSpellings []string `json:"alternateSpellings,omitempty"`
	Suggestions        []string `json:"suggestions,omitempty"`
}
type Seed struct {
	Final  []Final  `json:"final"`
	Normal []Normal `json:"normal"`
}

func main() {
	file, err := os.Open("result_rus.txt")

	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var txtlines []string

	for scanner.Scan() {
		txtlines = append(txtlines, scanner.Text())
	}

	file.Close()
	var normals []Normal

	for _, s := range txtlines {
		arr := strings.Split(s, ",")
		normal := Normal{}
		var idx = "0"
		for _, a := range arr {
			a = strings.TrimSuffix(strings.TrimPrefix(a, " "), " ")
			switch a {
			case "1":
				fallthrough
			case "2":
				fallthrough
			case "3":
				fallthrough
			case "4":
				fallthrough
			case "5":
				idx = a
				continue
			}
			switch idx {
			case "1":
				normal.Answer = a
			case "2":
				normal.Question = a
			case "3":
				normal.Category = a
			case "4":
				normal.AlternateSpellings = append(normal.AlternateSpellings, a)
			case "5":
				normal.Suggestions = append(normal.Suggestions, a)
			}
		}
		normals = append(normals, normal)
	}

	rankingsJson, _ := json.Marshal(normals)

	_ = ioutil.WriteFile("questions_trsl.json", rankingsJson, 0644)

}
