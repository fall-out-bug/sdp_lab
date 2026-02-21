// intake-telegram is a Telegram bot for swarm task intake.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
)

func main() {
	token := flag.String("token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token")
	gatewayURL := flag.String("gateway", os.Getenv("INTAKE_GATEWAY_URL"), "Intake Gateway URL")
	addr := flag.String("addr", ":8082", "HTTP listen address")
	flag.Parse()

	if *token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN required")
	}
	if *gatewayURL == "" {
		*gatewayURL = "http://localhost:8081"
	}

	http.HandleFunc("/webhook/"+*token, func(w http.ResponseWriter, r *http.Request) {
		var update struct {
			Message struct {
				Text string `json:"text"`
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			return
		}
		text := update.Message.Text
		if len(text) > 6 && text[:6] == "/task " {
			// Parse: /task project title
			rest := text[6:]
			project := "default"
			title := rest
			for i, c := range rest {
				if c == ' ' {
					project = rest[:i]
					title = rest[i+1:]
					break
				}
			}
			req := map[string]any{
				"project_id": project,
				"title":      title,
				"source":     "telegram",
			}
			body, _ := json.Marshal(req)
			resp, err := http.Post(*gatewayURL+"/api/v1/intake", "application/json", bytes.NewReader(body))
			if err != nil {
				log.Printf("gateway: %v", err)
				return
			}
			defer resp.Body.Close()
			var result map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&result)
			// Send reply to user via Telegram API
			_ = result
		}
	})

	log.Printf("intake-telegram listening on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
