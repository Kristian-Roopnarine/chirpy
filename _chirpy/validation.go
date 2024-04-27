package main

import (
	"slices"
	"strings"
)

func cleanInput(msg string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	splitMsg := strings.Split(msg, " ")
	for idx, word := range splitMsg {
		if slices.Contains(badWords, strings.ToLower(word)) {
			splitMsg[idx] = "****"
		}
	}
	return strings.Join(splitMsg, " ")
}
