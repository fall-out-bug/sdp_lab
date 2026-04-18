package python

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type FrameworkDetector struct {
	filePath string
}

func NewFrameworkDetector(filePath string) *FrameworkDetector {
	return &FrameworkDetector{filePath: filePath}
}

func (fd *FrameworkDetector) DetectFrameworks() ([]FrameworkRecord, error) {
	f, err := os.Open(fd.filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var frameworks []FrameworkRecord
	scanner := bufio.NewScanner(f)
	inTripleQuote := false
	tripleChar := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if inTripleQuote {
			if strings.Contains(trimmed, tripleChar) {
				inTripleQuote = false
			}
			continue
		}

		if countAndToggleTriple(trimmed, &inTripleQuote, &tripleChar) {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		detectFrameworksFromLine(trimmed, &frameworks)
	}

	return frameworks, scanner.Err()
}

func DetectFlask(filePath string) (bool, float64) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, 0
	}
	defer f.Close()

	maxConfidence := 0.0
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if reFlaskApp.MatchString(trimmed) {
			if confidence := 0.95; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
		if reFlaskRoute.MatchString(trimmed) {
			if confidence := 0.90; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
		if reFlaskBlueprint.MatchString(trimmed) {
			if confidence := 0.85; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
	}

	return maxConfidence > 0, maxConfidence
}

func DetectFastAPI(filePath string) (bool, float64) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, 0
	}
	defer f.Close()

	maxConfidence := 0.0
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if reFastAPIApp.MatchString(trimmed) {
			if confidence := 0.95; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
		if reFastAPIDecor.MatchString(trimmed) {
			if confidence := 0.90; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
		if reFastAPIRouter.MatchString(trimmed) {
			if confidence := 0.85; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
	}

	return maxConfidence > 0, maxConfidence
}

func DetectDjango(filePath string) (bool, float64) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, 0
	}
	defer f.Close()

	maxConfidence := 0.0
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if reDjangoApps.MatchString(trimmed) {
			if confidence := 0.95; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
		if reDjangoURLs.MatchString(trimmed) {
			if confidence := 0.90; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
		if reDjangoConfig.MatchString(trimmed) {
			if confidence := 0.90; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
		if reDjangoModel.MatchString(trimmed) {
			if confidence := 0.85; confidence > maxConfidence {
				maxConfidence = confidence
			}
		}
	}

	return maxConfidence > 0, maxConfidence
}

func DetectCelery(filePath string) (bool, float64) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, 0
	}
	defer f.Close()

	found := false
	confidence := 0.0
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if reCeleryApp.MatchString(trimmed) {
			found = true
			confidence = 0.90
			break
		}
	}

	return found, confidence
}

func IsAsyncFunction(line string) bool {
	return reAsyncFunc.MatchString(strings.TrimSpace(line))
}

func HasPydanticModel(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if rePydanticModel.MatchString(trimmed) {
			return true
		}
	}
	return false
}

var (
	reAsyncFunc    = regexp.MustCompile(`^async\s+def\s+\w+`)
	rePydanticModel = regexp.MustCompile(`class\s+\w+\s*\(\s*(?:pydantic\.)?BaseModel\s*\)`)
)
