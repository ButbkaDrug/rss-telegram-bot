package handler

import (
    tblib "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {}

func NewHandler() *Handler {
    return &Handler{}
}

func(h *Handler) TextMessageHandler(u tblib.Update){}
func(h *Handler) AudioMessageHandler(u tblib.Update){}
func(h *Handler) VideoMessageHandler(u tblib.Update){}
func(h *Handler) VoiceMessageHandler(u tblib.Update){}
