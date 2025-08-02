// types.go
package main

// ChatCompletionRequest представляет тело запроса к /chat/completions
type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	// здесь можно добавить Temperature, TopP,
	// и другие параметры - они будут проксированы "как есть"
}

// Message представляет одно сообщение в диалоге
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // строка или срез ContentPart
}

// ContentPart представляет часть составного сообщения (текст или изображение)
type ContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *ImageURLData `json:"image_url,omitempty"`
}

// ImageURLData содержит URL изображения и уровень детализации
type ImageURLData struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}
