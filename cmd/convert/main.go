package main

import (
	"bufio"
	"cloud.google.com/go/translate"
	"context"
	"fmt"
	"github.com/jinzhu/configor"
	"github.com/prometheus/common/log"
	"golang.org/x/text/language"
	"os"
	"path/filepath"
	"strings"
)

func translateText(targetLanguage, text string) (string, error) {
	// text := "The Go Gopher is cute"
	ctx := context.Background()

	lang, err := language.Parse(targetLanguage)
	if err != nil {
		return "", fmt.Errorf("language.Parse: %v", err)
	}

	client, err := translate.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	resp, err := client.Translate(ctx, []string{text}, lang, nil)
	if err != nil {
		return "", fmt.Errorf("Translate: %v", err)
	}
	if len(resp) == 0 {
		return "", fmt.Errorf("Translate returned empty response to text: %s", text)
	}
	return resp[0].Text, nil
}

type Final struct {
	Category           string   `json:"category"`
	Question           string   `json:"question"`
	Answer             string   `json:"answer"`
	AlternateSpellings []string `json:"alternateSpellings"`
	Suggestions        []string `json:"suggestions"`
}
type Normal struct {
	Category           string   `json:"category"`
	Question           string   `json:"question"`
	Answer             string   `json:"answer"`
	AlternateSpellings []string `json:"alternateSpellings"`
	Suggestions        []string `json:"suggestions"`
}
type Seed struct {
	Final  []Final  `json:"final"`
	Normal []Normal `json:"normal"`
}

func getSeeds() Seed {
	var seeds Seed
	filepaths, _ := filepath.Glob(filepath.Join("questions.json"))
	if err := configor.Load(&seeds, filepaths...); err != nil {
		panic(err)
	}
	return seeds

}
func main() {
	seeds := getSeeds()
	normals := seeds.Normal
	file, err := os.OpenFile("result.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	datawriter := bufio.NewWriter(file)

	for _, seed := range normals {
		var trslt []string
		trslt = append(trslt, "1")
		trslt = append(trslt, seed.Answer)
		trslt = append(trslt, "2")
		trslt = append(trslt, seed.Question)
		trslt = append(trslt, "3")
		trslt = append(trslt, seed.Category)
		trslt = append(trslt, "4")
		for _, s := range seed.AlternateSpellings {
			trslt = append(trslt, s)
		}
		trslt = append(trslt, "5")
		for _, s := range seed.Suggestions {
			trslt = append(trslt, s)
		}
		_, _ = datawriter.WriteString(strings.Join(trslt, ",") + "\n")
	}

	datawriter.Flush()
	file.Close()

}
