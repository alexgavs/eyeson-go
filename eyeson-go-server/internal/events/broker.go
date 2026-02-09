// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package events

import (
	"bufio"
	"encoding/json"
	"eyeson-go-server/internal/models"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Broker управляет SSE-подключениями клиентов и рассылает события.
type Broker struct {
	// Канал для широковещательной рассылки событий всем подключенным клиентам.
	Notifier chan []byte

	// Канал для регистрации новых клиентов.
	newClients chan chan []byte

	// Канал для отключения клиентов.
	closingClients chan chan []byte

	// Карта всех подключенных клиентов.
	clients map[chan []byte]bool
}

// NewBroker создает и запускает новый экземпляр брокера.
func NewBroker() (broker *Broker) {
	broker = &Broker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan (chan []byte)),
		closingClients: make(chan (chan []byte)),
		clients:        make(map[chan []byte]bool),
	}

	go broker.listen()

	return
}

// listen запускает главный цикл брокера для управления клиентами и событиями.
func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:
			// Новый клиент подключился.
			broker.clients[s] = true
			log.Printf("[SSE] Client added. %d registered clients", len(broker.clients))
		case s := <-broker.closingClients:
			// Клиент отключился.
			delete(broker.clients, s)
			log.Printf("[SSE] Client removed. %d registered clients", len(broker.clients))
		case event := <-broker.Notifier:
			// Рассылаем событие всем клиентам.
			for clientMessageChan := range broker.clients {
				clientMessageChan <- event
			}
		}
	}
}

// BroadcastEvent отправляет событие всем клиентам.
func (broker *Broker) BroadcastEvent(eventType string, data interface{}) {
	event := models.Event{
		Type: eventType,
		Data: data,
	}
	jsonData, err := json.Marshal(event.Data)
	if err != nil {
		log.Printf("[SSE] Error marshalling event data: %v", err)
		return
	}

	// Формируем сообщение в формате SSE
	sseMessage := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, jsonData))
	broker.Notifier <- sseMessage
}

// ServeHTTP обрабатывает HTTP-запросы для SSE-подключений.
func (broker *Broker) ServeHTTP(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Канал для отправки сообщений этому конкретному клиенту.
		messageChan := make(chan []byte)
		broker.newClients <- messageChan

		// Отписываемся при отключении.
		defer func() {
			broker.closingClients <- messageChan
		}()

		// Отправляем приветственное сообщение для подтверждения подключения.
		log.Println("[SSE] Sending initial connection event")
		event := models.Event{Type: "connection", Data: "ok"}
		jsonData, _ := json.Marshal(event.Data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, jsonData)
		w.Flush()

		// Ждем и отправляем события.
		for {
			select {
			case msg, open := <-messageChan:
				if !open {
					// Канал закрыт, клиент отключился.
					return
				}
				// Отправляем данные клиенту.
				_, err := w.Write(msg)
				if err != nil {
					// Ошибка записи, скорее всего клиент отключился.
					return
				}
				w.Flush()
			case <-time.After(30 * time.Second):
				// Отправляем keep-alive пинг каждые 30 секунд
				fmt.Fprintf(w, ": keep-alive\n\n")
				w.Flush()
			}
		}
	})

	return nil
}

// Глобальный экземпляр брокера для всего приложения.
var MainBroker *Broker
