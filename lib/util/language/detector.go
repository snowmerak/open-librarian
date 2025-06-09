package language

import (
	"strings"

	"github.com/pemistahl/lingua-go"
	"github.com/snowmerak/open-librarian/lib/util/logger"
)

// Detector provides language detection functionality
type Detector struct {
	detector lingua.LanguageDetector
}

// NewDetector creates a new language detector
func NewDetector() *Detector {
	languages := []lingua.Language{
		lingua.Korean,
		lingua.English,
		lingua.Japanese,
		lingua.Chinese,
		lingua.Spanish,
		lingua.French,
		lingua.German,
		lingua.Russian,
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		WithMinimumRelativeDistance(0.25).
		Build()

	return &Detector{
		detector: detector,
	}
}

// DetectLanguage detects the language of the given text
// Returns language code (ko, en, ja, zh, es, fr, de, ru) or "en" as default
// Always returns "en" (English) when language cannot be determined
func (d *Detector) DetectLanguage(text string) string {
	langLogger := logger.NewLogger("language-detection")
	langLogger.StartWithMsg("Detecting language of text")
	langLogger.Info().Int("text_length", len(text)).Msg("Language detection request")

	if text == "" {
		langLogger.Info().Str("result", "en").Msg("Empty text, using default language")
		langLogger.EndWithMsg("Language detection completed")
		return "en" // Default to English for empty text
	}

	// Clean text for analysis
	cleanText := strings.TrimSpace(text)
	if len(cleanText) < 10 {
		langLogger.Info().Str("result", "en").Msg("Short text, using default language")
		langLogger.EndWithMsg("Language detection completed")
		return "en" // Default to English for very short texts
	}

	// Try to detect language
	if language, exists := d.detector.DetectLanguageOf(cleanText); exists {
		detectedCode := d.mapLanguageToCode(language)
		// Ensure we return a valid language code, default to English if not supported
		if detectedCode != "" {
			langLogger.Info().Str("detected_language", detectedCode).Msg("Language detected successfully")
			langLogger.EndWithMsg("Language detection completed")
			return detectedCode
		}
	}

	// Always default to English if detection fails or returns unsupported language
	langLogger.Info().Str("result", "en").Msg("Detection failed, using default language")
	langLogger.EndWithMsg("Language detection completed")
	return "en"
}

// mapLanguageToCode converts lingua.Language to our language codes
// Returns empty string for unsupported languages (caller should default to "en")
func (d *Detector) mapLanguageToCode(lang lingua.Language) string {
	switch lang {
	case lingua.Korean:
		return "ko"
	case lingua.English:
		return "en"
	case lingua.Japanese:
		return "ja"
	case lingua.Chinese:
		return "zh"
	case lingua.Spanish:
		return "es"
	case lingua.French:
		return "fr"
	case lingua.German:
		return "de"
	case lingua.Russian:
		return "ru"
	default:
		return "" // Return empty string for unsupported languages
	}
}

// DetectLanguageWithConfidence detects language and returns confidence score
// Always returns "en" (English) with 0.0 confidence when language cannot be determined
func (d *Detector) DetectLanguageWithConfidence(text string) (string, float64) {
	if text == "" {
		return "en", 0.0 // Default to English for empty text
	}

	cleanText := strings.TrimSpace(text)
	if len(cleanText) < 10 {
		return "en", 0.0 // Default to English for very short texts
	}

	confidenceValues := d.detector.ComputeLanguageConfidenceValues(cleanText)
	if len(confidenceValues) == 0 {
		return "en", 0.0 // Default to English if no confidence values
	}

	// Get the most confident language
	mostConfident := confidenceValues[0]
	detectedCode := d.mapLanguageToCode(mostConfident.Language())

	// Ensure we return a valid language code, default to English if not supported
	if detectedCode != "" {
		return detectedCode, mostConfident.Value()
	}

	// Default to English if detected language is not supported
	return "en", 0.0
}

// GetSupportedLanguages returns list of supported language codes
func (d *Detector) GetSupportedLanguages() []string {
	return []string{"ko", "en", "ja", "zh", "es", "fr", "de", "ru"}
}

// ValidateLanguageCode checks if the given language code is supported
func (d *Detector) ValidateLanguageCode(lang string) bool {
	supported := d.GetSupportedLanguages()
	for _, l := range supported {
		if l == lang {
			return true
		}
	}
	return false
}
