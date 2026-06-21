// Package ai integrates an optional external LLM (Ollama, OpenAI or Anthropic)
// to extract a structured recipe from a web page. It is provider-agnostic: the
// admin configures provider, base URL, API key and model in the site settings.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Config is the runtime AI configuration (from the settings store).
type Config struct {
	Provider string // "openai" | "ollama" | "anthropic" ("" = disabled)
	BaseURL  string
	APIKey   string
	Model    string
}

// Enabled reports whether enough is configured to attempt a call.
func (c Config) Enabled() bool {
	return c.Provider != "" && c.Model != ""
}

// DraftIngredient mirrors the recipe form's ingredient row.
type DraftIngredient struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

// Draft is the structured recipe returned to prefill the form.
type Draft struct {
	Name        string            `json:"name"`
	BasePersons int               `json:"basePersons"`
	Ingredients []DraftIngredient `json:"ingredients"`
	Steps       []string          `json:"steps"`
}

const (
	fetchTimeout = 20 * time.Second
	aiTimeout    = 90 * time.Second
	maxPageChars = 14000
	maxBodyBytes = 3 << 20 // 3 MiB
)

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style|noscript|svg|head)[^>]*>.*?</\s*\1\s*>`)
	reComment     = regexp.MustCompile(`(?s)<!--.*?-->`)
	reTag         = regexp.MustCompile(`(?s)<[^>]+>`)
	reSpace       = regexp.MustCompile(`[ \t\x{00a0}]+`)
	reBlankLines  = regexp.MustCompile(`\n\s*\n\s*\n+`)
	reJSON        = regexp.MustCompile(`(?s)\{.*\}`)
)

// FetchAndClean downloads a page and reduces it to readable plain text, dropping
// scripts/styles/markup so the prompt stays small and signal-rich.
func FetchAndClean(ctx context.Context, url string) (string, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("URL invalide")
	}
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CampMenuBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("page inaccessible (HTTP %d)", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return "", err
	}
	text := string(raw)
	text = reScriptStyle.ReplaceAllString(text, " ")
	text = reComment.ReplaceAllString(text, " ")
	text = reTag.ReplaceAllString(text, "\n")
	text = html.UnescapeString(text)
	text = reSpace.ReplaceAllString(text, " ")
	// Trim each line and drop empties.
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	text = strings.Join(out, "\n")
	text = reBlankLines.ReplaceAllString(text, "\n\n")
	if len(text) > maxPageChars {
		text = text[:maxPageChars]
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("aucun contenu exploitable sur la page")
	}
	return text, nil
}

const systemPrompt = `Tu extrais des recettes de cuisine. Réponds UNIQUEMENT avec du JSON minifié valide, sans texte ni balises Markdown, correspondant exactement à ce schéma :
{"name":string,"basePersons":number,"ingredients":[{"name":string,"quantity":number,"unit":string}],"steps":[string]}
Règles : "basePersons" est le nombre de personnes de la recette (par défaut 4 si absent). "quantity" est un nombre pour ce nombre de personnes (0 si inconnu). "unit" est l'unité (g, ml, cuillère, pièce…) ou "" si aucune. "steps" est la liste ordonnée des étapes. Garde la langue d'origine de la recette. N'invente rien qui ne soit pas sur la page.`

// ExtractRecipe sends the cleaned page to the configured model and parses the
// JSON recipe it returns.
func ExtractRecipe(ctx context.Context, cfg Config, pageText string) (Draft, error) {
	ctx, cancel := context.WithTimeout(ctx, aiTimeout)
	defer cancel()
	userMsg := "Voici le contenu d'une page web. Extrais-en la recette :\n\n" + pageText

	var content string
	var err error
	if cfg.Provider == "anthropic" {
		content, err = callAnthropic(ctx, cfg, systemPrompt, userMsg)
	} else {
		content, err = callOpenAICompatible(ctx, cfg, systemPrompt, userMsg)
	}
	if err != nil {
		return Draft{}, err
	}
	return parseDraft(content)
}

func parseDraft(content string) (Draft, error) {
	m := reJSON.FindString(content)
	if m == "" {
		return Draft{}, fmt.Errorf("réponse de l'IA illisible")
	}
	var d Draft
	if err := json.Unmarshal([]byte(m), &d); err != nil {
		return Draft{}, fmt.Errorf("réponse de l'IA invalide : %w", err)
	}
	if d.BasePersons <= 0 {
		d.BasePersons = 4
	}
	return d, nil
}

// ---- OpenAI-compatible (OpenAI, Ollama, custom) ----

func callOpenAICompatible(ctx context.Context, cfg Config, system, user string) (string, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	endpoint := base
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	payload := map[string]any{
		"model":       cfg.Model,
		"temperature": 0.2,
		"stream":      false,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("erreur IA (HTTP %d): %s", resp.StatusCode, snippet(data))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("réponse IA illisible")
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("réponse IA vide")
	}
	return out.Choices[0].Message.Content, nil
}

// ---- Anthropic (Claude) ----

func callAnthropic(ctx context.Context, cfg Config, system, user string) (string, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	endpoint := base
	if !strings.HasSuffix(endpoint, "/v1/messages") {
		endpoint += "/v1/messages"
	}
	payload := map[string]any{
		"model":      cfg.Model,
		"max_tokens": 2000,
		"system":     system,
		"messages": []map[string]string{
			{"role": "user", "content": user},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("erreur IA (HTTP %d): %s", resp.StatusCode, snippet(data))
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("réponse IA illisible")
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("réponse IA vide")
	}
	return out.Content[0].Text, nil
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
