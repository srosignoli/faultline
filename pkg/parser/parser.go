package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MetricType represents the Prometheus metric type as declared in # TYPE lines.
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
	Untyped   MetricType = "untyped"
)

var knownTypes = map[MetricType]struct{}{
	Counter:   {},
	Gauge:     {},
	Histogram: {},
	Summary:   {},
	Untyped:   {},
}

// Metric holds one Prometheus sample. Value is intentionally a plain field so
// the mutation engine can modify it with a simple assignment.
type Metric struct {
	Name   string
	Help   string
	Type   MetricType
	Labels map[string]string // nil if the sample carries no labels
	Value  float64
}

// parseState accumulates HELP and TYPE metadata encountered before samples.
type parseState struct {
	helpByName map[string]string
	typeByName map[string]MetricType
}

// ParseDump reads a Prometheus text-format exposition and returns one *Metric
// per sample line. The function is fail-fast: the first parse error is returned
// with a line number, and no partial result is given.
func ParseDump(r io.Reader) ([]*Metric, error) {
	const maxBuf = 1 << 20 // 1 MiB — guards against dumps with very long label values

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, maxBuf), maxBuf)

	state := &parseState{
		helpByName: make(map[string]string),
		typeByName: make(map[string]MetricType),
	}

	var result []*Metric
	lineNum := 0

	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			rest := strings.TrimPrefix(line, "#")
			rest = strings.TrimLeft(rest, " \t")

			keyword, tail := splitFirstSpace(rest)
			switch keyword {
			case "HELP":
				name, text := splitFirstSpace(tail)
				state.helpByName[name] = unescapeHelp(text)
			case "TYPE":
				name, typStr := splitFirstSpace(tail)
				mt := MetricType(typStr)
				if _, ok := knownTypes[mt]; !ok {
					return nil, fmt.Errorf("line %d: unknown metric type %q", lineNum, typStr)
				}
				state.typeByName[name] = mt
			// other # comments are ignored
			}
			continue
		}

		m, err := parseSampleLine(line, lineNum, state)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return result, nil
}

// parseSampleLine parses a single non-comment, non-blank line such as:
//
//	http_requests_total{method="GET",code="200"} 1027 1395066363000
func parseSampleLine(line string, lineNum int, state *parseState) (*Metric, error) {
	// Split name (plus optional labels) from value (plus optional timestamp).
	braceOpen := strings.IndexByte(line, '{')
	var namepart, rest string

	if braceOpen >= 0 {
		namepart = line[:braceOpen]
		remainder := line[braceOpen:]
		braceClose := strings.IndexByte(remainder, '}')
		if braceClose < 0 {
			return nil, fmt.Errorf("line %d: unclosed '{' in label set", lineNum)
		}
		labelContent := remainder[1:braceClose]
		rest = strings.TrimLeft(remainder[braceClose+1:], " \t")

		labels, err := parseLabels(labelContent, lineNum)
		if err != nil {
			return nil, err
		}

		valueStr, _ := splitFirstSpace(rest) // ignore optional timestamp
		if valueStr == "" {
			return nil, fmt.Errorf("line %d: missing value", lineNum)
		}
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid value %q: %w", lineNum, valueStr, err)
		}

		name := strings.TrimSpace(namepart)
		baseName := metricBaseName(name)
		return &Metric{
			Name:   name,
			Help:   state.helpByName[baseName],
			Type:   metricType(name, state),
			Labels: labels,
			Value:  value,
		}, nil
	}

	// No labels.
	namepart, rest = splitFirstSpace(line)
	valueStr, _ := splitFirstSpace(rest) // ignore optional timestamp
	if valueStr == "" {
		return nil, fmt.Errorf("line %d: missing value", lineNum)
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil, fmt.Errorf("line %d: invalid value %q: %w", lineNum, valueStr, err)
	}

	baseName := metricBaseName(namepart)
	return &Metric{
		Name:   namepart,
		Help:   state.helpByName[baseName],
		Type:   metricType(namepart, state),
		Labels: nil,
		Value:  value,
	}, nil
}

// parseLabels decodes the content between { and } (exclusive).
// It handles escaped characters inside quoted values: \\ → \, \" → ", \n → newline.
func parseLabels(s string, lineNum int) (map[string]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	labels := make(map[string]string)
	pos := 0
	n := len(s)

	for pos < n {
		// Skip leading whitespace / commas between pairs.
		for pos < n && (s[pos] == ' ' || s[pos] == '\t') {
			pos++
		}
		if pos >= n {
			break
		}

		// Read key.
		keyStart := pos
		for pos < n && s[pos] != '=' && s[pos] != ',' && s[pos] != '}' {
			pos++
		}
		key := strings.TrimSpace(s[keyStart:pos])
		if key == "" {
			return nil, fmt.Errorf("line %d: empty label name", lineNum)
		}

		if pos >= n || s[pos] != '=' {
			return nil, fmt.Errorf("line %d: expected '=' after label name %q", lineNum, key)
		}
		pos++ // consume '='

		if pos >= n || s[pos] != '"' {
			return nil, fmt.Errorf("line %d: expected '\"' after '=' for label %q", lineNum, key)
		}
		pos++ // consume opening '"'

		// Read value with escape handling.
		var sb strings.Builder
		closed := false
		for pos < n {
			ch := s[pos]
			if ch == '\\' {
				pos++
				if pos >= n {
					return nil, fmt.Errorf("line %d: unterminated escape in label %q value", lineNum, key)
				}
				switch s[pos] {
				case '\\':
					sb.WriteByte('\\')
				case '"':
					sb.WriteByte('"')
				case 'n':
					sb.WriteByte('\n')
				default:
					sb.WriteByte('\\')
					sb.WriteByte(s[pos])
				}
				pos++
			} else if ch == '"' {
				pos++ // consume closing '"'
				closed = true
				break
			} else {
				sb.WriteByte(ch)
				pos++
			}
		}
		if !closed {
			return nil, fmt.Errorf("line %d: unterminated string for label %q", lineNum, key)
		}

		labels[key] = sb.String()

		// Consume optional comma.
		for pos < n && (s[pos] == ' ' || s[pos] == '\t') {
			pos++
		}
		if pos < n && s[pos] == ',' {
			pos++
		}
	}

	return labels, nil
}

// unescapeHelp converts \\ to \ and \n to a real newline in HELP text.
func unescapeHelp(s string) string {
	if !strings.ContainsAny(s, `\`) {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\':
				sb.WriteByte('\\')
				i++
			case 'n':
				sb.WriteByte('\n')
				i++
			default:
				sb.WriteByte(s[i])
			}
		} else {
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}

// splitFirstSpace returns the first word and the remainder of s (after the
// separating space), trimmed of leading whitespace. If s has no space,
// remainder is "".
func splitFirstSpace(s string) (word, rest string) {
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimLeft(s[idx+1:], " \t")
}

// metricBaseName strips well-known Prometheus suffixes (_total, _sum, _count,
// _bucket) to look up the family's HELP and TYPE metadata.
func metricBaseName(name string) string {
	for _, suffix := range []string{"_total", "_sum", "_count", "_bucket", "_created"} {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

// metricType resolves the MetricType for a sample, falling back to Untyped.
func metricType(name string, state *parseState) MetricType {
	if t, ok := state.typeByName[name]; ok {
		return t
	}
	base := metricBaseName(name)
	if t, ok := state.typeByName[base]; ok {
		return t
	}
	return Untyped
}
