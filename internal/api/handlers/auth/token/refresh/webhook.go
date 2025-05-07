package refresh

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

type Request struct {
	Message string `json:"message"`
}

var WebHookIP = os.Getenv("WEB_HOOK")

const DefMessage = "Попытка зайти с неизвенстного IP: "

func webHook(log *slog.Logger, IP string) {
	client := http.Client{}

	reqStruct := Request{Message: DefMessage + IP}

	jsonData, err := json.Marshal(reqStruct)
	if err != nil {
		log.Error("failed to marshal data", "error", err)
		return
	}

	req, err := http.NewRequest(
		http.MethodPost,
		WebHookIP,
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		log.Error("failed to make req", "error", err)
		return
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := client.Do(req)
	if err != nil {
		log.Error("failed to do req", "error", err)
		return
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode >= 400 {
		log.Error("webhook returned status code", "code", resp.StatusCode)
		return
	}

	log.Info("WebHook recived")

}
