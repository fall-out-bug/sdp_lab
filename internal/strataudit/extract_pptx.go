package strataudit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func extractPPTXFromZIP(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pptx zip: %w", err)
	}

	var slideNames []string
	var notesNames []string
	files := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		files[f.Name] = f
		switch {
		case strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml"):
			slideNames = append(slideNames, f.Name)
		case strings.HasPrefix(f.Name, "ppt/notesSlides/notesSlide") && strings.HasSuffix(f.Name, ".xml"):
			notesNames = append(notesNames, f.Name)
		}
	}

	if len(slideNames) == 0 && len(notesNames) == 0 {
		return "", fmt.Errorf("pptx: no slide xml found")
	}

	sort.Slice(slideNames, func(i, j int) bool { return slideIndex(slideNames[i]) < slideIndex(slideNames[j]) })
	sort.Slice(notesNames, func(i, j int) bool { return slideIndex(notesNames[i]) < slideIndex(notesNames[j]) })

	notesByIndex := make(map[int]string, len(notesNames))
	for _, name := range notesNames {
		text, err := extractOpenXMLText(files[name])
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		notesByIndex[slideIndex(name)] = text
	}

	var out strings.Builder
	for _, name := range slideNames {
		idx := slideIndex(name)
		slideText, err := extractOpenXMLText(files[name])
		if err != nil {
			return "", err
		}
		noteText := notesByIndex[idx]
		if strings.TrimSpace(slideText) == "" && strings.TrimSpace(noteText) == "" {
			continue
		}
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		fmt.Fprintf(&out, "[Slide %d]\n", idx)
		if strings.TrimSpace(slideText) != "" {
			out.WriteString(slideText)
		}
		if strings.TrimSpace(noteText) != "" {
			if strings.TrimSpace(slideText) != "" {
				out.WriteString("\n")
			}
			out.WriteString("[Notes]\n")
			out.WriteString(noteText)
		}
	}

	text := strings.TrimSpace(out.String())
	if text == "" {
		return "", fmt.Errorf("pptx: no readable text extracted")
	}
	return text, nil
}

func extractOpenXMLText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filepath.Base(f.Name), err)
	}
	defer rc.Close() //nolint:errcheck

	decoder := xml.NewDecoder(rc)
	var parts []string
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode %s: %w", filepath.Base(f.Name), err)
		}

		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "t" {
			continue
		}

		var value string
		if err := decoder.DecodeElement(&value, &start); err != nil {
			return "", fmt.Errorf("read text %s: %w", filepath.Base(f.Name), err)
		}
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}

	return strings.Join(parts, "\n"), nil
}

func slideIndex(name string) int {
	base := filepath.Base(name)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] < '0' || base[i] > '9' {
			n, err := strconv.Atoi(base[i+1:])
			if err != nil {
				return 0
			}
			return n
		}
	}
	n, err := strconv.Atoi(base)
	if err != nil {
		return 0
	}
	return n
}
