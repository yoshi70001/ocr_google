# Google Docs OCR

Este proyecto utiliza Google Drive y Google Docs como un motor de OCR (Reconocimiento Óptico de Caracteres) para extraer texto de imágenes. Opcionalmente, puede usar la API de Gemini para corregir el texto extraído y finalmente genera un archivo de subtítulos SRT.

## Características

- Se autentica con la API de Google Drive usando OAuth2.
- Crea una carpeta temporal en Google Drive para procesar las imágenes.
- Sube imágenes a Google Drive y las convierte a Google Docs para realizar el OCR.
- Descarga el texto extraído de los Google Docs.
- Limpia el texto extraído.
- (Opcional) Usa la API de Gemini para corregir errores gramaticales y ortográficos en el texto extraído.
- Genera un archivo de subtítulos SRT a partir de los archivos de texto procesados.

## Dependencias

- `golang.org/x/oauth2`
- `google.golang.org/api/drive/v3`
- `github.com/google/generative-ai-go/genai`

## Uso

1.  Cree un proyecto en la [Consola de Google Cloud](https://console.cloud.google.com/) y habilite la API de Google Drive.
2.  Cree credenciales de OAuth 2.0 y descargue el archivo `credentials.json`.
3.  Coloque el archivo `credentials.json` en la raíz del proyecto.
4.  Obtenga una clave de API de Gemini y establézcala como una variable de entorno llamada `GEMINI_API_KEY`.
5.  Coloque las imágenes que desea procesar en la carpeta `RGBImages`.
6.  Ejecute el programa:
    ```bash
    go run main.go
    ```
7.  El programa creará un archivo `subtitulo.srt` en la raíz del proyecto.

## Contribuciones

Las contribuciones son bienvenidas. Por favor, abra un issue o un pull request para discutir los cambios.

## Licencia

Este proyecto está bajo la Licencia MIT.
