package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gocarina/gocsv"
)

func main() {
	addr := flag.String("addr", ":8080", "address to bind")
	TemplateRoot := flag.String("template-root", "/home/nii236/git/deck", "set the template root")
	DeckRoot := flag.String("deck-root", "/home/nii236/japanese/output", "set the deck root")
	AudioRoot := flag.String("audio-root", "/home/nii236/japanese/output/Kimi_no_na_wa.media", "set the audio root")
	ImageRoot := flag.String("image-root", "/home/nii236/japanese/output/Kimi_no_na_wa.media", "set the image root")
	flag.Parse()

	c := &Controller{
		TemplateRoot: *TemplateRoot,
		DeckRoot:     *DeckRoot,
		AudioRoot:    *AudioRoot,
		ImageRoot:    *ImageRoot,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/audio/{filename}", c.audioHandler)
	r.Get("/image/{filename}", c.imageHandler)
	r.Get("/view/{card}", c.viewHandler)
	log.Fatalln(http.ListenAndServe(*addr, r))
}

// Controller holds handlers and init state
type Controller struct {
	TemplateRoot string
	DeckRoot     string
	AudioRoot    string
	ImageRoot    string
}

// Record shape of the deck CSV
type Record struct {
	Tag        string
	Sequence   string
	AudioPath  string
	ImagePath  string
	Expression string
	Meaning    string
}

func loadRecord(deckPath string, index int) (*Record, int, error) {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.FieldsPerRecord = 6
		r.LazyQuotes = true
		r.Comma = '\t'
		return r
	})

	deckFile, err := os.OpenFile(filepath.Join(deckPath, "Kimi_no_na_wa.tsv"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer deckFile.Close()
	records := []*Record{}
	err = gocsv.UnmarshalWithoutHeaders(deckFile, &records)
	if err != nil {
		return nil, 0, fmt.Errorf("unmarshal file: %w", err)
	}
	if index > len(records)-1 {
		return nil, 0, fmt.Errorf("record not found")
	}

	return records[index], len(records), nil

}

type Data struct {
	Index         int
	Total         int
	Tag           string
	Sequence      string
	AudioFilename string
	ImageFilename string
	Expression    string
	Meaning       string
}

func extractAudioName(in string) (string, error) {
	r := regexp.MustCompile(`sound:(.*.mp3)`)
	result := r.FindStringSubmatch(in)
	if len(result) != 2 {
		return "", fmt.Errorf("regex: bad error length %d", len(result))
	}
	return result[1], nil
}
func extractImageName(in string) (string, error) {
	r := regexp.MustCompile(`"(.*)"`)
	result := r.FindStringSubmatch(in)
	if len(result) != 2 {
		return "", fmt.Errorf("regex: bad error length %d", len(result))
	}
	return result[1], nil
}
func (c *Controller) audioHandler(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	imagePath := filepath.Join(c.ImageRoot, filename)
	f, err := os.Open(imagePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "audio/mpeg")
	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
func (c *Controller) imageHandler(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	imagePath := filepath.Join(c.ImageRoot, filename)
	f, err := os.Open(imagePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "image/jpeg")
	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Controller) viewHandler(w http.ResponseWriter, r *http.Request) {
	indexStr := chi.URLParam(r, "card")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	record, count, err := loadRecord(c.DeckRoot, index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	imageFilename, err := extractImageName(record.ImagePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	audioFilename, err := extractAudioName(record.AudioPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := &Data{
		Index:         index,
		Total:         count,
		Tag:           record.Tag,
		Sequence:      record.Sequence,
		AudioFilename: audioFilename,
		ImageFilename: imageFilename,
		Expression:    record.Expression,
		Meaning:       record.Meaning,
	}

	funcMap := template.FuncMap{
		// The name "inc" is what the function will be called in the template text.
		"inc": func(i int) int {
			if i == count-1 {
				return i
			}
			return i + 1
		},
		"dec": func(i int) int {
			if i == 0 {
				return 0
			}
			return i - 1
		},
	}
	ViewHTMLFile, err := os.Open(filepath.Join(c.TemplateRoot, "view.gotpl"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ViewHTMLFile.Close()
	ViewHTML, err := ioutil.ReadAll(ViewHTMLFile)
	t := template.Must(template.New("view").Funcs(funcMap).Parse(string(ViewHTML)))
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
