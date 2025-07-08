// geminifix/geminifix.go
package geminifix

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func NewClient(ctx context.Context) (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("la variable de entorno GEMINI_API_KEY no está configurada")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("error al crear el cliente de Gemini: %w", err)
	}
	return client, nil
}

// buildBatchAnimePrompt construye un prompt para corregir un lote de subtítulos.
func buildBatchAnimePrompt(texts []string) string {
	var numberedTexts strings.Builder
	for i, text := range texts {
		// Formateamos cada línea con un identificador único.
		numberedTexts.WriteString(fmt.Sprintf("LÍNEA %d: %s\n", i, text))
	}

	return fmt.Sprintf(`
Te proporcionaré un archivo de subtítulos en formato SRT que tiene texto mezclado en japonés y español, además de frases sin sentido o errores de transcripción.

Quiero que:

Conserves el formato SRT (número de línea, marcas de tiempo, texto).

Corrijas la gramática y ortografía del texto en español.

Para las partes en japonés no traducidas (o nombres japoneses), si no hay traducción disponible, déjalas tal cual.

Limpies cualquier texto suelto sin sentido, caracteres sobrantes o frases que no aportan nada (por ejemplo números aleatorios, palabras aisladas que no se entienden).

Mantengas la coherencia de estilo como si fueran subtítulos profesionales de anime, breves y naturales.

Responde con el archivo SRT corregido respetando el mismo orden de líneas y marcas de tiempo.

Dame solo la respuesta sin explicaciones ni nada mas.

%s
`, numberedTexts.String())
}

// CorrectTextBatch utiliza Gemini para corregir un lote de textos.
func CorrectTextBatch(ctx context.Context, client *genai.Client, batchToCorrect []string) ([]string, error) {
	if len(batchToCorrect) == 0 {
		return []string{}, nil
	}

	model := client.GenerativeModel("gemini-2.0-flash")
	prompt := buildBatchAnimePrompt(batchToCorrect)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("error al generar contenido con Gemini: %w", err)
	}

	// ... (código para extraer el texto de la respuesta de Gemini) ...
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini no devolvió candidatos")
	}
	rawResponse, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("la respuesta de Gemini no es de tipo texto")
	}

	// --- PARSEAR LA RESPUESTA DE GEMINI ---
	// Ahora debemos procesar la respuesta para extraer cada línea corregida.
	correctedBatch := make([]string, len(batchToCorrect))
	// fmt.Print(string(rawResponse))
	lines := strings.Split(string(rawResponse), "\n")

	parsedCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Buscamos el formato "LÍNEA <NÚMERO>: <TEXTO>"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue // Línea mal formateada, la ignoramos
		}

		var index int
		// Extraemos el número de "LÍNEA N"
		_, err := fmt.Sscanf(parts[0], "LÍNEA %d", &index)
		if err != nil {
			continue // No pudimos parsear el número, ignoramos la línea
		}

		// Verificamos que el índice esté dentro de los límites de nuestro slice
		if index >= 0 && index < len(correctedBatch) {
			correctedBatch[index] = strings.TrimSpace(parts[1])
			parsedCount++
		}
	}

	// Verificación de seguridad: si Gemini no devolvió todas las líneas, es un problema.
	if parsedCount != len(batchToCorrect) {
		log.Printf("[!] ADVERTENCIA: Gemini devolvió %d líneas, pero se esperaban %d. Puede haber subtítulos vacíos.", parsedCount, len(batchToCorrect))
		// Rellenamos las líneas que falten con el texto original para no perder subtítulos.
		for i, text := range correctedBatch {
			if text == "" {
				correctedBatch[i] = batchToCorrect[i]
				log.Printf("    - Rellenando línea faltante %d con el texto original.", i)
			}
		}
	}

	return correctedBatch, nil
}
