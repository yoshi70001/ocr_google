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

## Releases

Puedes encontrar los ejecutables pre-compilados para Windows, Linux y macOS en la [página de Releases de GitHub](https://github.com/yoshi70001/googleDocsOCR/releases).

## Uso

1.  Descarga el ejecutable para tu sistema operativo desde la [página de Releases](https://github.com/yoshi70001/googleDocsOCR/releases).
2.  Cree un proyecto en la [Consola de Google Cloud](https://console.cloud.google.com/) y habilite la API de Google Drive.
3.  Cree credenciales de OAuth 2.0 y descargue el archivo `credentials.json`.
4.  Coloque el archivo `credentials.json` en la misma carpeta que el ejecutable.
5.  (Opcional) Obtenga una clave de API de Gemini y establézcala como una variable de entorno llamada `GEMINI_API_KEY`.
6.  Cree una carpeta llamada `RGBImages` y coloque ahí las imágenes que desea procesar.
7.  Ejecute el programa. Por ejemplo, en Windows:
    ```bash
    googleDocsOCR-windows-amd64.exe
    ```
    Para usar la corrección con Gemini:
    ```bash
    googleDocsOCR-windows-amd64.exe -use-gemini
    ```
8.  El programa creará un archivo `subtitulo.srt` en la misma carpeta.

## Compilación

Si prefieres compilar el proyecto tú mismo, sigue estos pasos:

1.  Clona el repositorio.
2.  Asegúrate de tener Go instalado (versión 1.22 o superior).
3.  Para compilar el proyecto, puedes usar el script de compilación incluido. Este script generará los ejecutables en una carpeta `release`.
    ```bash
    go run build/build.go <version>
    ```
    Reemplaza `<version>` con la versión que deseas (por ejemplo, `1.0.0`).

## Limitaciones Conocidas

- **Calidad del OCR**: La calidad del texto extraído depende en gran medida de la calidad de la imagen de entrada. Las imágenes borrosas, con poca luz o con fuentes muy estilizadas pueden dar como resultado un texto incorrecto.
- **Limpieza de Texto**: El proceso de limpieza de texto es básico. Puede que no elimine todos los artefactos no deseados del OCR, especialmente en casos complejos.
- **Corrección con Gemini**: La corrección de texto con Gemini es potente pero no infalible. Puede malinterpretar el contexto o introducir errores. Además, depende de un formato de respuesta específico que podría cambiar en el futuro.

## Contribuciones

¡Las contribuciones son bienvenidas! Si deseas mejorar este proyecto, aquí hay algunas ideas:

- Mejorar el algoritmo de limpieza de texto.
- Hacer más robusto el análisis de la respuesta de Gemini (por ejemplo, solicitando una salida en formato JSON).
- Añadir más tests unitarios.
- Mejorar la documentación.

Siéntete libre de abrir un *issue* para discutir ideas o un *pull request* con tus mejoras.

## Licencia

Este proyecto está bajo la Licencia MIT.
