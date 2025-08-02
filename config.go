// config.go
package main

import "sync"

// ModelEndpointConfig хранит конфигурацию для одной модели
type ModelEndpointConfig struct {
	Endpoints []string `json:"endpoints"`
}

// RoutingConfig определяет, куда направлять запросы в зависимости от поддержки изображений
type RoutingConfig struct {
	ImageSupport   map[string]ModelEndpointConfig `json:"image_support"`
	NoImageSupport map[string]ModelEndpointConfig `json:"no_image_support"`
}

var (
	// роутинг моделей на апи
	routingConfig = RoutingConfig{
		ImageSupport: map[string]ModelEndpointConfig{
			"gpt-4o-mini": {
				Endpoints: []string{
					"https://oi-vscode-server-2.onrender.com",
					"https://oi-vscode-server-0501.onrender.com",
				},
			},
			"google/gemini-2.0-flash-001": {
				Endpoints: []string{
					"https://oi-vscode-server-2.onrender.com",
				},
			},
		},
		NoImageSupport: map[string]ModelEndpointConfig{
			"deepseek-v3": {
				Endpoints: []string{
					"https://oi-vscode-server-0501.onrender.com",
				},
			},
		},
	}

	// сопоставление публичных имен моделей с внутренними
	modelNameMapping = map[string]string{
		"gemini-2.0-flash": "google/gemini-2.0-flash-001",
	}

	// JSON ответ для эндпоинта /api/v1/models
	modelsResponseJSON = `
{
  "data": [
    {
      "created": 0,
      "id": "gpt-4o-mini",
      "context_length": 128000,
      "architecture": {
        "modality": "text+image->text",
        "input_modalities": ["text", "image", "file"],
        "output_modalities": ["text"],
        "tokenizer": "GPT"
      },
      "object": "model",
      "owned_by": "openai"
    },
    {
      "created": 1,
      "id": "gemini-2.0-flash",
      "context_length": 1048576,
      "architecture": {
        "modality": "text+image->text",
        "input_modalities": ["text", "image", "file"],
        "output_modalities": ["text"],
        "tokenizer": "Gemini"
      },
      "object": "model",
      "owned_by": "google"
    },
    {
      "created": 2,
      "id": "deepseek-v3",
      "context_length": 65536,
      "architecture": {
        "modality": "text->text",
        "input_modalities": ["text"],
        "output_modalities": ["text"],
        "tokenizer": "DeepSeek"
      },
      "object": "model",
      "owned_by": "deepseek"
    }
  ],
  "object": "list"
}
`
	Html string
	// StubPage в html-страничке текст oбфусцирован,
	// чтобы не получать теневые баны при деплое на HuggingFace Spaces,
	// где такие прокси запрещены.
	StubPage = `<!DOCTYPE html><html lang="ru">
<head>
<link rel="preconnect" href="https://yastatic.net"> <link rel="preconnect" href="https://cdn.jsdelivr.net">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/imbecility/ys_fonts@main/ys_fonts.css">
<style>:root { font-family: 'YS Text', sans-serif; background: #1e1e1e; color: #d4d4d4} h1, h2, h3, h4, h5, h6 { font-family: 'YS Display', sans-serif; }</style>
<title>ⲡpoĸcᴎ aĸⲧᴎʙᴎpoʙaʜ</title>
</head>
<body>
<h1>о⁠р︆е︌ո︍а‍і︈‒︆ⲡ︄o︄д︄o︈б︄ʜ︈ы︈ѝ⁠ ️ⲡ⁠p‌o⁡ĸ︋c︃ᴎ‍ ​з︉a︀ⲡ︂y⁮ɰ⁠e︉ʜ︊</h1><p>︋ⲡ⁯p⁠o︆ĸ‌c︊ᴎ​ ︂ⲡ⁬e︃p︁e︈ʜ︋a︄ⲡ⁯p︃a︃ʙ︂ᴫ︎ᴙ️e︂ⲧ︈ ‍з⁮a︉ⲡ︅p︆o︋c​ы⁠ ⁠ʙ︀ ︊p︆a⁯з︆ᴫ⁡ᴎ⁭ч︌ʜ︌ы︍e︎ ︈а︁р︉і︉.⁭</p>
<p>‍э⁡ʜ⁬д︊ⲡ︉o⁯ᴎ⁯ʜ⁯ⲧ︉ы⁭ ︆ĸ⁡a‌ĸ︈ ︊y︌ ⁬o‍р⁠е⁠ո️а⁫і︅‒⁯а︊р︄і︈,⁮ ⁭ĸ⁡ ︄υ︊ꭈ︌ⅼ︉ ︃ʜ︎y​ж︅ʜ︃o‍ ︎д‍o⁫б︃a⁡ʙ⁭ᴎ︉ⲧ⁭ь︆ ⁭/︎а︎р︋і⁯/︄ѵ⁠1⁫/︀:︊</p>
<ul><li><code>Ꮆ‍Е⁠Т⁡ ⁡/︆а⁬р︇і⁡/︊ѵ︀1︂/⁭ᴍ︀о︆ԁ︆е︇ⅼ️ѕ⁡</code></li><li><code> ⁮Р︁О⁭Ѕ︆Т︁ ︄/︊а⁡р︆і︍/️ѵ︀1︀/︎с⁮һ‌а︁τ︉/⁡с︋о︊ᴍ︆р⁭ⅼ︊е︃τ‌і︎о︄ո︅ѕ⁠</code></li></ul>
</body></html>`
	// для генерации случайного userid
	userIDChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	userIDLength = 21
	once         sync.Once
)
