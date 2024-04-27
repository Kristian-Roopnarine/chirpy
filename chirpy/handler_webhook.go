package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Kristian-Roopnarine/chirpy/internal/auth"
	"github.com/Kristian-Roopnarine/chirpy/internal/database"
)

var UserUpgradedEvent = "user.upgraded"

func (cfg *apiConfig) handlerWebhook(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserId int `json:"user_id"`
		} `json:"data"`
	}
	apiKey, err := auth.GetApiKey(r.Header)
	if err != nil {
		if errors.Is(err, auth.ErrNoAuthHeaderIncluded) {
			respondWithError(w, http.StatusUnauthorized, "incorrect key")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if apiKey != cfg.polkaApiKey {
		respondWithError(w, http.StatusUnauthorized, "incorrect key")
		return
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode params")
		return
	}

	if params.Event != UserUpgradedEvent {
		respondWithJSON(w, http.StatusOK, "ok")
		return
	}

	err = cfg.DB.UpdateChirpyRedSubscription(params.Data.UserId, true)
	if err != nil {
		if errors.Is(err, database.ErrNotExist) {
			respondWithError(w, http.StatusNotFound, "user not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "error updating user subscription")
		return
	}
	respondWithJSON(w, http.StatusOK, "user upgraded")

}
