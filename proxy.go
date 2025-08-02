// proxy.go
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
)

// handleChatCompletions главный обработчик для проксирования запросов
func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, fmt.Sprintf("вместо запросов %s нужно использовать только POST-запросы", r.Method), http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "ОШИБКА: не удалось прочитать тело запроса", http.StatusInternalServerError)
		return
	}

	if err := r.Body.Close(); err != nil {
		log.Printf("ПРЕДУПРЕЖДЕНИЕ: не удалось закрыть тело исходного запроса: %v", err)
	}

	var reqData ChatCompletionRequest
	if err := json.Unmarshal(bodyBytes, &reqData); err != nil {
		http.Error(w, "не удалось распарсить тело JSON", http.StatusBadRequest)
		return
	}

	hasImages, modifiedMessages, err := processMessages(reqData.Messages)
	if err != nil {
		http.Error(w, fmt.Sprintf("не удалось обработать сообщения: %v", err), http.StatusBadRequest)
		return
	}
	reqData.Messages = modifiedMessages

	targetEndpoints, err := getTargetEndpoints(reqData.Model, hasImages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	modifiedBodyBytes, err := json.Marshal(reqData)
	if err != nil {
		http.Error(w, "не удалось преобразовать тело запроса", http.StatusInternalServerError)
		return
	}

	var lastErr error
	for _, endpoint := range targetEndpoints {
		targetURL := endpoint + "/v1/chat/completions"
		log.Printf("попытка проксирования в: %s модели %s", targetURL, reqData.Model)

		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewBuffer(modifiedBodyBytes))
		if err != nil {
			lastErr = fmt.Errorf("не удалось создать проксированный запрос для %s: %v", targetURL, err)
			log.Println(lastErr)
			continue
		}

		setProxyHeaders(proxyReq, reqData.Stream)

		client := &http.Client{}
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			lastErr = fmt.Errorf("запрос на %s не удался: %v", targetURL, err)
			log.Println(lastErr)
			continue
		}

		if proxyResp.StatusCode < 200 || proxyResp.StatusCode >= 300 {
			body, _ := io.ReadAll(proxyResp.Body)
			lastErr = fmt.Errorf("%s вернул неудачный %s-статус:\n%s", targetURL, proxyResp.Status, string(body))
			log.Println(lastErr)
			if err := proxyResp.Body.Close(); err != nil {
				log.Printf("ПРЕДУПРЕЖДЕНИЕ: не удалось закрыть тело ответа при ошибке: %v", err)
			}
			continue
		}

		log.Printf("успешное подключение к %s. стримминг ответа...", targetURL)
		copyHeaders(w.Header(), proxyResp.Header)
		w.WriteHeader(proxyResp.StatusCode)

		_, err = io.Copy(w, proxyResp.Body)
		if errClose := proxyResp.Body.Close(); errClose != nil {
			log.Printf("ПРЕДУПРЕЖДЕНИЕ: не удалось закрыть тело при копировании: %v", errClose)
		}
		if err != nil {
			log.Printf("ОШИБКА: неудалось доставить потоковый ответ клиенту %v", err)
		}
		return
	}

	log.Printf("запросы ко всем api были неудачны, последняя ошибка: %v", lastErr)
	http.Error(w, fmt.Sprintf("запросы ко всем api были неудачны, последняя ошибка: %v", lastErr), http.StatusBadGateway)
}

// processMessages проверяет наличие изображений и модифицирует сообщения
func processMessages(messages []Message) (hasImages bool, modifiedMessages []Message, err error) {
	modifiedMessages = make([]Message, len(messages))
	for i, msg := range messages {
		newMsg := Message{Role: msg.Role}

		switch content := msg.Content.(type) {
		case string:
			newMsg.Content = content
		case []interface{}:
			var parts []ContentPart
			for _, partData := range content {
				partMap, ok := partData.(map[string]interface{})
				if !ok {
					return false, nil, fmt.Errorf("недопустимый контент в составном сообщении: %T\nожидалось `image_url,omitempty`", content)
				}

				part := ContentPart{Type: partMap["type"].(string)}
				if part.Type == "text" {
					part.Text = partMap["text"].(string)
				} else if part.Type == "image_url" {
					hasImages = true
					imageURLMap := partMap["image_url"].(map[string]interface{})
					part.ImageURL = &ImageURLData{
						URL:    imageURLMap["url"].(string),
						Detail: "high", // "detail": "high" всегда
					}
				}
				parts = append(parts, part)
			}
			newMsg.Content = parts
		default:
			return false, nil, fmt.Errorf("неподдерживаемый тип контента сообщения: %T", msg.Content)
		}
		modifiedMessages[i] = newMsg
	}
	return hasImages, modifiedMessages, nil
}

// getTargetEndpoints выбирает список URL для проксирования
func getTargetEndpoints(modelName string, hasImages bool) ([]string, error) {
	// маппинг имен
	if mappedName, ok := modelNameMapping[modelName]; ok {
		modelName = mappedName
	}

	var config map[string]ModelEndpointConfig
	if hasImages {
		config = routingConfig.ImageSupport
	} else {
		config = routingConfig.NoImageSupport
	}

	if modelConfig, ok := config[modelName]; ok && len(modelConfig.Endpoints) > 0 {
		return modelConfig.Endpoints, nil
	}

	// поиск в другой категории как фоллбэк
	if hasImages {
		config = routingConfig.NoImageSupport
	} else {
		config = routingConfig.ImageSupport
	}
	if modelConfig, ok := config[modelName]; ok && len(modelConfig.Endpoints) > 0 {
		return modelConfig.Endpoints, nil
	}

	return nil, fmt.Errorf("в конфиге нет эндпоинтов связанных с modelName='%s' и hasImages=%t", modelName, hasImages)
}

// setProxyHeaders устанавливает заголовки для запроса к целевому API
func setProxyHeaders(req *http.Request, isStream bool) {
	req.Header.Set("Content-Type", "application/json")
	if isStream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("UserID", generateUserID(userIDLength))
	req.Header.Del("Connection")
}

// generateUserID создает случайную строку для заголовка
func generateUserID(length int) string {
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(userIDChars))))
		if err != nil {
			// логирование предупреждения однократно
			once.Do(func() {
				_, err := fmt.Fprintln(os.Stderr, "ВНИМАНИЕ: crypto/rand потерпел неудачу, и для генерации userid используется небезопасный запасной вариант math/rand.")
				if err != nil {
					return
				}
			})
			sb.WriteByte(userIDChars[mathrand.Intn(len(userIDChars))])
			continue
		}
		sb.WriteByte(userIDChars[num.Int64()])
	}
	return sb.String()
}

// copyHeaders копирует заголовки из ответа прокси в оригинальный ответ
func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
