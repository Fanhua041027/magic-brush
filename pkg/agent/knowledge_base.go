package agent

import (
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

// KnowledgeBase manages document storage and TF-IDF search
type KnowledgeBase struct {
	mu        sync.RWMutex
	documents []KBDocument
	idCounter int
}

// NewKnowledgeBase creates a new knowledge base
func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		documents: make([]KBDocument, 0),
	}
}

// AddDocument adds a document to the knowledge base
func (kb *KnowledgeBase) AddDocument(title, content, source string) KBDocument {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	kb.idCounter++
	now := nowUnix()
	doc := KBDocument{
		ID:        kb.docID(kb.idCounter),
		Title:     title,
		Content:   content,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
	}
	kb.documents = append(kb.documents, doc)
	return doc
}

// DeleteDocument removes a document by ID
func (kb *KnowledgeBase) DeleteDocument(id string) bool {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	for i, d := range kb.documents {
		if d.ID == id {
			kb.documents = append(kb.documents[:i], kb.documents[i+1:]...)
			return true
		}
	}
	return false
}

// ListDocuments returns all documents
func (kb *KnowledgeBase) ListDocuments() []KBDocument {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	result := make([]KBDocument, len(kb.documents))
	copy(result, kb.documents)
	return result
}

// Search performs TF-IDF search against the knowledge base
func (kb *KnowledgeBase) Search(query string, maxResults int, minScore float64) []RetrievedPassage {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	if maxResults <= 0 {
		maxResults = 5
	}
	if minScore <= 0 {
		minScore = 0.1
	}

	if len(kb.documents) == 0 {
		return nil
	}

	keywords := tokenize(query)

	var scored []RetrievedPassage
	for _, doc := range kb.documents {
		docText := doc.Title + " " + doc.Content
		score := computeTFIDFScore(keywords, docText)
		if score >= minScore {
			scored = append(scored, RetrievedPassage{
				Source:  doc.Title,
				Content: extractSnippet(doc.Content, keywords),
				Score:   score,
			})
		}
	}

	// Sort by score descending (simple bubble sort for small sets)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if len(scored) > maxResults {
		scored = scored[:maxResults]
	}

	return scored
}

// SearchString returns a formatted string of search results
func (kb *KnowledgeBase) SearchString(query string, maxResults int) string {
	results := kb.Search(query, maxResults, 0.1)
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, r := range results {
		sb.WriteString("来源「")
		sb.WriteString(r.Source)
		sb.WriteString("」：")
		sb.WriteString(r.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (kb *KnowledgeBase) docID(n int) string {
	return "kb-" + itoa(n)
}

// ============================================================
// TF-IDF Search Implementation
// ============================================================

func tokenize(text string) []string {
	var tokens []string
	seen := make(map[string]bool)

	// Chinese characters (2-4 char substrings)
	re := regexp.MustCompile(`[\x{4e00}-\x{9fff}]+`)
	for _, match := range re.FindAllString(text, -1) {
		runes := []rune(match)
		for i := 0; i < len(runes)-1; i++ {
			if i < len(runes)-1 {
				t := string(runes[i : i+2])
				if !seen[t] {
					tokens = append(tokens, t)
					seen[t] = true
				}
			}
			if i < len(runes)-2 {
				t := string(runes[i : i+3])
				if !seen[t] {
					tokens = append(tokens, t)
					seen[t] = true
				}
			}
		}
	}

	// English words
	re2 := regexp.MustCompile(`[a-zA-Z_]\w*`)
	for _, match := range re2.FindAllString(text, -1) {
		lower := strings.ToLower(match)
		if !seen[lower] {
			tokens = append(tokens, lower)
			seen[lower] = true
		}
	}

	return tokens
}

func computeTFIDFScore(keywords []string, docText string) float64 {
	lowerText := strings.ToLower(docText)
	var score float64

	for _, kw := range keywords {
		lowerKW := strings.ToLower(kw)
		count := strings.Count(lowerText, lowerKW)
		if count > 0 {
			w := 1.0
			if len(kw) > 2 {
				w = 1.5
			}
			score += math.Log10(1+float64(count)) * w
		}
	}

	docLen := float64(len(docText))
	if docLen > 1 {
		score /= math.Log10(docLen)
	}

	return score
}

func extractSnippet(content string, keywords []string) string {
	sentences := strings.FieldsFunc(content, func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '\n' || r == '.' || r == '!' || r == '?'
	})

	var matching []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		lower := strings.ToLower(s)
		matched := false
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				matched = true
				break
			}
		}
		if matched {
			matching = append(matching, s)
			if len(matching) >= 3 {
				break
			}
		}
	}

	if len(matching) > 0 {
		return strings.Join(matching, "。") + "。"
	}

	// Fallback: first 200 chars
	runes := []rune(content)
	if len(runes) > 200 {
		return string(runes[:200])
	}
	return content
}

// ============================================================
// Utility functions
// ============================================================

// nowUnix returns current unix timestamp
func nowUnix() int64 {
	return time.Now().Unix()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
